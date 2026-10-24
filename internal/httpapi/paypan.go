package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/notify"
	"github.com/jhopan/Pan-Wallet/internal/paypan"
)

// paypanClient membangun client dari pengaturan yang tersimpan di database.
// Dibaca setiap request supaya perubahan token/URL di panel langsung berlaku
// tanpa restart, sama seperti pola PayPan.
func (server *Server) paypanClient(request *http.Request) (*paypan.Client, error) {
	if server.deps.Settings == nil {
		return nil, errors.New("pengaturan belum siap")
	}
	baseURL, err := server.deps.Settings.Setting(request.Context(), "paypan_base_url")
	if err != nil {
		return nil, err
	}
	token, err := server.deps.Settings.Setting(request.Context(), "paypan_token")
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimSpace(baseURL)
	token = strings.TrimSpace(token)

	if baseURL == "" {
		return nil, errors.New("PayPan base URL belum diisi di Konfigurasi")
	}
	if token == "" {
		return nil, errors.New("PayPan token belum diisi di Konfigurasi")
	}
	return paypan.NewClient(baseURL, token), nil
}

// paypanWebhook menerima order.paid dari PayPan, memverifikasi HMAC, lalu
// mengkredit saldo. Idempotent: kiriman ulang tidak menambah saldo lagi.
func (server *Server) paypanWebhook(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil || server.deps.Settings == nil {
		writeError(response, http.StatusServiceUnavailable, "belum siap")
		return
	}

	body, err := readLimitedBody(response, request, 1<<20)
	if err != nil {
		writeError(response, http.StatusBadRequest, "body tidak valid")
		return
	}

	secret, err := server.deps.Settings.Setting(request.Context(), "paypan_webhook_secret")
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		writeError(response, http.StatusServiceUnavailable, "webhook secret belum diisi")
		return
	}

	signature := request.Header.Get("X-Paypan-Signature")
	if !paypan.VerifyWebhook(body, signature, secret) {
		writeError(response, http.StatusForbidden, "signature tidak valid")
		return
	}

	event, err := paypan.ParseWebhook(body)
	if err != nil {
		writeError(response, http.StatusBadRequest, "payload webhook tidak valid")
		return
	}
	if event.Event != "order.paid" {
		writeJSON(response, http.StatusOK, map[string]any{"ok": true, "ignored": event.Event})
		return
	}
	if event.Order.ID == "" {
		writeError(response, http.StatusBadRequest, "id invoice kosong")
		return
	}

	invoice, err := server.deps.Topups.MarkTopupPaidByPaypan(request.Context(), event.Order.ID, "paypan-webhook")
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err.Error())
		return
	}
	// Kabari admin (Telegram / webhook) hanya pada pelunasan PERTAMA.
	// PayPan mengirim ulang webhook yang sama bila tidak menerima 200,
	// jadi tanpa penjagaan ini testimoni bisa terkirim berkali-kali.
	if invoice.FreshlyPaid && server.deps.Notify != nil {
		publicID := invoice.WalletUserID
		if server.deps.PublicIDs != nil {
			if resolved, err := server.deps.PublicIDs.PublicIDByWalletUserID(request.Context(), invoice.WalletUserID); err == nil {
				publicID = resolved
			}
		}
		event := notify.TopupEvent{
			PublicID:  publicID,
			Requested: invoice.Price,
			Amount:    invoice.Total,
			Balance:   invoice.BalanceAfter,
			PaidAt:    time.Now().Unix(),
			InvoiceID: invoice.ID,
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = server.deps.Notify.SendTopupPaid(ctx, event)
		}()
	}

	writeJSON(response, http.StatusOK, map[string]any{
		"ok": true, "invoice": invoice.ID, "credited": invoice.Total,
	})
}

// paypanReconcile menanyakan status invoice ke PayPan sebagai cadangan bila
// webhook tidak sampai. Dipakai tombol "cek status" di panel.
func (server *Server) paypanReconcile(response http.ResponseWriter, request *http.Request) {
	client, err := server.paypanClient(request)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	localID := request.PathValue("id")
	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), localID)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	if invoice.PaypanInvoice == "" {
		redirectWith(response, request, "/admin/topup", "err", "invoice tanpa referensi PayPan")
		return
	}

	remote, err := client.GetInvoice(request.Context(), invoice.PaypanInvoice)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}

	switch remote.Status {
	case "paid":
		updated, err := server.deps.Topups.MarkTopupPaidByPaypan(request.Context(), remote.ID, "paypan-reconcile")
		if err != nil {
			redirectWith(response, request, "/admin/topup", "err", err.Error())
			return
		}
		redirectWith(response, request, "/admin/topup", "ok", "Lunas terkonfirmasi, saldo +Rp"+Rupiah(updated.Total))
	case "expired", "refunded":
		if err := server.deps.Topups.ExpireTopupInvoice(request.Context(), localID); err != nil {
			redirectWith(response, request, "/admin/topup", "err", err.Error())
			return
		}
		redirectWith(response, request, "/admin/topup", "ok", "Invoice ditandai "+remote.Status)
	default:
		redirectWith(response, request, "/admin/topup", "ok", "Status PayPan: "+remote.Status)
	}
}

func readLimitedBody(response http.ResponseWriter, request *http.Request, limit int64) ([]byte, error) {
	reader := http.MaxBytesReader(response, request.Body, limit)
	return io.ReadAll(reader)
}

// paypanClientFrom membangun client dari pengaturan tanpa *http.Request.
// Dipakai rekonsiliasi latar belakang.
func (server *Server) paypanClientFrom(ctx context.Context) (*paypan.Client, error) {
	if server.deps.Settings == nil {
		return nil, errors.New("pengaturan belum siap")
	}
	baseURL, err := server.deps.Settings.Setting(ctx, "paypan_base_url")
	if err != nil {
		return nil, err
	}
	token, err := server.deps.Settings.Setting(ctx, "paypan_token")
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimSpace(baseURL)
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return nil, errors.New("PayPan belum dikonfigurasi")
	}
	return paypan.NewClient(baseURL, token), nil
}

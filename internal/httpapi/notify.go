package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jhopan/Pan-Wallet/internal/notify"
)

// notifySettings membaca konfigurasi notifikasi dari database (live, tanpa restart).
func (server *Server) notifySettings(ctx context.Context) (token, chats, webhook string) {
	if server.deps.Settings == nil {
		return "", "", ""
	}
	token, _ = server.deps.Settings.Setting(ctx, "notify_tg_token")
	chats, _ = server.deps.Settings.Setting(ctx, "notify_tg_chat")
	webhook, _ = server.deps.Settings.Setting(ctx, "notify_webhook_url")
	return strings.TrimSpace(token), strings.TrimSpace(chats), strings.TrimSpace(webhook)
}

// SendTopupPaid memberi tahu admin bahwa saldo masuk.
// Pesan sengaja dibuat layak ditempel ke grup testimoni: ada tanggal,
// jam, nominal, dan ID pelanggan.
func (server *Server) SendTopupPaid(ctx context.Context, event notify.TopupEvent) error {
	token, chats, webhook := server.notifySettings(ctx)
	topic, _ := server.deps.Settings.Setting(ctx, "notify_tg_topic")

	message := TopupMessage(event)
	target := notify.NewTelegram(token, chats, "")
	if strings.TrimSpace(topic) != "" {
		target = target.WithTopic(strings.TrimSpace(topic))
	}

	var firstErr error
	if target.Enabled() {
		if err := target.Send(ctx, message); err != nil {
			firstErr = err
		}
	}
	if webhook != "" {
		body := notify.WebhookBodyFrom(event.Payload())
		if err := notify.PostWebhook(ctx, webhook, body); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// TopupMessage menyusun teks notifikasi/testimoni.
func TopupMessage(event notify.TopupEvent) string {
	paidAt := event.PaidTime()

	// Yang dilaporkan sebagai "nominal top-up" adalah angka yang diminta
	// pelanggan (25.000). Total unik (25.004) tetap masuk saldo penuh,
	// dan ditampilkan terpisah supaya tidak ada yang merasa kurang.
	requested := event.Requested
	if requested == 0 {
		requested = event.Amount
	}

	message := fmt.Sprintf(
		"Top-up berhasil\n\n"+
			"ID Pelanggan: %s\n"+
			"Nominal: Rp %s\n",
		event.PublicID,
		Rupiah(requested),
	)
	if event.Amount != requested {
		message += fmt.Sprintf("Masuk saldo: Rp %s (termasuk kode unik)\n", Rupiah(event.Amount))
	}
	message += fmt.Sprintf(
		"Saldo sekarang: Rp %s\n"+
			"Waktu: %s",
		Rupiah(event.Balance),
		paidAt.Format("02 Jan 2006, 15:04 WIB"),
	)
	return message
}

// SendTest mengirim pesan uji ke kanal yang dikonfigurasi.
func (server *Server) SendTest(ctx context.Context) error {
	token, chats, webhook := server.notifySettings(ctx)
	if token == "" && webhook == "" {
		return errors.New("notifikasi belum dikonfigurasi")
	}

	var firstErr error
	if token != "" && chats != "" {
		if err := notify.NewTelegram(token, chats, "").Send(ctx, "Tes notifikasi Pan-Wallet berhasil."); err != nil {
			firstErr = err
		}
	}
	if webhook != "" {
		body := notify.WebhookBody("test", map[string]any{"message": "Tes notifikasi Pan-Wallet"})
		if err := notify.PostWebhook(ctx, webhook, body); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// adminNotifySave menyimpan konfigurasi notifikasi.
func (server *Server) adminNotifySave(response http.ResponseWriter, request *http.Request) {
	if server.deps.Settings == nil {
		redirectWith(response, request, "/admin/config", "err", "database belum siap")
		return
	}
	if err := request.ParseForm(); err != nil {
		redirectWith(response, request, "/admin/config", "err", "form tidak valid")
		return
	}

	fields := map[string]string{
		"notify_tg_token":    strings.TrimSpace(request.FormValue("notify_tg_token")),
		"notify_tg_chat":     strings.TrimSpace(request.FormValue("notify_tg_chat")),
		"notify_webhook_url": strings.TrimSpace(request.FormValue("notify_webhook_url")),
	}
	for key, value := range fields {
		if err := server.deps.Settings.SetSetting(request.Context(), key, value); err != nil {
			redirectWith(response, request, "/admin/config", "err", err.Error())
			return
		}
	}
	redirectWith(response, request, "/admin/config", "ok", "Notifikasi disimpan.")
}

// adminNotifyTest mengirim pesan uji.
func (server *Server) adminNotifyTest(response http.ResponseWriter, request *http.Request) {
	if server.deps.Notify == nil {
		redirectWith(response, request, "/admin/config", "err", "notifikasi belum siap")
		return
	}
	if err := server.deps.Notify.SendTest(request.Context()); err != nil {
		redirectWith(response, request, "/admin/config", "err", "Gagal: "+err.Error())
		return
	}
	redirectWith(response, request, "/admin/config", "ok", "Pesan uji terkirim.")
}

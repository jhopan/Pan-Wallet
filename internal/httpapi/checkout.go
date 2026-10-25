package httpapi

import (
	"errors"
	"net/http"

	"github.com/jhopan/PanWallet/internal/database"
	"github.com/jhopan/PanWallet/internal/sqlite"
)

func migrationsNames() []string {
	return []string{
		"000001_wallet_core",
		"000002_api_security",
		"000003_hold_transitions",
		"000004_qris_topup",
	}
}

// checkout mengarahkan pelanggan ke halaman bayar PayPan.
// PanWallet tidak merender QR sendiri: PayPan yang memegang gateway.
func (server *Server) checkout(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), request.PathValue("id"))
	if err != nil {
		http.NotFound(response, request)
		return
	}

	client, err := server.paypanClient(request)
	if err != nil {
		http.Error(response, err.Error(), http.StatusServiceUnavailable)
		return
	}
	remote, err := client.GetInvoice(request.Context(), invoice.PaypanInvoice)
	if err != nil {
		http.Error(response, "gagal membaca invoice PayPan: "+err.Error(), http.StatusBadGateway)
		return
	}
	target := client.BaseURL() + remote.PayURL
	http.Redirect(response, request, target, http.StatusFound)
}

// topupStatus adalah API ringan yang dipoll halaman checkout.
func (server *Server) topupStatus(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), request.PathValue("id"))
	if errors.Is(err, database.ErrInvoiceNotFound) || errors.Is(err, sqlite.ErrInvoiceNotFound) {
		writeError(response, http.StatusNotFound, "invoice tidak ditemukan")
		return
	}
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"id":     invoice.ID,
		"status": invoice.Status,
		"total":  invoice.Total,
	})
}

// statusLabelID menerjemahkan status invoice untuk tampilan panel.
func statusLabelID(status string) string {
	switch status {
	case "paid":
		return "lunas"
	case "pending":
		return "tertunda"
	case "expired":
		return "kedaluwarsa"
	case "cancelled":
		return "dibatalkan"
	default:
		return status
	}
}

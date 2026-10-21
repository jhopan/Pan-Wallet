package httpapi

import (
	"errors"
	"net/http"

	"github.com/jhopan/Pan-Wallet/internal/database"
	"github.com/jhopan/Pan-Wallet/internal/qris"
	"github.com/jhopan/Pan-Wallet/internal/topup"
)

func migrationsNames() []string {
	migrations := database.Migrations()
	names := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		names = append(names, migration.Name)
	}
	return names
}

// checkout merender halaman pembayaran QRIS untuk satu invoice.
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
	data := map[string]any{
		"ID":          invoice.ID,
		"Status":      invoice.Status,
		"StatusLabel": statusLabel(invoice.Status),
		"Total":       Rupiah(invoice.Total),
		"ShowQR":      invoice.Status == "pending" && server.qrisReady(request),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("checkout", response, data)
}

// checkoutQR mengirim gambar QR dinamis untuk invoice yang masih pending.
func (server *Server) checkoutQR(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil || server.deps.Settings == nil {
		http.Error(response, "belum siap", http.StatusServiceUnavailable)
		return
	}
	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), request.PathValue("id"))
	if err != nil {
		http.NotFound(response, request)
		return
	}
	if invoice.Status != "pending" {
		http.Error(response, "QR tidak tersedia", http.StatusNotFound)
		return
	}
	static, err := server.deps.Settings.Setting(request.Context(), "qris_base")
	if err != nil || static == "" {
		http.Error(response, "QRIS belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	payload, err := qris.BuildDynamic(static, invoice.Total)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	png, err := encodeQR(payload)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "image/png")
	response.Write(png)
}

// topupStatus adalah API ringan yang dipoll halaman checkout.
func (server *Server) topupStatus(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), request.PathValue("id"))
	if errors.Is(err, database.ErrInvoiceNotFound) {
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

func (server *Server) qrisReady(request *http.Request) bool {
	if server.deps.Settings == nil {
		return false
	}
	static, err := server.deps.Settings.Setting(request.Context(), "qris_base")
	return err == nil && static != ""
}

func statusLabel(status string) string {
	switch status {
	case "paid":
		return "LUNAS"
	case "expired":
		return "KADALUARSA"
	case "cancelled":
		return "DIBATALKAN"
	default:
		return "MENUNGGU PEMBAYARAN"
	}
}

var _ = topup.MinPrice

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

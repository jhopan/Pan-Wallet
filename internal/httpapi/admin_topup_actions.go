package httpapi

import (
	"net/http"
	"strings"
)

// adminTopupCancel membatalkan invoice top-up yang masih pending.
//
// Cancel di PayPan berarti invoice jadi "expired" dan kode unik kembali ke
// pool supaya tidak bentrok dengan invoice berikutnya. Saldo pelanggan tidak
// tersentuh karena invoice yang dibatalkan belum pernah dikredit.
func (server *Server) adminTopupCancel(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		redirectWith(response, request, "/admin/topup", "err", "database belum siap")
		return
	}

	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), request.PathValue("id"))
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	if invoice.PaypanInvoice == "" {
		redirectWith(response, request, "/admin/topup", "err", "invoice tanpa referensi PayPan")
		return
	}
	if invoice.Status != "pending" {
		redirectWith(response, request, "/admin/topup", "err",
			"hanya invoice pending yang bisa dibatalkan (sekarang: "+invoice.Status+")")
		return
	}

	client, err := server.paypanClient(request)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	if err := client.CancelInvoice(request.Context(), invoice.PaypanInvoice); err != nil {
		redirectWith(response, request, "/admin/topup", "err", "batal di PayPan gagal: "+err.Error())
		return
	}

	// Samakan status lokal. Kalau ini gagal, invoice tetap pending di
	// PanWallet padahal sudah dibatalkan di PayPan — jadi_error harus
	// terlihat, bukan diabaikan diam-diam.
	if err := server.deps.Topups.ExpireTopupInvoice(request.Context(), invoice.ID); err != nil {
		redirectWith(response, request, "/admin/topup", "err",
			"dibatalkan di PayPan tapi gagal update status lokal: "+err.Error())
		return
	}

	redirectWith(response, request, "/admin/topup", "ok", "Invoice "+invoice.ID+" dibatalkan.")
}

// adminTopupRefund menandai invoice yang sudah lunas sebagai refunded.
//
// PENTING: ini pembukuan administratif. Status invoice di PayPan menjadi
// "refunded", tapi PayPan TIDAK mengembalikan dana ke pelanggan.
// Pengembalian dana tetap harus dilakukan manual oleh admin.
func (server *Server) adminTopupRefund(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		redirectWith(response, request, "/admin/topup", "err", "database belum siap")
		return
	}

	invoice, err := server.deps.Topups.TopupInvoice(request.Context(), request.PathValue("id"))
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	if invoice.PaypanInvoice == "" {
		redirectWith(response, request, "/admin/topup", "err", "invoice tanpa referensi PayPan")
		return
	}
	if invoice.Status != "paid" {
		redirectWith(response, request, "/admin/topup", "err",
			"hanya invoice lunas yang bisa di-refund (sekarang: "+invoice.Status+")")
		return
	}

	client, err := server.paypanClient(request)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	if err := client.RefundInvoice(request.Context(), invoice.PaypanInvoice); err != nil {
		redirectWith(response, request, "/admin/topup", "err", "refund di PayPan gagal: "+err.Error())
		return
	}

	// Samakan status lokal supaya panel dan API tidak bilang "lunas"
	// padahal sudah direfund di PayPan.
	if err := server.deps.Topups.MarkTopupRefunded(request.Context(), invoice.ID); err != nil {
		redirectWith(response, request, "/admin/topup", "err",
			"direfund di PayPan tapi gagal update status lokal: "+err.Error())
		return
	}

	redirectWith(response, request, "/admin/topup", "ok",
		"Invoice "+invoice.ID+" ditandai refunded di PayPan. Dana tidak dikembalikan otomatis — selesaikan manual.")
}

// paypanStatusLabel memberi label status invoice yang enak dibaca.
func paypanStatusLabel(status string) string {
	switch strings.ToLower(status) {
	case "paid":
		return "Lunas"
	case "pending":
		return "Menunggu"
	case "expired":
		return "Kedaluwarsa"
	case "refunded":
		return "Refund"
	default:
		return status
	}
}

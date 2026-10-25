package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jhopan/PanWallet/internal/sqlite"
	"github.com/jhopan/PanWallet/internal/wallet"
)

// PublicIDStore mencari wallet lewat ID pendek yang dipakai pelanggan.
type PublicIDStore interface {
	ResolvePublicID(context.Context, string) (sqlite.WalletRow, error)
	PublicIDByWalletUserID(context.Context, string) (string, error)
}

// publicWallet melayani GET /v1/public/{id} — dipakai bot Telegram/WhatsApp
// untuk menampilkan saldo pelanggan dari ID pendek mereka.
// Tidak butuh login: ID-nya sendiri adalah rahasia (8 karakter acak).
func (server *Server) publicWallet(response http.ResponseWriter, request *http.Request) {
	if server.deps.PublicIDs == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	walletRow, err := server.deps.PublicIDs.ResolvePublicID(request.Context(), request.PathValue("id"))
	if err != nil {
		writeError(response, http.StatusNotFound, "ID tidak ditemukan")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"public_id": walletRow.PublicID,
		"balance":   walletRow.Balance,
		"held":      walletRow.HeldActive,
		"available": walletRow.Available,
		"currency":  walletRow.Currency,
		"status":    walletRow.Status,
	})
}

// publicTopup membuat invoice top-up dari ID pendek (dipakai bot).
func (server *Server) publicTopup(response http.ResponseWriter, request *http.Request) {
	if server.deps.PublicIDs == nil || server.deps.Topups == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	walletRow, err := server.deps.PublicIDs.ResolvePublicID(request.Context(), request.PathValue("id"))
	if err != nil {
		writeError(response, http.StatusNotFound, "ID tidak ditemukan")
		return
	}
	if walletRow.Status != "active" {
		writeError(response, http.StatusForbidden, "wallet diblokir")
		return
	}

	client, err := server.paypanClient(request)
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, err.Error())
		return
	}

	var body struct {
		Amount int64 `json:"amount"`
	}
	if err := decodeJSON(response, request, &body); err != nil || body.Amount <= 0 {
		writeError(response, http.StatusBadRequest, "amount wajib diisi")
		return
	}

	invoice, err := server.deps.Topups.CreateTopupInvoice(request.Context(), client, walletRow.WalletUserID, body.Amount)
	if err != nil {
		if errors.Is(err, wallet.ErrWalletBlocked) {
			writeError(response, http.StatusForbidden, "wallet diblokir")
			return
		}
		writeError(response, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(response, http.StatusCreated, map[string]any{
		"invoice_id": invoice.ID,
		"total":      invoice.Total,
		"status":     invoice.Status,
		"pay_url":    invoice.PayURL,
		"expires_at": invoice.ExpiresAt.Unix(),
	})
}

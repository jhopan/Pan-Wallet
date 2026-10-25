package httpapi

import (
	"errors"
	"net/http"

	"github.com/jhopan/PanWallet/internal/wallet"
)

// capture menyelesaikan hold: saldo dipotong dan ledger dicatat.
func (server *Server) capture(response http.ResponseWriter, request *http.Request) {
	server.transition(response, request, "capture")
}

// release membatalkan hold tanpa memotong saldo.
func (server *Server) release(response http.ResponseWriter, request *http.Request) {
	server.transition(response, request, "release")
}

func (server *Server) transition(response http.ResponseWriter, request *http.Request, action string) {
	if server.deps.Holds == nil {
		writeError(response, http.StatusServiceUnavailable, "wallet store belum siap")
		return
	}

	product, ok := productFromContext(request.Context())
	if !ok {
		writeError(response, http.StatusUnauthorized, "product API key tidak valid")
		return
	}

	transition := wallet.TransitionRequest{
		HoldID:         request.PathValue("id"),
		Product:        product,
		IdempotencyKey: request.Header.Get("Idempotency-Key"),
	}

	var (
		hold wallet.Hold
		err  error
	)
	if action == "capture" {
		hold, err = server.deps.Holds.CaptureHold(request.Context(), transition)
	} else {
		hold, err = server.deps.Holds.ReleaseHold(request.Context(), transition)
	}

	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrHoldNotFound):
			writeError(response, http.StatusNotFound, "hold tidak ditemukan untuk produk ini")
		case errors.Is(err, wallet.ErrHoldProcessed):
			writeError(response, http.StatusConflict, "hold sudah diproses")
		case errors.Is(err, wallet.ErrHoldExpired):
			writeError(response, http.StatusConflict, "hold sudah kedaluwarsa")
		case errors.Is(err, wallet.ErrIdempotencyConflict):
			writeError(response, http.StatusConflict, "idempotency key konflik")
		case errors.Is(err, wallet.ErrInsufficientBalance):
			writeError(response, http.StatusUnprocessableEntity, "saldo tidak cukup")
		default:
			writeError(response, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}

	writeJSON(response, http.StatusOK, map[string]any{
		"id":      hold.ID,
		"status":  hold.Status,
		"amount":  hold.Amount,
		"product": hold.Product,
	})
}

// walletBalance mengembalikan saldo tersimpan, tertahan, dan tersedia.
func (server *Server) walletBalance(response http.ResponseWriter, request *http.Request) {
	if server.deps.Holds == nil {
		writeError(response, http.StatusServiceUnavailable, "wallet store belum siap")
		return
	}
	balance, err := server.deps.Holds.WalletBalance(request.Context(), request.PathValue("id"))
	if err != nil {
		if errors.Is(err, wallet.ErrHoldNotFound) {
			writeError(response, http.StatusNotFound, "wallet tidak ditemukan")
			return
		}
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"wallet_user_id": balance.WalletUserID,
		"balance":        balance.Balance,
		"held":           balance.HeldActive,
		"available":      balance.Available,
		"currency":       balance.Currency,
	})
}

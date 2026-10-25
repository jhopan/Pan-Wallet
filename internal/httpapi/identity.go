package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jhopan/PanWallet/internal/identity"
	"github.com/jhopan/PanWallet/internal/sqlite"
)

// IdentityStore mengelola tautan identitas pelanggan.
type IdentityStore interface {
	ResolveIdentity(context.Context, string, string) (sqlite.WalletRow, bool, error)
	StartIdentityLink(context.Context, string, string) (string, error)
	ConfirmIdentityLink(context.Context, string, string) (sqlite.WalletRow, error)
	ListIdentities(context.Context, string) ([]string, error)
}

// identityResolve dipanggil bot setiap pelanggan chat.
// Telegram: wallet dibuat otomatis bila belum ada.
// Channel lain: hanya mengembalikan wallet yang sudah tertaut.
func (server *Server) identityResolve(response http.ResponseWriter, request *http.Request) {
	if server.deps.Identities == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	var body struct {
		Provider   string `json:"provider"`
		ExternalID string `json:"external_id"`
	}
	if err := decodeJSON(response, request, &body); err != nil {
		writeError(response, http.StatusBadRequest, "body tidak valid")
		return
	}

	walletRow, created, err := server.deps.Identities.ResolveIdentity(request.Context(), body.Provider, body.ExternalID)
	if err != nil {
		if errors.Is(err, sqlite.ErrInvoiceNotFound) {
			writeError(response, http.StatusNotFound, "belum terdaftar")
			return
		}
		writeError(response, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(response, http.StatusOK, map[string]any{
		"public_id": walletRow.PublicID,
		"balance":   walletRow.Balance,
		"held":      walletRow.HeldActive,
		"available": walletRow.Available,
		"currency":  walletRow.Currency,
		"status":    walletRow.Status,
		"is_new":    created,
	})
}

// identityLinkStart menerbitkan kode untuk mengikat channel baru.
// Dipakai website: pelanggan masukkan email, dapat kode, kirim ke bot Telegram.
func (server *Server) identityLinkStart(response http.ResponseWriter, request *http.Request) {
	if server.deps.Identities == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	var body struct {
		Provider   string `json:"provider"`
		ExternalID string `json:"external_id"`
	}
	if err := decodeJSON(response, request, &body); err != nil {
		writeError(response, http.StatusBadRequest, "body tidak valid")
		return
	}

	code, err := server.deps.Identities.StartIdentityLink(request.Context(), body.Provider, body.ExternalID)
	if err != nil {
		if errors.Is(err, sqlite.ErrIdentityTaken) {
			writeError(response, http.StatusConflict, "identitas sudah tertaut ke akun lain")
			return
		}
		writeError(response, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(response, http.StatusCreated, map[string]any{
		"code":        code,
		"instruction": identity.LinkInstruction(code),
		"expires_in":  600,
	})
}

// identityLinkConfirm dipanggil bot Telegram saat pelanggan mengirim /ikat <kode>.
func (server *Server) identityLinkConfirm(response http.ResponseWriter, request *http.Request) {
	if server.deps.Identities == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	var body struct {
		Code       string `json:"code"`
		TelegramID string `json:"telegram_id"`
	}
	if err := decodeJSON(response, request, &body); err != nil {
		writeError(response, http.StatusBadRequest, "body tidak valid")
		return
	}
	if strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.TelegramID) == "" {
		writeError(response, http.StatusBadRequest, "code dan telegram_id wajib diisi")
		return
	}

	walletRow, err := server.deps.Identities.ConfirmIdentityLink(request.Context(), body.Code, body.TelegramID)
	if err != nil {
		switch {
		case errors.Is(err, sqlite.ErrCodeInvalid):
			writeError(response, http.StatusUnprocessableEntity, "kode tidak valid atau sudah kedaluwarsa")
		case errors.Is(err, sqlite.ErrIdentityTaken):
			writeError(response, http.StatusConflict, "identitas sudah tertaut ke akun lain")
		default:
			writeError(response, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}

	writeJSON(response, http.StatusOK, map[string]any{
		"public_id": walletRow.PublicID,
		"linked":    true,
	})
}

// identityList menampilkan channel yang tertaut ke satu ID pelanggan.
func (server *Server) identityList(response http.ResponseWriter, request *http.Request) {
	if server.deps.Identities == nil || server.deps.PublicIDs == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	walletRow, err := server.deps.PublicIDs.ResolvePublicID(request.Context(), request.PathValue("id"))
	if err != nil {
		writeError(response, http.StatusNotFound, "ID tidak ditemukan")
		return
	}
	links, err := server.deps.Identities.ListIdentities(request.Context(), walletRow.WalletUserID)
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"public_id": walletRow.PublicID,
		"links":     links,
	})
}

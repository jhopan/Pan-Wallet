package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

// WalletRowView adalah baris wallet untuk tampilan panel.
type WalletRowView struct {
	WalletUserID string
	PublicID     string
	StatusLabel  string
	StatusClass  string
	Balance      Money
	HeldActive   Money
	Available    Money
	CreatedAt    string
}

// adminWallets menampilkan daftar wallet user.
func (server *Server) adminWallets(response http.ResponseWriter, request *http.Request) {
	if server.deps.Wallets == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	wallets, err := server.deps.Wallets.ListWallets(request.Context(), 200)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([]WalletRowView, 0, len(wallets))
	for _, wallet := range wallets {
		label, class := "aktif", "paid"
		if wallet.Status == "blocked" {
			label, class = "diblokir", "expired"
		}
		rows = append(rows, WalletRowView{
			WalletUserID: wallet.WalletUserID,
			PublicID:     wallet.PublicID,
			StatusLabel:  label,
			StatusClass:  class,
			Balance:      Money(wallet.Balance),
			HeldActive:   Money(wallet.HeldActive),
			Available:    Money(wallet.Available),
			CreatedAt:    wallet.CreatedAt.Format("02 Jan 15:04"),
		})
	}

	data := map[string]any{
		"Title":   "Wallet User",
		"Tab":     "wallets",
		"Wallets": rows,
		"Flash":   request.URL.Query().Get("ok"),
		"Error":   request.URL.Query().Get("err"),
		"User":    server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("wallets", response, data)
}

// adminWalletsCreate membuat wallet user baru dengan saldo awal opsional.
func (server *Server) adminWalletsCreate(response http.ResponseWriter, request *http.Request) {
	if server.deps.Wallets == nil {
		redirectWith(response, request, "/admin/wallets", "err", "database belum siap")
		return
	}
	if err := request.ParseForm(); err != nil {
		redirectWith(response, request, "/admin/wallets", "err", "form tidak valid")
		return
	}

	var initial int64
	if raw := strings.TrimSpace(request.FormValue("initial_balance")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			redirectWith(response, request, "/admin/wallets", "err", "saldo awal tidak valid")
			return
		}
		initial = parsed
	}

	if _, err := server.deps.Wallets.CreateWalletUser(request.Context(), initial); err != nil {
		redirectWith(response, request, "/admin/wallets", "err", err.Error())
		return
	}
	redirectWith(response, request, "/admin/wallets", "ok", "Wallet dibuat.")
}

// adminWalletsStatus memblokir atau mengaktifkan kembali wallet user.
func (server *Server) adminWalletsStatus(response http.ResponseWriter, request *http.Request) {
	if server.deps.Wallets == nil {
		redirectWith(response, request, "/admin/wallets", "err", "database belum siap")
		return
	}
	status := request.FormValue("status")
	if status == "" {
		status = "blocked"
	}
	walletUserID := request.PathValue("id")
	if err := server.deps.Wallets.SetWalletStatus(request.Context(), walletUserID, status); err != nil {
		redirectWith(response, request, "/admin/wallets", "err", err.Error())
		return
	}
	label := "diaktifkan"
	if status == "blocked" {
		label = "diblokir"
	}
	redirectWith(response, request, "/admin/wallets", "ok", "Wallet "+label+".")
}

package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jhopan/PanWallet/internal/sqlite"
)

// portalLogin menampilkan form masuk pelanggan (email).
// Halaman ini publik dan tidak membocorkan data apa pun sebelum terverifikasi.
func (server *Server) portalLogin(response http.ResponseWriter, request *http.Request) {
	data := map[string]any{
		"Title": "Masuk",
		"Step":  "email",
		"Error": request.URL.Query().Get("err"),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("portal", response, data)
}

// portalStart menerbitkan kode ikat untuk email, lalu menampilkan instruksi.
func (server *Server) portalStart(response http.ResponseWriter, request *http.Request) {
	if server.deps.Identities == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	if err := request.ParseForm(); err != nil {
		http.Redirect(response, request, "/masuk?err=form+tidak+valid", http.StatusFound)
		return
	}
	email := strings.TrimSpace(request.FormValue("email"))
	if email == "" {
		http.Redirect(response, request, "/masuk?err=email+wajib+diisi", http.StatusFound)
		return
	}

	// Sudah tertaut? Langsung tampilkan saldo.
	if walletRow, _, err := server.deps.Identities.ResolveIdentity(request.Context(), "email", email); err == nil {
		server.portalShowWallet(response, request, walletRow)
		return
	}

	code, err := server.deps.Identities.StartIdentityLink(request.Context(), "email", email)
	if err != nil {
		if errors.Is(err, sqlite.ErrIdentityTaken) {
			http.Redirect(response, request, "/masuk?err=email+sudah+tertaut+ke+akun+lain", http.StatusFound)
			return
		}
		http.Redirect(response, request, "/masuk?err="+urlQueryEscape(err.Error()), http.StatusFound)
		return
	}

	data := map[string]any{
		"Title": "Ikat akun",
		"Step":  "code",
		"Email": email,
		"Code":  code,
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("portal", response, data)
}

// portalCheck dipanggil tombol "Saya sudah kirim" setelah pelanggan mengirim
// /ikat ke Telegram. Kalau berhasil, saldo ditampilkan.
func (server *Server) portalCheck(response http.ResponseWriter, request *http.Request) {
	if server.deps.Identities == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	if err := request.ParseForm(); err != nil {
		http.Redirect(response, request, "/masuk?err=form+tidak+valid", http.StatusFound)
		return
	}
	email := strings.TrimSpace(request.FormValue("email"))
	if email == "" {
		http.Redirect(response, request, "/masuk?err=email+wajib+diisi", http.StatusFound)
		return
	}

	walletRow, _, err := server.deps.Identities.ResolveIdentity(request.Context(), "email", email)
	if err != nil {
		http.Redirect(response, request, "/masuk?err=belum+tertaut,+kirim+kode+ke+bot+Telegram", http.StatusFound)
		return
	}
	server.portalShowWallet(response, request, walletRow)
}

// portalShowWallet menampilkan saldo dan riwayat untuk satu wallet.
func (server *Server) portalShowWallet(response http.ResponseWriter, request *http.Request, walletRow sqlite.WalletRow) {
	entries := []LedgerEntry{}
	if server.deps.Ledger != nil {
		if raw, err := server.deps.Ledger.LedgerEntries(request.Context(), walletRow.WalletUserID, 20); err == nil {
			for _, entry := range raw {
				amount := entry.Amount
				if amount < 0 {
					amount = -amount
				}
				entries = append(entries, LedgerEntry{
					EntryType:    entryTypeLabel(entry.EntryType),
					Amount:       Money(amount),
					BalanceAfter: Money(entry.BalanceAfter),
					Product:      entry.Product,
					ReferenceID:  entry.ReferenceID,
					CreatedAt:    entry.CreatedAt.Format("02 Jan 15:04"),
					IsCredit:     entry.Amount > 0,
				})
			}
		}
	}

	links := []string{}
	if server.deps.Identities != nil {
		if raw, err := server.deps.Identities.ListIdentities(request.Context(), walletRow.WalletUserID); err == nil {
			links = raw
		}
	}

	data := map[string]any{
		"Title":     "Saldo Saya",
		"Step":      "wallet",
		"PublicID":  walletRow.PublicID,
		"Balance":   Money(walletRow.Balance),
		"Held":      Money(walletRow.HeldActive),
		"Available": Money(walletRow.Available),
		"Status":    walletRow.Status,
		"Entries":   entries,
		"Links":     links,
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("portal", response, data)
}

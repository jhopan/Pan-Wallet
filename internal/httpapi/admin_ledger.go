package httpapi

import (
	"net/http"
	"strings"
)

// LedgerEntry adalah baris mutasi untuk tampilan panel.
type LedgerEntry struct {
	EntryType    string
	Amount       Money
	BalanceAfter Money
	Product      string
	ReferenceID  string
	CreatedAt    string
	IsCredit     bool
}

// ProductKey adalah baris API key produk untuk tampilan panel.
type ProductKey struct {
	Product     string
	TokenPrefix string
	Active      bool
	StatusLabel string
	RotatedAt   string
}

// adminLedger menampilkan riwayat mutasi saldo.
func (server *Server) adminLedger(response http.ResponseWriter, request *http.Request) {
	if server.deps.Ledger == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	walletUserID := strings.TrimSpace(request.URL.Query().Get("wallet"))
	if walletUserID == "" {
		if server.deps.Overview != nil {
			if overview, err := server.deps.Overview.Overview(request.Context()); err == nil && len(overview.Wallets) > 0 {
				walletUserID = overview.Wallets[0].WalletUserID
			}
		}
	}

	var rows []LedgerEntry
	if walletUserID != "" {
		entries, err := server.deps.Ledger.LedgerEntries(request.Context(), walletUserID, 100)
		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, entry := range entries {
			amount := entry.Amount
			if amount < 0 {
				amount = -amount
			}
			rows = append(rows, LedgerEntry{
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

	data := map[string]any{
		"Title":        "Mutasi Saldo",
		"Tab":          "ledger",
		"WalletUserID": walletUserID,
		"Entries":      rows,
		"Flash":        request.URL.Query().Get("ok"),
		"Error":        request.URL.Query().Get("err"),
		"User":         server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("ledger", response, data)
}

func entryTypeLabel(entryType string) string {
	switch entryType {
	case "topup":
		return "Top-up"
	case "hold_capture":
		return "Potong order"
	case "refund":
		return "Refund"
	case "adjustment":
		return "Koreksi"
	default:
		return entryType
	}
}

// adminKeys menampilkan dan mengelola API key produk.
func (server *Server) adminKeys(response http.ResponseWriter, request *http.Request) {
	if server.deps.Keys == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	keys, err := server.deps.Keys.ListProductKeys(request.Context())
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([]ProductKey, 0, len(keys))
	for _, key := range keys {
		status := "nonaktif"
		if key.Active {
			status = "aktif"
		}
		rows = append(rows, ProductKey{
			Product:     key.Product,
			TokenPrefix: key.TokenPrefix,
			Active:      key.Active,
			StatusLabel: status,
			RotatedAt:   key.RotatedAt.Format("02 Jan 15:04"),
		})
	}

	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + request.Host

	data := map[string]any{
		"Title":    "API Key Produk",
		"Tab":      "keys",
		"BaseURL":  baseURL,
		"Keys":     rows,
		"NewToken": request.URL.Query().Get("token"),
		"Flash":    request.URL.Query().Get("ok"),
		"Error":    request.URL.Query().Get("err"),
		"User":     server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("keys", response, data)
}

// adminKeysCreate membuat atau merotasi key produk, lalu menampilkan token SEKALI.
func (server *Server) adminKeysCreate(response http.ResponseWriter, request *http.Request) {
	if server.deps.Keys == nil {
		redirectWith(response, request, "/admin/keys", "err", "database belum siap")
		return
	}
	if err := request.ParseForm(); err != nil {
		redirectWith(response, request, "/admin/keys", "err", "form tidak valid")
		return
	}
	product := strings.TrimSpace(request.FormValue("product"))
	if product == "" {
		redirectWith(response, request, "/admin/keys", "err", "nama produk wajib diisi")
		return
	}
	token, err := server.deps.Keys.CreateProductKey(request.Context(), product)
	if err != nil {
		redirectWith(response, request, "/admin/keys", "err", err.Error())
		return
	}
	http.Redirect(response, request,
		"/admin/keys?ok="+urlQueryEscape("Key untuk "+product+" dibuat. Salin sekarang, tidak akan ditampilkan lagi.")+
			"&token="+urlQueryEscape(token), http.StatusFound)
}

// adminKeysRevoke menonaktifkan key produk.
func (server *Server) adminKeysRevoke(response http.ResponseWriter, request *http.Request) {
	if server.deps.Keys == nil {
		redirectWith(response, request, "/admin/keys", "err", "database belum siap")
		return
	}
	product := request.PathValue("product")
	if err := server.deps.Keys.RevokeProductKey(request.Context(), product); err != nil {
		redirectWith(response, request, "/admin/keys", "err", err.Error())
		return
	}
	redirectWith(response, request, "/admin/keys", "ok", "Key produk "+product+" dinonaktifkan.")
}

package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// adminUser mengembalikan nama admin untuk ditampilkan di header panel.
func (server *Server) adminUser() string {
	if server.deps.Admin == nil {
		return ""
	}
	return server.deps.Admin.Username()
}

// adminRow adalah bentuk invoice untuk ditampilkan di panel.
type adminRow struct {
	ID           string
	WalletUserID string
	Price        Money
	Code         int
	Total        Money
	Status       string
	StatusLabel  string
	CreatedAt    string
}

// adminWallet adalah baris saldo terformat untuk panel.
type adminWallet struct {
	WalletUserID string
	Balance      Money
	HeldActive   Money
	Available    Money
}

func (server *Server) adminHome(response http.ResponseWriter, request *http.Request) {
	if server.deps.Overview == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	overview, err := server.deps.Overview.Overview(request.Context())
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	wallets := make([]adminWallet, 0, len(overview.Wallets))
	for _, wallet := range overview.Wallets {
		wallets = append(wallets, adminWallet{
			WalletUserID: wallet.WalletUserID,
			Balance:      Money(wallet.Balance),
			HeldActive:   Money(wallet.HeldActive),
			Available:    Money(wallet.Available),
		})
	}

	invoices, pending := server.recentInvoices(request, 10)
	data := map[string]any{
		"Title":        "Dashboard",
		"Tab":          "home",
		"Overview":     overview,
		"Wallets":      wallets,
		"Invoices":     invoices,
		"PendingCount": pending,
		"Flash":        request.URL.Query().Get("ok"),
		"User":         server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("home", response, data)
}

func (server *Server) adminTopup(response http.ResponseWriter, request *http.Request) {
	invoices, _ := server.recentInvoices(request, 50)
	data := map[string]any{
		"Title":    "Top-up QRIS",
		"Tab":      "topup",
		"Invoices": invoices,
		"Flash":    request.URL.Query().Get("ok"),
		"Error":    request.URL.Query().Get("err"),
		"User":     server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("topup", response, data)
}

func (server *Server) adminTopupCreate(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		redirectWith(response, request, "/admin/topup", "err", "database belum siap")
		return
	}
	if err := request.ParseForm(); err != nil {
		redirectWith(response, request, "/admin/topup", "err", "form tidak valid")
		return
	}
	walletUserID := request.FormValue("wallet_user_id")
	price, err := strconv.ParseInt(request.FormValue("price"), 10, 64)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", "nominal tidak valid")
		return
	}
	ttl := server.invoiceTTL(request)
	invoice, err := server.deps.Topups.CreateTopupInvoice(request.Context(), walletUserID, price, ttl)
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	redirectWith(response, request, "/admin/topup", "ok", "Invoice "+invoice.ID[:8]+" dibuat, total Rp"+Rupiah(invoice.Total))
}

func (server *Server) adminTopupPaid(response http.ResponseWriter, request *http.Request) {
	if server.deps.Topups == nil {
		redirectWith(response, request, "/admin/topup", "err", "database belum siap")
		return
	}
	invoiceID := request.PathValue("id")
	invoice, err := server.deps.Topups.MarkTopupPaid(request.Context(), invoiceID, "manual-admin")
	if err != nil {
		redirectWith(response, request, "/admin/topup", "err", err.Error())
		return
	}
	redirectWith(response, request, "/admin/topup", "ok", "Invoice ditandai lunas, saldo +Rp"+Rupiah(invoice.Total))
}

func (server *Server) adminConfig(response http.ResponseWriter, request *http.Request) {
	qrisBase := ""
	ttlMinutes := 15
	engine := "PostgreSQL"
	if server.deps.Settings != nil {
		if value, err := server.deps.Settings.Setting(request.Context(), "qris_base"); err == nil {
			qrisBase = value
		}
		if value, err := server.deps.Settings.Setting(request.Context(), "invoice_ttl_minutes"); err == nil && value != "" {
			if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
				ttlMinutes = parsed
			}
		}
	}
	qrisStatus := "belum diset"
	if qrisBase != "" {
		qrisStatus = "tersimpan"
	}
	data := map[string]any{
		"Title":      "Konfigurasi",
		"Tab":        "config",
		"QrisBase":   qrisBase,
		"TTLMinutes": ttlMinutes,
		"QrisStatus": qrisStatus,
		"Engine":     engine,
		"Migrations": len(migrationsNames()),
		"Flash":      request.URL.Query().Get("ok"),
		"Error":      request.URL.Query().Get("err"),
		"User":       server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("config", response, data)
}

func (server *Server) adminConfigSave(response http.ResponseWriter, request *http.Request) {
	if server.deps.Settings == nil {
		redirectWith(response, request, "/admin/config", "err", "database belum siap")
		return
	}
	if err := request.ParseForm(); err != nil {
		redirectWith(response, request, "/admin/config", "err", "form tidak valid")
		return
	}
	qrisBase := request.FormValue("qris_base")
	if qrisBase != "" {
		if err := server.deps.Settings.SetSetting(request.Context(), "qris_base", qrisBase); err != nil {
			redirectWith(response, request, "/admin/config", "err", err.Error())
			return
		}
	}
	if ttl := request.FormValue("invoice_ttl_minutes"); ttl != "" {
		if _, err := strconv.Atoi(ttl); err != nil {
			redirectWith(response, request, "/admin/config", "err", "masa berlaku tidak valid")
			return
		}
		if err := server.deps.Settings.SetSetting(request.Context(), "invoice_ttl_minutes", ttl); err != nil {
			redirectWith(response, request, "/admin/config", "err", err.Error())
			return
		}
	}
	redirectWith(response, request, "/admin/config", "ok", "Konfigurasi disimpan.")
}

// invoiceTTL membaca masa berlaku invoice dari settings, default 15 menit.
func (server *Server) invoiceTTL(request *http.Request) time.Duration {
	minutes := 15
	if server.deps.Settings != nil {
		if value, err := server.deps.Settings.Setting(request.Context(), "invoice_ttl_minutes"); err == nil && value != "" {
			if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 120 {
				minutes = parsed
			}
		}
	}
	return time.Duration(minutes) * time.Minute
}

func (server *Server) recentInvoices(request *http.Request, limit int) ([]adminRow, int) {
	if server.deps.Topups == nil {
		return nil, 0
	}
	invoices, err := server.deps.Topups.RecentTopupInvoices(request.Context(), limit)
	if err != nil {
		return nil, 0
	}
	rows := make([]adminRow, 0, len(invoices))
	pending := 0
	for _, invoice := range invoices {
		if invoice.Status == "pending" {
			pending++
		}
		rows = append(rows, adminRow{
			ID:           invoice.ID,
			WalletUserID: invoice.WalletUserID,
			Price:        Money(invoice.Price),
			Code:         invoice.Code,
			Total:        Money(invoice.Total),
			Status:       invoice.Status,
			StatusLabel:  statusLabelID(invoice.Status),
			CreatedAt:    invoice.ExpiresAt.Format("02 Jan 15:04"),
		})
	}
	return rows, pending
}

func redirectWith(response http.ResponseWriter, request *http.Request, path, key, message string) {
	http.Redirect(response, request, path+"?"+key+"="+url.QueryEscape(message), http.StatusFound)
}

package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/jhopan/Pan-Wallet/internal/backup"
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

// AuditStore mencatat dan membaca jejak tindakan admin.
type AuditStore interface {
	RecordAudit(context.Context, sqlite.AuditEntry) error
	AuditLog(context.Context, int) ([]sqlite.AuditEntry, error)
	AdjustBalance(context.Context, string, int64, string, string, string) (int64, error)
}

// RevenueStore menyediakan laporan pendapatan.
type RevenueStore interface {
	RevenueSummary(context.Context) (sqlite.RevenueSummary, error)
	RevenueByDay(context.Context, int) ([]sqlite.RevenueDay, error)
	TopupStats(context.Context) (sqlite.TopupStats, error)
}

// BackupRunner menjalankan backup database.
type BackupRunner interface {
	Once() (backup.Backup, error)
	List() ([]backup.Backup, error)
}

// adminAudit menampilkan jejak tindakan admin.
func (server *Server) adminAudit(response http.ResponseWriter, request *http.Request) {
	if server.deps.Audit == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	entries, err := server.deps.Audit.AuditLog(request.Context(), 200)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, map[string]any{
			"CreatedAt": entry.CreatedAt.Format("02 Jan 15:04"),
			"Actor":     entry.Actor,
			"Action":    entry.Action,
			"Wallet":    entry.WalletUserID,
			"Reference": entry.ReferenceID,
			"Details":   entry.Details,
		})
	}

	data := map[string]any{
		"Title":   "Audit Log",
		"Tab":     "audit",
		"Entries": rows,
		"User":    server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("audit", response, data)
}

// adminReport menampilkan laporan pendapatan.
func (server *Server) adminReport(response http.ResponseWriter, request *http.Request) {
	if server.deps.Revenue == nil {
		http.Error(response, "database belum siap", http.StatusServiceUnavailable)
		return
	}
	ctx := request.Context()

	summary, err := server.deps.Revenue.RevenueSummary(ctx)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	days, err := server.deps.Revenue.RevenueByDay(ctx, 30)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	dayRows := make([]map[string]any, 0, len(days))
	for _, day := range days {
		dayRows = append(dayRows, map[string]any{
			"Date":    day.Date,
			"Count":   day.Count,
			"Revenue": Money(day.Revenue),
		})
	}

	data := map[string]any{
		"Title":        "Laporan Pendapatan",
		"Tab":          "report",
		"TotalRevenue": Money(summary.TotalRevenue),
		"TodayRevenue": Money(summary.TodayRevenue),
		"TodayCount":   summary.TodayCount,
		"PaidCount":    summary.PaidCount,
		"PendingCount": summary.PendingCount,
		"ExpiredCount": summary.ExpiredCount,
		"Days":         dayRows,
		"User":         server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("report", response, data)
}

// adminAdjust menambah atau mengurangi saldo pelanggan secara manual.
func (server *Server) adminAdjust(response http.ResponseWriter, request *http.Request) {
	if server.deps.Audit == nil {
		redirectWith(response, request, "/admin/wallets", "err", "database belum siap")
		return
	}
	if err := request.ParseForm(); err != nil {
		redirectWith(response, request, "/admin/wallets", "err", "form tidak valid")
		return
	}

	walletUserID := strings.TrimSpace(request.FormValue("wallet_user_id"))
	amount, err := strconv.ParseInt(strings.TrimSpace(request.FormValue("amount")), 10, 64)
	if err != nil || amount == 0 {
		redirectWith(response, request, "/admin/wallets", "err", "nominal tidak valid")
		return
	}
	reason := strings.TrimSpace(request.FormValue("reason"))
	if reason == "" {
		redirectWith(response, request, "/admin/wallets", "err", "alasan wajib diisi")
		return
	}

	balance, err := server.deps.Audit.AdjustBalance(request.Context(), walletUserID, amount, reason,
		server.adminUser(), strings.TrimSpace(request.FormValue("reference_id")))
	if err != nil {
		redirectWith(response, request, "/admin/wallets", "err", err.Error())
		return
	}
	redirectWith(response, request, "/admin/wallets", "ok",
		"Saldo disesuaikan. Saldo sekarang Rp"+Rupiah(balance))
}

// adminBackup menampilkan daftar backup dan tombol buat sekarang.
func (server *Server) adminBackup(response http.ResponseWriter, request *http.Request) {
	if server.deps.Backup == nil {
		http.Error(response, "backup belum aktif", http.StatusServiceUnavailable)
		return
	}
	items, err := server.deps.Backup.List()
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"Name":      item.Name,
			"Size":      humanSize(item.SizeBytes),
			"CreatedAt": item.CreatedAt.Format("02 Jan 15:04"),
		})
	}

	data := map[string]any{
		"Title":   "Backup",
		"Tab":     "backup",
		"Backups": rows,
		"Flash":   request.URL.Query().Get("ok"),
		"Error":   request.URL.Query().Get("err"),
		"User":    server.adminUser(),
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("backup", response, data)
}

// adminBackupRun membuat backup baru sekarang.
func (server *Server) adminBackupRun(response http.ResponseWriter, request *http.Request) {
	if server.deps.Backup == nil {
		redirectWith(response, request, "/admin/backup", "err", "backup belum aktif")
		return
	}
	created, err := server.deps.Backup.Once()
	if err != nil {
		redirectWith(response, request, "/admin/backup", "err", err.Error())
		return
	}
	redirectWith(response, request, "/admin/backup", "ok", "Backup dibuat: "+created.Name)
}

// humanSize memformat ukuran berkas.
func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(bytes)/float64(div), 'f', 1, 64) + " " + []string{"KB", "MB", "GB"}[exp]
}

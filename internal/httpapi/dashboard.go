package httpapi

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type dashboardData struct {
	WalletCount int
	HoldCount   int
	LedgerCount int
	Wallets     []dashboardWallet
}

type dashboardWallet struct {
	WalletUserID string
	Balance      string
	HeldActive   string
	Available    string
	Currency     string
}

func (server *Server) dashboard(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(response, request)
		return
	}
	if server.overview == nil {
		writeError(response, http.StatusServiceUnavailable, "database belum siap")
		return
	}
	overview, err := server.overview.Overview(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, err.Error())
		return
	}

	data := dashboardData{
		WalletCount: overview.WalletCount,
		HoldCount:   overview.HoldCount,
		LedgerCount: overview.LedgerCount,
	}
	for _, wallet := range overview.Wallets {
		data.Wallets = append(data.Wallets, dashboardWallet{
			WalletUserID: wallet.WalletUserID,
			Balance:      formatIDR(wallet.Balance),
			HeldActive:   formatIDR(wallet.HeldActive),
			Available:    formatIDR(wallet.Available),
			Currency:     wallet.Currency,
		})
	}

	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplate.Execute(response, data); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func formatIDR(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	digits := strconv.FormatInt(amount, 10)
	var builder strings.Builder
	length := len(digits)
	for index, digit := range digits {
		if index > 0 && (length-index)%3 == 0 {
			builder.WriteByte('.')
		}
		builder.WriteRune(digit)
	}
	if negative {
		return "-" + builder.String()
	}
	return builder.String()
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pan-Wallet</title>
<style>
  :root { --ink:#1a1a1a; --muted:#6b6b6b; --line:#e2e2e2; --accent:#0f766e; }
  * { box-sizing:border-box; }
  body { margin:0; font-family:system-ui,-apple-system,"Segoe UI",sans-serif; color:var(--ink); background:#fafafa; }
  header { padding:24px 32px; border-bottom:1px solid var(--line); background:#fff; }
  header h1 { margin:0; font-size:20px; letter-spacing:-0.01em; }
  header p { margin:4px 0 0; color:var(--muted); font-size:13px; }
  main { padding:24px 32px; max-width:960px; }
  .stats { display:flex; gap:16px; flex-wrap:wrap; margin-bottom:28px; }
  .stat { background:#fff; border:1px solid var(--line); border-radius:10px; padding:16px 20px; min-width:150px; }
  .stat span { display:block; color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:0.05em; }
  .stat strong { font-size:24px; font-variant-numeric:tabular-nums; }
  table { width:100%; border-collapse:collapse; background:#fff; border:1px solid var(--line); border-radius:10px; overflow:hidden; }
  th,td { text-align:left; padding:12px 16px; border-bottom:1px solid var(--line); font-size:14px; }
  th { background:#f4f4f4; color:var(--muted); font-weight:600; font-size:12px; text-transform:uppercase; letter-spacing:0.04em; }
  td.num { text-align:right; font-variant-numeric:tabular-nums; }
  td.id { font-family:ui-monospace,monospace; font-size:12px; color:var(--muted); }
  tr:last-child td { border-bottom:none; }
  .empty { padding:32px; text-align:center; color:var(--muted); }
  footer { padding:16px 32px; color:var(--muted); font-size:12px; }
</style>
</head>
<body>
<header>
  <h1>Pan-Wallet</h1>
  <p>Saldo terpusat untuk AgenPulsa, Tembak Paket, AutoBuy, Xray Premium</p>
</header>
<main>
  <section class="stats">
    <div class="stat"><span>Wallet</span><strong>{{.WalletCount}}</strong></div>
    <div class="stat"><span>Hold</span><strong>{{.HoldCount}}</strong></div>
    <div class="stat"><span>Ledger</span><strong>{{.LedgerCount}}</strong></div>
  </section>
  <table>
    <thead>
      <tr><th>Wallet User</th><th class="num">Saldo</th><th class="num">Tertahan</th><th class="num">Tersedia</th></tr>
    </thead>
    <tbody>
    {{range .Wallets}}
      <tr>
        <td class="id">{{.WalletUserID}}</td>
        <td class="num">{{.Balance}}</td>
        <td class="num">{{.HeldActive}}</td>
        <td class="num">{{.Available}}</td>
      </tr>
    {{else}}
      <tr><td colspan="4" class="empty">Belum ada wallet.</td></tr>
    {{end}}
    </tbody>
  </table>
</main>
<footer>by JhopanStore</footer>
</body>
</html>
`))

package httpapi

import (
	"html/template"
	"strconv"
	"strings"
)

// Renderer menyatukan template panel kendali dan halaman checkout. Satu tempat
// supaya semua halaman memakai layout dan format rupiah yang sama.
type Renderer struct {
	pages map[string]*template.Template
}

func NewRenderer() *Renderer {
	return &Renderer{pages: map[string]*template.Template{
		"login":    template.Must(template.New("login").Parse(loginHTML)),
		"home":     template.Must(template.New("home").Parse(layoutHTML + homeHTML)),
		"topup":    template.Must(template.New("topup").Parse(layoutHTML + topupHTML)),
		"config":   template.Must(template.New("config").Parse(layoutHTML + configHTML)),
		"checkout": template.Must(template.New("checkout").Parse(checkoutHTML)),
	}}
}

func (renderer *Renderer) Render(name string, response interface{ Write([]byte) (int, error) }, data any) error {
	return renderer.pages[name].Execute(response, data)
}

// Rupiah memformat nominal integer menjadi "25.000".
func Rupiah(amount int64) string {
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

// Money adalah nominal yang dirender sebagai rupiah di template.
type Money int64

func (money Money) String() string { return Rupiah(int64(money)) }

const layoutHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - Pan-Wallet</title>
<style>
  :root { --ink:#141414; --muted:#6b6b6b; --line:#e4e4e4; --accent:#0f766e; --bg:#f7f7f6; }
  * { box-sizing:border-box; }
  body { margin:0; font-family:system-ui,-apple-system,"Segoe UI",sans-serif; color:var(--ink); background:var(--bg); }
  a { color:var(--accent); text-decoration:none; }
  a:hover { text-decoration:underline; }
  .shell { display:flex; min-height:100vh; }
  nav { width:210px; background:#fff; border-right:1px solid var(--line); padding:20px 0; flex-shrink:0; }
  nav .brand { padding:0 20px 16px; font-weight:700; font-size:17px; letter-spacing:-0.01em; }
  nav .brand small { display:block; font-weight:400; color:var(--muted); font-size:11px; margin-top:2px; }
  nav a { display:block; padding:10px 20px; color:var(--ink); font-size:14px; border-left:3px solid transparent; }
  nav a.active { background:#f0faf8; border-left-color:var(--accent); font-weight:600; }
  nav a:hover { background:#f6f6f6; text-decoration:none; }
  main { flex:1; padding:28px 32px; max-width:1080px; }
  h1 { font-size:21px; margin:0 0 4px; letter-spacing:-0.01em; }
  .sub { color:var(--muted); font-size:13px; margin:0 0 24px; }
  .card { background:#fff; border:1px solid var(--line); border-radius:12px; padding:20px 22px; margin-bottom:20px; }
  .card h2 { font-size:15px; margin:0 0 14px; }
  table { width:100%; border-collapse:collapse; }
  th,td { text-align:left; padding:10px 12px; border-bottom:1px solid var(--line); font-size:13px; }
  th { color:var(--muted); font-weight:600; font-size:11px; text-transform:uppercase; letter-spacing:0.04em; }
  td.num { text-align:right; font-variant-numeric:tabular-nums; }
  td.id { font-family:ui-monospace,monospace; font-size:11px; color:var(--muted); }
  tr:last-child td { border-bottom:none; }
  .stats { display:flex; gap:14px; flex-wrap:wrap; margin-bottom:22px; }
  .stat { background:#fff; border:1px solid var(--line); border-radius:12px; padding:14px 18px; min-width:140px; }
  .stat span { display:block; color:var(--muted); font-size:11px; text-transform:uppercase; letter-spacing:0.05em; }
  .stat strong { font-size:22px; font-variant-numeric:tabular-nums; }
  label { display:block; font-size:13px; margin-bottom:5px; color:var(--ink); }
  input,textarea,select { width:100%; padding:9px 11px; border:1px solid var(--line); border-radius:8px; font:inherit; font-size:14px; background:#fff; }
  input:focus,textarea:focus,select:focus { outline:2px solid var(--accent); outline-offset:1px; border-color:var(--accent); }
  button { background:var(--accent); color:#fff; border:none; border-radius:8px; padding:10px 18px; font:inherit; font-weight:600; cursor:pointer; }
  button:hover { background:#0c5f58; }
  button.ghost { background:#fff; color:var(--ink); border:1px solid var(--line); }
  button.ghost:hover { background:#f2f2f2; }
  .row { display:flex; gap:14px; flex-wrap:wrap; }
  .row > div { flex:1; min-width:180px; }
  .badge { display:inline-block; padding:2px 9px; border-radius:999px; font-size:11px; font-weight:600; }
  .b-pending { background:#fff3cd; color:#8a6d00; }
  .b-paid { background:#d4edda; color:#186a3b; }
  .b-expired,.b-cancelled { background:#f1f1f1; color:#666; }
  .flash { background:#e8f5f2; border:1px solid #b8ddd6; color:#0c5f58; padding:10px 14px; border-radius:8px; margin-bottom:16px; font-size:13px; }
  .err { background:#fdeaea; border:1px solid #f3c2c2; color:#8a1c1c; padding:10px 14px; border-radius:8px; margin-bottom:16px; font-size:13px; }
  .empty { padding:26px; text-align:center; color:var(--muted); font-size:13px; }
  footer { color:var(--muted); font-size:11px; padding:10px 0 0; }
  @media (max-width:720px){
    .shell{flex-direction:column}
    nav{width:100%;border-right:none;border-bottom:1px solid var(--line);padding:12px 0;display:flex;flex-wrap:wrap}
    nav .brand{width:100%;padding-bottom:8px}
    nav a{border-left:none;border-bottom:3px solid transparent;padding:8px 14px}
    nav a.active{border-left:none;border-bottom-color:var(--accent)}
    main{padding:18px}
  }
</style>
</head>
<body>
<div class="shell">
<nav>
  <div class="brand">Pan-Wallet<small>by JhopanStore</small></div>
  <a href="/admin" class="{{if eq .Tab "home"}}active{{end}}">Dashboard</a>
  <a href="/admin/topup" class="{{if eq .Tab "topup"}}active{{end}}">Top-up QRIS</a>
  <a href="/admin/config" class="{{if eq .Tab "config"}}active{{end}}">Konfigurasi</a>
  <a href="/" class="">Saldo</a>
  <a href="/admin/logout">Keluar</a>
</nav>
<main>
{{if .Flash}}<div class="flash">{{.Flash}}</div>{{end}}
{{if .Error}}<div class="err">{{.Error}}</div>{{end}}
`

const homeHTML = `
<h1>Dashboard</h1>
<p class="sub">Ringkasan saldo dan invoice top-up.</p>
<section class="stats">
  <div class="stat"><span>Wallet</span><strong>{{.Overview.WalletCount}}</strong></div>
  <div class="stat"><span>Hold</span><strong>{{.Overview.HoldCount}}</strong></div>
  <div class="stat"><span>Ledger</span><strong>{{.Overview.LedgerCount}}</strong></div>
  <div class="stat"><span>Invoice pending</span><strong>{{.PendingCount}}</strong></div>
</section>
<div class="card">
  <h2>Saldo pelanggan</h2>
  <table>
    <thead><tr><th>Wallet User</th><th class="num">Saldo</th><th class="num">Tertahan</th><th class="num">Tersedia</th></tr></thead>
    <tbody>
    {{range .Wallets}}
      <tr><td class="id">{{.WalletUserID}}</td><td class="num">{{.Balance}}</td><td class="num">{{.HeldActive}}</td><td class="num">{{.Available}}</td></tr>
    {{else}}
      <tr><td colspan="4" class="empty">Belum ada wallet.</td></tr>
    {{end}}
    </tbody>
  </table>
</div>
<div class="card">
  <h2>Invoice top-up terbaru</h2>
  <table>
    <thead><tr><th>Dibuat</th><th>Status</th><th class="num">Nominal</th><th class="num">Total</th><th></th></tr></thead>
    <tbody>
    {{range .Invoices}}
      <tr>
        <td>{{.CreatedAt}}</td>
        <td><span class="badge b-{{.Status}}">{{.Status}}</span></td>
        <td class="num">{{.Price}}</td>
        <td class="num">{{.Total}}</td>
        <td><a href="/pay/{{.ID}}">checkout</a></td>
      </tr>
    {{else}}
      <tr><td colspan="5" class="empty">Belum ada invoice top-up.</td></tr>
    {{end}}
    </tbody>
  </table>
</div>
<footer>Pan-Wallet &middot; by JhopanStore</footer>
</main>
</div>
</body>
</html>
`

const topupHTML = `
<h1>Top-up QRIS</h1>
<p class="sub">Buat invoice top-up bernominal unik, lalu kirim link checkout ke pelanggan.</p>
<div class="card">
  <h2>Buat invoice</h2>
  <form method="post" action="/admin/topup">
    <div class="row">
      <div>
        <label for="wallet_user_id">Wallet User ID</label>
        <input id="wallet_user_id" name="wallet_user_id" required placeholder="uuid wallet user">
      </div>
      <div>
        <label for="price">Nominal top-up (Rp)</label>
        <input id="price" name="price" type="number" min="1000" step="1" required placeholder="25000">
      </div>
      <div style="flex:0;align-self:flex-end"><button type="submit">Buat</button></div>
    </div>
  </form>
</div>
<div class="card">
  <h2>Invoice</h2>
  <table>
    <thead><tr><th>Dibuat</th><th>Wallet</th><th>Status</th><th class="num">Nominal</th><th class="num">Kode</th><th class="num">Total</th><th>Aksi</th></tr></thead>
    <tbody>
    {{range .Invoices}}
      <tr>
        <td>{{.CreatedAt}}</td>
        <td class="id">{{.WalletUserID}}</td>
        <td><span class="badge b-{{.Status}}">{{.Status}}</span></td>
        <td class="num">{{.Price}}</td>
        <td class="num">{{.Code}}</td>
        <td class="num">{{.Total}}</td>
        <td>
          <a href="/pay/{{.ID}}">checkout</a>
          {{if eq .Status "pending"}}
          <form method="post" action="/admin/topup/{{.ID}}/paid" style="display:inline">
            <button class="ghost" type="submit" title="Tandai lunas manual">lunas</button>
          </form>
          {{end}}
        </td>
      </tr>
    {{else}}
      <tr><td colspan="7" class="empty">Belum ada invoice.</td></tr>
    {{end}}
    </tbody>
  </table>
</div>
<footer>Pan-Wallet &middot; by JhopanStore</footer>
</main>
</div>
</body>
</html>
`

const configHTML = `
<h1>Konfigurasi</h1>
<p class="sub">Payload QRIS statis merchant dan pengaturan gateway.</p>
<div class="card">
  <h2>QRIS</h2>
  <form method="post" action="/admin/config">
    <label for="qris_base">Payload QRIS statis</label>
    <textarea id="qris_base" name="qris_base" rows="4" placeholder="000201010211...5802ID...6304XXXX">{{.QrisBase}}</textarea>
    <p class="sub" style="margin:8px 0 14px">Tempel payload QRIS statis dari aplikasi merchant. Nominal diisi otomatis per invoice.</p>
    <div class="row">
      <div>
        <label for="invoice_ttl_minutes">Masa berlaku invoice (menit)</label>
        <input id="invoice_ttl_minutes" name="invoice_ttl_minutes" type="number" min="1" max="120" value="{{.TTLMinutes}}">
      </div>
      <div style="flex:0;align-self:flex-end"><button type="submit">Simpan</button></div>
    </div>
  </form>
</div>
<div class="card">
  <h2>Info</h2>
  <table>
    <tbody>
      <tr><th>Status QRIS</th><td>{{.QrisStatus}}</td></tr>
      <tr><th>Engine database</th><td>{{.Engine}}</td></tr>
      <tr><th>Jumlah migrasi</th><td>{{.Migrations}}</td></tr>
    </tbody>
  </table>
</div>
<footer>Pan-Wallet &middot; by JhopanStore</footer>
</main>
</div>
</body>
</html>
`

const loginHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Masuk - Pan-Wallet</title>
<style>
  body { margin:0; font-family:system-ui,-apple-system,"Segoe UI",sans-serif; background:#f7f7f6; color:#141414; display:flex; min-height:100vh; align-items:center; justify-content:center; }
  .card { background:#fff; border:1px solid #e4e4e4; border-radius:14px; padding:28px 30px; width:100%; max-width:360px; }
  h1 { font-size:19px; margin:0 0 4px; }
  p.sub { color:#6b6b6b; font-size:13px; margin:0 0 22px; }
  label { display:block; font-size:13px; margin-bottom:5px; }
  input { width:100%; padding:10px 12px; border:1px solid #e4e4e4; border-radius:8px; font:inherit; margin-bottom:14px; }
  input:focus { outline:2px solid #0f766e; outline-offset:1px; }
  button { width:100%; background:#0f766e; color:#fff; border:none; border-radius:8px; padding:11px; font:inherit; font-weight:600; cursor:pointer; }
  button:hover { background:#0c5f58; }
  .err { background:#fdeaea; border:1px solid #f3c2c2; color:#8a1c1c; padding:10px 14px; border-radius:8px; margin-bottom:16px; font-size:13px; }
  footer { color:#6b6b6b; font-size:11px; margin-top:18px; text-align:center; }
</style>
</head>
<body>
<form class="card" method="post" action="/admin/login">
  <h1>Pan-Wallet</h1>
  <p class="sub">Panel kendali wallet dan gateway QRIS</p>
  {{if .Error}}<div class="err">{{.Error}}</div>{{end}}
  <label for="username">Username</label>
  <input id="username" name="username" autocomplete="username" required>
  <label for="password">Password</label>
  <input id="password" name="password" type="password" autocomplete="current-password" required>
  <button type="submit">Masuk</button>
  <footer>by JhopanStore</footer>
</form>
</body>
</html>
`

const checkoutHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pembayaran QRIS</title>
<style>
  body { margin:0; font-family:system-ui,-apple-system,"Segoe UI",sans-serif; background:#f6f7f9; display:flex; justify-content:center; padding:24px; }
  .card { background:#fff; border-radius:16px; max-width:380px; width:100%; padding:24px; box-shadow:0 2px 12px rgba(0,0,0,.08); text-align:center; }
  h1 { font-size:18px; margin:0 0 4px; }
  .amount { font-size:32px; font-weight:700; margin:8px 0; font-variant-numeric:tabular-nums; }
  .qr { width:260px; height:260px; margin:12px auto; border:1px solid #eee; border-radius:12px; padding:8px; }
  .badge { display:inline-block; padding:6px 14px; border-radius:999px; font-weight:600; font-size:14px; }
  .pending { background:#fff3cd; color:#8a6d00; }
  .paid { background:#d4edda; color:#186a3b; }
  .expired,.cancelled { background:#f8d7da; color:#8a1c1c; }
  small { color:#888; }
  .paid .qr,.expired .qr,.cancelled .qr { filter:grayscale(1); opacity:.5; }
  footer { margin-top:16px; font-size:11px; color:#98a2b3; }
</style>
</head>
<body>
<div class="card {{.Status}}">
  <h1>Pembayaran QRIS</h1>
  <div class="amount">Rp {{.Total}}</div>
  <span class="badge {{.Status}}" id="badge">{{.StatusLabel}}</span>
  <div><img class="qr" id="qr" src="/pay/{{.ID}}/qr.png" alt="QRIS" {{if not .ShowQR}}style="display:none"{{end}}></div>
  {{if .ShowQR}}<small>Scan dengan aplikasi e-wallet atau m-banking.<br>Halaman ini memperbarui otomatis.</small>
  {{else}}<small>QR tidak tersedia untuk invoice ini.</small>{{end}}
  <footer>Powered by <b>JhopanStore</b></footer>
</div>
<script>
(function poll(){
  fetch('/api/topup/{{.ID}}').then(function(r){return r.json()}).then(function(j){
    if(j.status==='paid'){document.getElementById('badge').textContent='LUNAS';location.reload();}
    else if(j.status==='expired'||j.status==='cancelled'){location.reload();}
    else setTimeout(poll,3000);
  }).catch(function(){setTimeout(poll,5000)});
})();
</script>
</body>
</html>
`

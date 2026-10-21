package httpapi

import (
	"html/template"
	"strconv"
	"strings"
)

// Renderer menyatukan template panel kendali dan halaman checkout.
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

// Layout mengikuti gaya PayPan: sidebar gelap, aksen hijau, kartu putih.
const layoutHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - Pan-Wallet</title>
<style>
*{box-sizing:border-box}
body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f2f4f8;margin:0;color:#101828}
.layout{display:flex;min-height:100vh}
.sidebar{width:230px;background:#101828;color:#cbd5e1;padding:18px 14px;display:flex;flex-direction:column;position:fixed;top:0;bottom:0;left:0}
.brand{display:flex;align-items:center;gap:10px;padding:6px 8px 20px;border-bottom:1px solid #1e293b;margin-bottom:14px}
.brand .dot{width:36px;height:36px;border-radius:10px;background:#1a7f37;color:#fff;display:flex;align-items:center;justify-content:center;font-weight:800;font-size:17px;flex-shrink:0}
.brand b{color:#fff;font-size:16px;display:block;line-height:1.1}
.brand small{color:#64748b;font-size:11px}
.menu a{display:flex;align-items:center;gap:10px;padding:11px 12px;border-radius:10px;color:#cbd5e1;text-decoration:none;font-size:14px;font-weight:500;margin-bottom:4px}
.menu a:hover{background:#1e293b;color:#fff}
.menu a.on{background:#1a7f37;color:#fff;font-weight:600}
.menu .ico{width:20px;text-align:center;flex-shrink:0}
.sidebar .foot{margin-top:auto;padding:12px 8px 0;border-top:1px solid #1e293b;font-size:12px;color:#64748b}
.main{margin-left:230px;flex:1;min-width:0}
header{background:#fff;border-bottom:1px solid #e4e7ec;padding:14px 28px;display:flex;justify-content:space-between;align-items:center;position:sticky;top:0;z-index:5}
header h1{font-size:18px;margin:0;color:#101828}
header .right{display:flex;align-items:center;gap:14px}
header .who{font-size:13px;color:#667085}
header a.out{color:#b42318;text-decoration:none;font-size:13px;font-weight:600;padding:7px 14px;border:1px solid #fda29b;border-radius:8px}
header a.out:hover{background:#fee4e2}
.content{max-width:980px;margin:24px auto;padding:0 24px}
.card{background:#fff;border-radius:12px;padding:20px;margin-bottom:20px;box-shadow:0 1px 6px rgba(0,0,0,.06)}
h2{margin:0 0 12px;font-size:16px;color:#101828}
table{width:100%;border-collapse:collapse;font-size:14px}
th,td{padding:8px 10px;border-bottom:1px solid #eee;text-align:left}
th{color:#667085;font-weight:600;background:#f9fafb}
code{background:#f2f4f8;padding:2px 6px;border-radius:6px;font-size:13px}
input,select,textarea{padding:8px;border:1px solid #d0d5dd;border-radius:8px;margin:4px 0;font-size:14px;width:100%;font-family:inherit}
input:focus,textarea:focus,select:focus{outline:2px solid #1a7f37;border-color:#1a7f37}
label{font-size:13px;color:#344054;font-weight:600;display:block;margin:10px 0 2px}
button{padding:8px 14px;background:#1a7f37;color:#fff;border:0;border-radius:8px;cursor:pointer;font-weight:600;font-size:13px}
button:hover{background:#166f30}
button.sec{background:#fff;color:#344054;border:1px solid #d0d5dd}
button.sec:hover{background:#f9fafb}
.badge{padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600}
.paid{background:#d4edda;color:#186a3b}
.pending{background:#fff3cd;color:#8a6d00}
.expired,.cancelled{background:#f8d7da;color:#8a1c1c}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:12px;margin-bottom:20px}
.stat{background:#fff;border-radius:12px;padding:16px;box-shadow:0 1px 6px rgba(0,0,0,.06);text-align:center}
.stat .n{font-size:26px;font-weight:800;color:#101828;font-variant-numeric:tabular-nums}
.stat .l{font-size:12px;color:#667085;margin-top:2px}
td small{color:#98a2b3;font-size:12px;white-space:nowrap}
td.empty{text-align:center;color:#98a2b3;padding:18px}
.money{font-variant-numeric:tabular-nums;text-align:right}
th.money{text-align:right}
.flash{background:#d4edda;color:#186a3b;padding:10px 14px;border-radius:8px;margin-bottom:14px}
.err{background:#fee4e2;color:#b42318;padding:10px 14px;border-radius:8px;margin-bottom:14px}
.pg{color:#1a7f37;text-decoration:none;font-weight:600;margin:0 6px}
a.act{display:inline-block;padding:5px 11px;border:1px solid #d0d5dd;border-radius:7px;color:#344054;font-size:12px;font-weight:600;text-decoration:none;background:#fff}
a.act:hover{background:#f9fafb;border-color:#98a2b3}
.row{display:flex;gap:14px;flex-wrap:wrap}
.row>div{flex:1;min-width:180px}
form.inline{display:inline}
@media(max-width:800px){.sidebar{display:none}.main{margin-left:0}.content{padding:0 14px}}
</style>
</head>
<body>
<div class="layout">
<div class="sidebar">
<div class="brand"><div class="dot">P</div><div><b>Pan-Wallet</b><small>by JhopanStore</small></div></div>
<div class="menu">
<a href="/admin" class="{{if eq .Tab "home"}}on{{end}}"><span class="ico">▤</span> Dashboard</a>
<a href="/admin/topup" class="{{if eq .Tab "topup"}}on{{end}}"><span class="ico">▣</span> Top-up QRIS</a>
<a href="/admin/config" class="{{if eq .Tab "config"}}on{{end}}"><span class="ico">⚙</span> Konfigurasi</a>
</div>
<div class="foot">v1.0 - JhopanStore</div>
</div>
<div class="main">
<header><h1>{{.Title}}</h1>
<div class="right"><span class="who">{{.User}}</span><a class="out" href="/admin/logout">Keluar</a></div></header>
<div class="content">
{{if .Flash}}<div class="flash">{{.Flash}}</div>{{end}}
{{if .Error}}<div class="err">{{.Error}}</div>{{end}}
`

const homeHTML = `
<div class="stats">
  <div class="stat"><div class="n">{{.Overview.WalletCount}}</div><div class="l">Dompet</div></div>
  <div class="stat"><div class="n">{{.Overview.HoldCount}}</div><div class="l">Dana tertahan</div></div>
  <div class="stat"><div class="n">{{.Overview.LedgerCount}}</div><div class="l">Mutasi</div></div>
  <div class="stat"><div class="n">{{.PendingCount}}</div><div class="l">Invoice tertunda</div></div>
</div>
<div class="card"><h2>Saldo pelanggan</h2><table>
<tr><th>Wallet User</th><th class="money">Saldo</th><th class="money">Tertahan</th><th class="money">Tersedia</th></tr>
{{range .Wallets}}<tr><td><code>{{.WalletUserID}}</code></td><td class="money">{{.Balance}}</td><td class="money">{{.HeldActive}}</td><td class="money"><b>{{.Available}}</b></td></tr>
{{else}}<tr><td colspan="4" class="empty">Belum ada wallet</td></tr>{{end}}
</table></div>
<div class="card"><h2>Invoice top-up terbaru</h2><table>
<tr><th>Dibuat</th><th>Status</th><th class="money">Nominal</th><th class="money">Total</th><th>Aksi</th></tr>
{{range .Invoices}}<tr><td><small>{{.CreatedAt}}</small></td><td><span class="badge {{.Status}}">{{.StatusLabel}}</span></td><td class="money">{{.Price}}</td><td class="money"><b>{{.Total}}</b></td><td>{{if eq .Status "pending"}}<a class="act" href="/pay/{{.ID}}">checkout</a>{{else}}<small>-</small>{{end}}</td></tr>
{{else}}<tr><td colspan="5" class="empty">Belum ada invoice</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const topupHTML = `
<div class="card"><h2>Buat invoice top-up</h2>
<form method="post" action="/admin/topup">
<div class="row">
<div><label for="wallet_user_id">Wallet User ID</label><input id="wallet_user_id" name="wallet_user_id" required placeholder="id wallet user"></div>
<div><label for="price">Nominal top-up (Rp)</label><input id="price" name="price" type="number" min="1000" step="1" required placeholder="25000"></div>
<div style="flex:0;align-self:flex-end"><button type="submit">Buat invoice</button></div>
</div>
</form></div>
<div class="card"><h2>Daftar invoice</h2><table>
<tr><th>Dibuat</th><th>Wallet</th><th>Status</th><th class="money">Nominal</th><th class="money">Kode</th><th class="money">Total</th><th>Aksi</th></tr>
{{range .Invoices}}<tr>
<td><small>{{.CreatedAt}}</small></td>
<td><code>{{.WalletUserID}}</code></td>
<td><span class="badge {{.Status}}">{{.StatusLabel}}</span></td>
<td class="money">{{.Price}}</td>
<td class="money">{{.Code}}</td>
<td class="money"><b>{{.Total}}</b></td>
<td><a class="act" href="/pay/{{.ID}}">checkout</a>
{{if eq .Status "pending"}}
<form class="inline" method="post" action="/admin/topup/{{.ID}}/sync"><button class="sec" type="submit" title="Cek status ke PayPan">sync</button></form>
<form class="inline" method="post" action="/admin/topup/{{.ID}}/paid"><button class="sec" type="submit" title="Tandai lunas manual">lunas</button></form>
{{end}}</td>
</tr>
{{else}}<tr><td colspan="7" class="empty">Belum ada invoice</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const configHTML = `
<div class="card"><h2>PayPan (gateway QRIS)</h2>
<form method="post" action="/admin/config">
<label for="paypan_base_url">Base URL PayPan</label>
<input id="paypan_base_url" name="paypan_base_url" placeholder="https://paypan.jhopan.my.id" value="{{.PaypanBase}}">
<label for="paypan_token">Token API (scope order)</label>
<input id="paypan_token" name="paypan_token" placeholder="pp_xxxxxxxxxxxxxxxx" value="{{.PaypanToken}}">
<label for="paypan_webhook_secret">Webhook secret</label>
<input id="paypan_webhook_secret" name="paypan_webhook_secret" placeholder="pp_xxxxxxxxxxxxxxxx" value="{{.WebhookSecret}}">
<div style="margin-top:16px"><button type="submit">Simpan</button></div>
</form>
<table style="margin-top:18px">
<tr><th>Status konfigurasi</th><td>{{.PaypanStatus}}</td></tr>
<tr><th>URL webhook untuk PayPan</th><td><code>{{.WebhookURL}}</code></td></tr>
</table>
<small style="color:#667085;display:block;margin-top:10px">Daftarkan URL webhook di atas ke admin PayPan &rarr; Konfigurasi &rarr; Webhook, lalu salin secret-nya ke kolom Webhook secret.</small>
</div>
</div></div></div></body></html>
`

const loginHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Login - Pan-Wallet</title>
<style>
body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f2f4f8;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.c{background:#fff;padding:36px;border-radius:16px;box-shadow:0 2px 16px rgba(0,0,0,.09);width:340px}
.logo{display:flex;align-items:center;gap:10px;margin-bottom:20px}
.logo .dot{width:38px;height:38px;border-radius:10px;background:#101828;color:#fff;display:flex;align-items:center;justify-content:center;font-weight:800;font-size:18px}
.logo b{font-size:19px} .logo small{display:block;color:#667085;font-size:12px}
label{font-size:13px;color:#344054;font-weight:600;display:block;margin:10px 0 4px}
input{width:100%;box-sizing:border-box;padding:10px;margin:0 0 4px;border:1px solid #d0d5dd;border-radius:8px;font-size:14px}
input:focus{outline:2px solid #1a7f37;border-color:#1a7f37}
button{width:100%;padding:11px;background:#1a7f37;color:#fff;border:0;border-radius:8px;font-weight:600;cursor:pointer;margin-top:14px;font-size:15px}
button:hover{background:#166f30}
.e{color:#b42318;font-size:14px;background:#fee4e2;padding:8px 12px;border-radius:8px;margin-bottom:10px}
.hint{color:#667085;font-size:12px;margin-top:14px;text-align:center}
</style>
</head>
<body>
<div class="c">
<div class="logo"><div class="dot">P</div><div><b>Pan-Wallet</b><small>by JhopanStore</small></div></div>
{{if .Error}}<p class="e">{{.Error}}</p>{{end}}
<form method="post" action="/admin/login">
<label for="username">Username</label><input id="username" type="text" name="username" placeholder="username" autofocus autocomplete="username">
<label for="password">Password</label><input id="password" type="password" name="password" placeholder="password" autocomplete="current-password">
<button>Login</button>
</form>
<div class="hint">@Jhopanstore</div>
</div>
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
body{margin:0;font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f2f4f8;display:flex;justify-content:center;align-items:center;min-height:100vh;padding:24px}
.card{background:#fff;border-radius:16px;max-width:380px;width:100%;padding:24px;box-shadow:0 2px 16px rgba(0,0,0,.09);text-align:center}
h1{font-size:18px;margin:0 0 4px;color:#101828}
.amount{font-size:32px;font-weight:800;margin:8px 0;font-variant-numeric:tabular-nums;color:#101828}
.qr{width:260px;height:260px;margin:12px auto;border:1px solid #eee;border-radius:12px;padding:8px}
.badge{display:inline-block;padding:6px 14px;border-radius:999px;font-weight:600;font-size:14px}
.pending{background:#fff3cd;color:#8a6d00}
.paid{background:#d4edda;color:#186a3b}
.expired,.cancelled{background:#f8d7da;color:#8a1c1c}
small{color:#667085}
.paid .qr,.expired .qr,.cancelled .qr{filter:grayscale(1);opacity:.5}
footer{margin-top:16px;font-size:11px;color:#98a2b3}
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

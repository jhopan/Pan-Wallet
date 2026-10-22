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
		"wallets":  template.Must(template.New("wallets").Parse(layoutHTML + walletsHTML)),
		"ledger":   template.Must(template.New("ledger").Parse(layoutHTML + ledgerHTML)),
		"keys":     template.Must(template.New("keys").Parse(layoutHTML + keysHTML)),
		"audit":    template.Must(template.New("audit").Parse(layoutHTML + auditHTML)),
		"report":   template.Must(template.New("report").Parse(layoutHTML + reportHTML)),
		"backup":   template.Must(template.New("backup").Parse(layoutHTML + backupHTML)),
		"checkout": template.Must(template.New("checkout").Parse(checkoutHTML)),
		"portal":   template.Must(template.New("portal").Parse(portalHTML)),
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
.paid{background:#d4edda;color:#0f5132}
.pending{background:#fff3cd;color:#664d03}
.expired,.cancelled{background:#f8d7da;color:#842029}
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
<a href="/admin/wallets" class="{{if eq .Tab "wallets"}}on{{end}}"><span class="ico">◉</span> Wallet User</a>
<a href="/admin/ledger" class="{{if eq .Tab "ledger"}}on{{end}}"><span class="ico">▦</span> Mutasi Saldo</a>
<a href="/admin/keys" class="{{if eq .Tab "keys"}}on{{end}}"><span class="ico">⧉</span> API Key Produk</a>
<a href="/admin/laporan" class="{{if eq .Tab "report"}}on{{end}}"><span class="ico">▩</span> Laporan</a>
<a href="/admin/audit" class="{{if eq .Tab "audit"}}on{{end}}"><span class="ico">☰</span> Audit Log</a>
<a href="/admin/backup" class="{{if eq .Tab "backup"}}on{{end}}"><span class="ico">⛁</span> Backup</a>
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
<tr><th>ID Pelanggan</th><th class="money">Saldo</th><th class="money">Tertahan</th><th class="money">Tersedia</th></tr>
{{range .Wallets}}<tr><td><b style="letter-spacing:0.05em">{{.PublicID}}</b></td><td class="money">{{.Balance}}</td><td class="money">{{.HeldActive}}</td><td class="money"><b>{{.Available}}</b></td></tr>
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
<div><label for="price">Nominal top-up (Rp)</label><input id="price" name="price" type="number" min="25000" step="1" required placeholder="25000"></div>
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

<div class="card"><h2>Notifikasi</h2>
<p style="font-size:13px;color:#667085;margin:0 0 14px">Dapatkan kabar saat ada top-up masuk. Bisa lewat Telegram, webhook, atau keduanya.</p>
<form method="post" action="/admin/notify">
<label for="notify_tg_token">Telegram bot token</label>
<input id="notify_tg_token" name="notify_tg_token" placeholder="123456789:AA..." value="{{.NotifyToken}}">
<label for="notify_tg_chat">Chat ID admin (boleh beberapa, pisah koma)</label>
<input id="notify_tg_chat" name="notify_tg_chat" placeholder="123456789,987654321" value="{{.NotifyChat}}">
<label for="notify_webhook_url">Webhook URL (opsional)</label>
<input id="notify_webhook_url" name="notify_webhook_url" placeholder="https://bot-anda/hook" value="{{.NotifyWebhook}}">
<div style="margin-top:16px"><button type="submit">Simpan</button></div>
</form>
<form method="post" action="/admin/notify/test" style="margin-top:12px">
<button class="sec" type="submit">Kirim pesan uji</button>
</form>
<small style="color:#667085;display:block;margin-top:10px">Pesan uji memakai konfigurasi yang sudah tersimpan. Simpan dulu sebelum menguji.</small>
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
.pending{background:#fff3cd;color:#664d03}
.paid{background:#d4edda;color:#0f5132}
.expired,.cancelled{background:#f8d7da;color:#842029}
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

const ledgerHTML = `
<div class="card"><h2>Cari wallet</h2>
<form method="get" action="/admin/ledger">
<div class="row">
<div><label for="wallet">Wallet User ID</label>
<input id="wallet" name="wallet" value="{{.WalletUserID}}" placeholder="id wallet user"></div>
<div style="flex:0;align-self:flex-end"><button type="submit">Tampilkan</button></div>
</div>
</form></div>
<div class="card"><h2>Riwayat mutasi</h2><table>
<tr><th>Waktu</th><th>Jenis</th><th>Produk</th><th>Referensi</th><th class="money">Nominal</th><th class="money">Saldo akhir</th></tr>
{{range .Entries}}<tr>
<td><small>{{.CreatedAt}}</small></td>
<td>{{.EntryType}}</td>
<td><code>{{.Product}}</code></td>
<td><code>{{.ReferenceID}}</code></td>
<td class="money">{{if .IsCredit}}<span style="color:#1a7f37">+{{.Amount}}</span>{{else}}<span style="color:#b42318">-{{.Amount}}</span>{{end}}</td>
<td class="money"><b>{{.BalanceAfter}}</b></td>
</tr>
{{else}}<tr><td colspan="6" class="empty">Belum ada mutasi untuk wallet ini</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const keysHTML = `
{{if .NewToken}}<div class="card" style="border-color:#1a7f37">
<h2>Token baru (salin sekarang)</h2>
<p style="font-size:13px;color:#667085;margin:0 0 10px">Token ini hanya ditampilkan sekali. Simpan di env/bot kamu.</p>
<code style="display:block;padding:12px;background:#f2f4f8;border-radius:8px;font-size:15px;user-select:all">{{.NewToken}}</code>
</div>{{end}}
<div class="card"><h2>Buat / rotasi key</h2>
<form method="post" action="/admin/keys">
<div class="row">
<div><label for="product">Nama produk</label>
<input id="product" name="product" placeholder="agenpulsa" required></div>
<div style="flex:0;align-self:flex-end"><button type="submit">Buat key</button></div>
</div>
</form>
<small style="color:#667085;display:block;margin-top:10px">Nama produk yang sudah ada akan dirotasi: token lama langsung mati.</small>
</div>
<div class="card"><h2>Cara pakai untuk bot lain</h2>
<p style="font-size:13px;color:#667085;margin:0 0 12px">
Bot toko (AgenPulsa, Xray, dll) memakai <b>API key produk</b> di atas untuk cek saldo dan memotong saldo.
Bot tidak boleh menyimpan saldo sendiri.
</p>

<h3 style="font-size:13px;margin:16px 0 6px">1. Cek saldo pelanggan</h3>
<pre style="background:#f2f4f8;padding:12px;border-radius:8px;overflow-x:auto;font-size:12px;margin:0">curl -s {{.BaseURL}}/v1/wallets/&lt;wallet_user_id&gt; \
  -H "Authorization: Bearer &lt;API_KEY&gt;"</pre>

<h3 style="font-size:13px;margin:16px 0 6px">2. Kunci saldo sebelum order (hold)</h3>
<pre style="background:#f2f4f8;padding:12px;border-radius:8px;overflow-x:auto;font-size:12px;margin:0">curl -s -X POST {{.BaseURL}}/v1/holds \
  -H "Authorization: Bearer &lt;API_KEY&gt;" \
  -H "Idempotency-Key: hold:&lt;REF&gt;" \
  -H "Content-Type: application/json" \
  -d '{"wallet_user_id":"&lt;ID&gt;","reference_id":"&lt;REF&gt;","amount":25000,"expires_at":"2030-01-01T00:00:00Z"}'</pre>

<h3 style="font-size:13px;margin:16px 0 6px">3a. Order sukses &rarr; potong saldo</h3>
<pre style="background:#f2f4f8;padding:12px;border-radius:8px;overflow-x:auto;font-size:12px;margin:0">curl -s -X POST {{.BaseURL}}/v1/holds/&lt;HOLD_ID&gt;/capture \
  -H "Authorization: Bearer &lt;API_KEY&gt;" \
  -H "Idempotency-Key: capture:&lt;REF&gt;"</pre>

<h3 style="font-size:13px;margin:16px 0 6px">3b. Order gagal &rarr; lepas kunci</h3>
<pre style="background:#f2f4f8;padding:12px;border-radius:8px;overflow-x:auto;font-size:12px;margin:0">curl -s -X POST {{.BaseURL}}/v1/holds/&lt;HOLD_ID&gt;/release \
  -H "Authorization: Bearer &lt;API_KEY&gt;" \
  -H "Idempotency-Key: release:&lt;REF&gt;"</pre>

<h3 style="font-size:13px;margin:16px 0 6px">4. Cek saldo pakai ID pelanggan (tanpa API key)</h3>
<pre style="background:#f2f4f8;padding:12px;border-radius:8px;overflow-x:auto;font-size:12px;margin:0">curl -s {{.BaseURL}}/v1/public/&lt;PUBLIC_ID&gt;</pre>

<small style="color:#667085;display:block;margin-top:14px">
Dokumentasi lengkap: <code>docs/API.md</code> di repo Pan-Wallet.
</small>
</div>

<div class="card"><h2>Daftar key</h2><table>
<tr><th>Produk</th><th>Prefix</th><th>Status</th><th>Rotasi terakhir</th><th>Aksi</th></tr>
{{range .Keys}}<tr>
<td><code>{{.Product}}</code></td>
<td><code>{{.TokenPrefix}}...</code></td>
<td><span class="badge {{if .Active}}paid{{else}}expired{{end}}">{{.StatusLabel}}</span></td>
<td><small>{{.RotatedAt}}</small></td>
<td>{{if .Active}}<form class="inline" method="post" action="/admin/keys/{{.Product}}/revoke"><button class="sec" type="submit">nonaktifkan</button></form>{{else}}<small>-</small>{{end}}</td>
</tr>
{{else}}<tr><td colspan="5" class="empty">Belum ada API key</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const walletsHTML = `
<div class="card"><h2>Buat wallet user</h2>
<form method="post" action="/admin/wallets">
<div class="row">
<div><label for="initial_balance">Saldo awal (opsional)</label>
<input id="initial_balance" name="initial_balance" type="number" min="0" step="1" placeholder="0"></div>
<div style="flex:0;align-self:flex-end"><button type="submit">Buat wallet</button></div>
</div>
</form>
<small style="color:#667085;display:block;margin-top:10px">ID wallet dibuat otomatis. Saldo awal boleh 0, nanti diisi lewat top-up QRIS.</small>
</div>
<div class="card"><h2>Sesuaikan saldo manual</h2>
<form method="post" action="/admin/wallets/adjust">
<div class="row">
<div><label for="adj_wallet">Wallet User ID</label><input id="adj_wallet" name="wallet_user_id" required placeholder="id wallet"></div>
<div><label for="adj_amount">Nominal (+ tambah / - kurangi)</label><input id="adj_amount" name="amount" type="number" step="1" required placeholder="50000"></div>
</div>
<div class="row" style="margin-top:10px">
<div><label for="adj_reason">Alasan (wajib)</label><input id="adj_reason" name="reason" required placeholder="bonus / koreksi selisih"></div>
<div style="flex:0;align-self:flex-end"><button type="submit">Simpan</button></div>
</div>
</form>
<small style="color:#667085;display:block;margin-top:10px">Tercatat di ledger sebagai adjustment dan masuk audit log.</small>
</div>
<div class="card"><h2>Daftar wallet</h2><table>
<tr><th>Dibuat</th><th>ID Pelanggan</th><th>Status</th><th class="money">Saldo</th><th class="money">Tertahan</th><th class="money">Tersedia</th><th>Aksi</th></tr>
{{range .Wallets}}<tr>
<td><small>{{.CreatedAt}}</small></td>
<td><b style="font-size:15px;letter-spacing:0.05em">{{.PublicID}}</b></td>
<td><span class="badge {{.StatusClass}}">{{.StatusLabel}}</span></td>
<td class="money">{{.Balance}}</td>
<td class="money">{{.HeldActive}}</td>
<td class="money"><b>{{.Available}}</b></td>
<td>
<a class="act" href="/admin/ledger?wallet={{.WalletUserID}}">mutasi</a>
<form class="inline" method="post" action="/admin/wallets/{{.WalletUserID}}/status">
{{if eq .StatusLabel "aktif"}}<button class="sec" name="status" value="blocked" type="submit">blokir</button>
{{else}}<button class="sec" name="status" value="active" type="submit">aktifkan</button>{{end}}
</form>
</td>
</tr>
{{else}}<tr><td colspan="7" class="empty">Belum ada wallet</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const portalHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - Pan-Wallet</title>
<style>
body{margin:0;font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f2f4f8;color:#101828;padding:24px;display:flex;justify-content:center}
.wrap{width:100%;max-width:440px}
.card{background:#fff;border-radius:16px;padding:26px;box-shadow:0 2px 16px rgba(0,0,0,.08);margin-bottom:16px}
h1{font-size:19px;margin:0 0 4px}
p.sub{color:#667085;font-size:13px;margin:0 0 18px}
label{display:block;font-size:13px;color:#344054;font-weight:600;margin:0 0 5px}
input{width:100%;box-sizing:border-box;padding:11px;border:1px solid #d0d5dd;border-radius:8px;font-size:15px;margin-bottom:14px}
input:focus{outline:2px solid #1a7f37;border-color:#1a7f37}
button{width:100%;padding:12px;background:#1a7f37;color:#fff;border:0;border-radius:8px;font-weight:600;cursor:pointer;font-size:15px}
button:hover{background:#166f30}
button.sec{background:#fff;color:#344054;border:1px solid #d0d5dd}
.err{background:#fee4e2;color:#b42318;padding:10px 14px;border-radius:8px;margin-bottom:14px;font-size:13px}
.code{background:#f2f4f8;border:1px dashed #98a2b3;border-radius:10px;padding:16px;text-align:center;font-size:30px;font-weight:800;letter-spacing:0.18em;margin:14px 0}
.amount{font-size:34px;font-weight:800;margin:6px 0;font-variant-numeric:tabular-nums}
.rowline{display:flex;justify-content:space-between;font-size:13px;color:#667085;padding:6px 0;border-bottom:1px solid #f0f0f0}
.rowline b{color:#101828}
table{width:100%;border-collapse:collapse;font-size:13px}
th,td{padding:7px 6px;border-bottom:1px solid #eee;text-align:left}
th{color:#667085;font-size:11px;text-transform:uppercase}
td.money{text-align:right;font-variant-numeric:tabular-nums}
.badge{display:inline-block;padding:2px 9px;border-radius:999px;font-size:11px;font-weight:600;background:#d4edda;color:#0f5132}
footer{color:#98a2b3;font-size:11px;text-align:center;margin-top:8px}
</style>
</head>
<body>
<div class="wrap">

{{if eq .Step "email"}}
<div class="card">
  <h1>Saldo Saya</h1>
  <p class="sub">Masukkan email yang sudah terhubung ke bot Telegram.<br>Minimal top-up Rp 25.000. Saldo tidak dapat diuangkan.</p>
  {{if .Error}}<div class="err">{{.Error}}</div>{{end}}
  <form method="post" action="/masuk">
    <label for="email">Email</label>
    <input id="email" name="email" type="email" placeholder="nama@email.com" required autofocus>
    <button type="submit">Lanjut</button>
  </form>
  <footer>by JhopanStore</footer>
</div>
{{end}}

{{if eq .Step "code"}}
<div class="card">
  <h1>Ikat akun</h1>
  <p class="sub">Email <b>{{.Email}}</b> belum terhubung. Kirim kode ini ke bot Telegram Pan-Wallet.</p>
  <div class="code">{{.Code}}</div>
  <p class="sub">Buka Telegram, kirim:<br><b>/ikat {{.Code}}</b></p>
  <form method="post" action="/masuk/cek">
    <input type="hidden" name="email" value="{{.Email}}">
    <button type="submit">Saya sudah kirim</button>
  </form>
  <footer>Kode berlaku 10 menit</footer>
</div>
{{end}}

{{if eq .Step "wallet"}}
<div class="card">
  <h1>Saldo Saya</h1>
  <p class="sub">ID Pelanggan: <b style="letter-spacing:0.08em">{{.PublicID}}</b></p>
  <div class="amount">Rp {{.Available}}</div>
  <div class="rowline"><span>Tersedia</span><b>Rp {{.Available}}</b></div>
  <div class="rowline"><span>Tertahan</span><b>Rp {{.Held}}</b></div>
  <div class="rowline"><span>Total saldo</span><b>Rp {{.Balance}}</b></div>
  <div class="rowline"><span>Status</span><b><span class="badge">{{.Status}}</span></b></div>
  <div style="margin-top:14px;padding:10px 12px;background:#fff8e6;border:1px solid #ffe0a3;border-radius:8px;font-size:12px;color:#7a5a00">
    Saldo hanya bisa dipakai untuk transaksi di layanan kami. Saldo tidak dapat diuangkan atau ditarik tunai.
  </div>
  <footer>Untuk top-up, buka bot Telegram Pan-Wallet. Minimal top-up Rp 25.000.</footer>
</div>

<div class="card">
  <h2 style="font-size:15px;margin:0 0 10px">Riwayat terakhir</h2>
  <table>
    <tr><th>Waktu</th><th>Jenis</th><th class="money">Nominal</th></tr>
    {{range .Entries}}
    <tr>
      <td>{{.CreatedAt}}</td>
      <td>{{.EntryType}}</td>
      <td class="money">{{if .IsCredit}}<span style="color:#0f5132">+{{.Amount}}</span>{{else}}<span style="color:#b42318">-{{.Amount}}</span>{{end}}</td>
    </tr>
    {{else}}
    <tr><td colspan="3" style="text-align:center;color:#98a2b3;padding:16px">Belum ada transaksi</td></tr>
    {{end}}
  </table>
</div>

<div class="card">
  <h2 style="font-size:15px;margin:0 0 8px">Akun terhubung</h2>
  {{range .Links}}<div class="rowline"><span>{{.}}</span></div>{{else}}<div class="rowline"><span>-</span></div>{{end}}
  <footer><a href="/masuk" style="color:#1a7f37">Keluar</a></footer>
</div>
{{end}}

</div>
</body>
</html>
`

const reportHTML = `
<div class="stats">
  <div class="stat"><div class="n">{{.TodayRevenue}}</div><div class="l">Pendapatan hari ini</div></div>
  <div class="stat"><div class="n">{{.TodayCount}}</div><div class="l">Transaksi hari ini</div></div>
  <div class="stat"><div class="n">{{.TotalRevenue}}</div><div class="l">Total pendapatan</div></div>
  <div class="stat"><div class="n">{{.PaidCount}}</div><div class="l">Invoice lunas</div></div>
</div>
<div class="card"><h2>Status invoice</h2><table>
<tr><th>Status</th><th class="money">Jumlah</th></tr>
<tr><td><span class="badge paid">lunas</span></td><td class="money">{{.PaidCount}}</td></tr>
<tr><td><span class="badge pending">tertunda</span></td><td class="money">{{.PendingCount}}</td></tr>
<tr><td><span class="badge expired">kedaluwarsa</span></td><td class="money">{{.ExpiredCount}}</td></tr>
</table></div>
<div class="card"><h2>Pendapatan harian (30 hari)</h2><table>
<tr><th>Tanggal</th><th class="money">Transaksi</th><th class="money">Pendapatan</th></tr>
{{range .Days}}<tr><td>{{.Date}}</td><td class="money">{{.Count}}</td><td class="money"><b>{{.Revenue}}</b></td></tr>
{{else}}<tr><td colspan="3" class="empty">Belum ada pendapatan</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const auditHTML = `
<div class="card"><h2>Jejak tindakan admin</h2>
<small style="color:#667085;display:block;margin-bottom:12px">Semua perubahan saldo, blokir, dan rotasi key tercatat di sini.</small>
<table>
<tr><th>Waktu</th><th>Pelaku</th><th>Aksi</th><th>Wallet</th><th>Referensi</th><th>Detail</th></tr>
{{range .Entries}}<tr>
<td><small>{{.CreatedAt}}</small></td>
<td>{{.Actor}}</td>
<td><code>{{.Action}}</code></td>
<td><small>{{.Wallet}}</small></td>
<td><small>{{.Reference}}</small></td>
<td><small>{{.Details}}</small></td>
</tr>
{{else}}<tr><td colspan="6" class="empty">Belum ada aktivitas</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

const backupHTML = `
<div class="card"><h2>Backup database</h2>
<p style="font-size:13px;color:#667085;margin:0 0 12px">Backup otomatis dibuat setiap jam dan disimpan di folder <code>backup/</code>. Berkas lama dihapus otomatis agar tidak menumpuk.</p>
<form method="post" action="/admin/backup"><button type="submit">Buat backup sekarang</button></form>
</div>
<div class="card"><h2>Daftar backup</h2><table>
<tr><th>Berkas</th><th class="money">Ukuran</th><th>Dibuat</th></tr>
{{range .Backups}}<tr><td><code>{{.Name}}</code></td><td class="money">{{.Size}}</td><td><small>{{.CreatedAt}}</small></td></tr>
{{else}}<tr><td colspan="3" class="empty">Belum ada backup</td></tr>{{end}}
</table></div>
</div></div></div></body></html>
`

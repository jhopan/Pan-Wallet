package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

// dashboard adalah halaman publik "/" yang hanya menyapa, tanpa data saldo.
// Saldo hanya tampil di panel admin yang butuh login.
func (server *Server) dashboard(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(response, request)
		return
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Write([]byte(`<!doctype html><html lang="id"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>Pan-Wallet</title>
<style>body{margin:0;font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f7f7f6;color:#141414;display:flex;min-height:100vh;align-items:center;justify-content:center}
.card{background:#fff;border:1px solid #e4e4e4;border-radius:14px;padding:28px 32px;max-width:420px;text-align:center}
h1{margin:0 0 6px;font-size:20px}p{color:#6b6b6b;font-size:13px;margin:0 0 18px}
a{display:inline-block;background:#0f766e;color:#fff;text-decoration:none;padding:10px 20px;border-radius:8px;font-weight:600;font-size:14px}
a:hover{background:#0c5f58}footer{color:#6b6b6b;font-size:11px;margin-top:18px}</style></head><body>
<div class="card"><h1>Pan-Wallet</h1><p>Saldo terpusat untuk AgenPulsa, Tembak Paket, AutoBuy, dan Xray Premium.</p>
<a href="/admin/login">Masuk panel</a><footer>by JhopanStore</footer></div></body></html>`))
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

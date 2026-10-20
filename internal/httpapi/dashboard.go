package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

// root meniru situs yang tidak ada: 404 besar di tengah halaman, tanpa satu pun
// petunjuk bahwa Pan-Wallet ada di sini. Panel hanya lewat /admin.
func (server *Server) root(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(http.StatusNotFound)
	response.Write([]byte(emptySiteHTML))
}

// emptySiteHTML sengaja tidak memuat nama produk, saldo, atau tautan panel.
const emptySiteHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>404 Not Found</title>
<style>
  * { box-sizing: border-box; }
  html, body { height: 100%; }
  body {
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #fbfbfb;
    color: #2b2b2b;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    padding: 24px;
  }
  main { text-align: center; max-width: 720px; }
  h1 {
    margin: 0;
    font-size: clamp(96px, 22vw, 220px);
    line-height: 0.9;
    font-weight: 800;
    letter-spacing: -0.05em;
    color: #1b1b1b;
  }
  p {
    margin: 24px 0 0;
    font-size: clamp(15px, 2.2vw, 19px);
    color: #7a7a7a;
    line-height: 1.6;
  }
  hr {
    border: 0;
    border-top: 1px solid #e6e6e6;
    margin: 32px auto 0;
    width: min(320px, 70%);
  }
</style>
</head>
<body>
<main>
  <h1>404</h1>
  <p>This page could not be found.</p>
  <hr>
</main>
</body>
</html>
`

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

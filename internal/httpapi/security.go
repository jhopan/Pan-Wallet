package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

// securityHeaders memasang header pengaman dasar pada setiap respons.
// Tanpa ini panel bisa di-embed di iframe situs lain (clickjacking)
// dan browser tidak dipaksa memakai HTTPS.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		header := response.Header()
		header.Set("X-Frame-Options", "DENY")
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "same-origin")
		header.Set("Content-Security-Policy",
			"default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; form-action 'self'")
		// HSTS hanya berguna di HTTPS; aman dikirim selalu karena browser
		// mengabaikannya pada koneksi http biasa.
		header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next.ServeHTTP(response, request)
	})
}

// sameOriginOnly menolak POST/PUT/DELETE yang datang dari situs lain.
// Ini pertahanan CSRF: penyerang tidak bisa memaksa admin yang sedang
// login untuk melakukan aksi (mis. mengubah saldo) dari halaman jahat.
func sameOriginOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(response, request)
			return
		}

		// Webhook PayPan berasal dari server lain dan sudah diautentikasi
		// dengan HMAC, jadi tidak ikut aturan origin.
		if strings.HasPrefix(request.URL.Path, "/webhook/") {
			next.ServeHTTP(response, request)
			return
		}

		origin := request.Header.Get("Origin")
		if origin == "" {
			// Sebagian klien (curl, bot) tidak mengirim Origin.
			// Kalau ada Referer, tetap diverifikasi.
			referer := request.Header.Get("Referer")
			if referer == "" {
				next.ServeHTTP(response, request)
				return
			}
			origin = referer
		}

		parsed, err := url.Parse(origin)
		if err != nil || !sameHost(parsed.Host, request.Host) {
			writeError(response, http.StatusForbidden, "origin tidak diizinkan")
			return
		}

		next.ServeHTTP(response, request)
	})
}

// sameHost membandingkan host tanpa peduli port default.
func sameHost(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}
	return strings.EqualFold(stripPort(a), stripPort(b))
}

func stripPort(host string) string {
	if index := strings.LastIndex(host, ":"); index > 0 {
		return host[:index]
	}
	return host
}

// isHTTPS mendeteksi koneksi terenkripsi, termasuk di belakang reverse proxy.
func isHTTPS(request *http.Request) bool {
	if request.TLS != nil {
		return true
	}
	forwarded := request.Header.Get("X-Forwarded-Proto")
	return strings.EqualFold(strings.TrimSpace(forwarded), "https")
}

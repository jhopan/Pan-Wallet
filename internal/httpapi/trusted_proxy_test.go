package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestRequest membuat request untuk pengujian.
func newTestRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	request.RemoteAddr = "203.0.113.1:1234"
	return request
}

func TestClientIPIgnoresSpoofedHeader(t *testing.T) {
	// Tanpa proxy tepercaya, header X-Forwarded-For harus diabaikan.
	server := &Server{trustedProxies: newTrustedProxies(nil)}

	request := newTestRequest(t, "GET", "/v1/public/ABC", "")
	request.RemoteAddr = "203.0.113.9:5555"
	request.Header.Set("X-Forwarded-For", "1.2.3.4")
	request.Header.Set("CF-Connecting-IP", "5.6.7.8")

	if got := server.clientIP(request); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want 203.0.113.9 (header palsu harus diabaikan)", got)
	}
}

func TestClientIPTrustsConfiguredProxy(t *testing.T) {
	server := &Server{trustedProxies: newTrustedProxies([]string{"127.0.0.1"})}

	request := newTestRequest(t, "GET", "/v1/public/ABC", "")
	request.RemoteAddr = "127.0.0.1:5555"
	request.Header.Set("CF-Connecting-IP", "198.51.100.7")

	if got := server.clientIP(request); got != "198.51.100.7" {
		t.Fatalf("clientIP = %q, want 198.51.100.7 (proxy tepercaya)", got)
	}
}

func TestClientIPIgnoresHeaderFromUntrustedSource(t *testing.T) {
	server := &Server{trustedProxies: newTrustedProxies([]string{"127.0.0.1"})}

	request := newTestRequest(t, "GET", "/v1/public/ABC", "")
	request.RemoteAddr = "203.0.113.9:5555" // bukan proxy tepercaya
	request.Header.Set("CF-Connecting-IP", "198.51.100.7")

	if got := server.clientIP(request); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want 203.0.113.9", got)
	}
}

func TestTrustedProxiesParsesCIDRAndSingleIP(t *testing.T) {
	trusted := newTrustedProxies([]string{"10.0.0.0/8", "192.168.1.5", " ", "rusak"})

	if !trusted.contains("10.1.2.3") {
		t.Fatal("10.1.2.3 harus tepercaya (CIDR)")
	}
	if !trusted.contains("192.168.1.5") {
		t.Fatal("192.168.1.5 harus tepercaya (IP tunggal)")
	}
	if trusted.contains("8.8.8.8") {
		t.Fatal("8.8.8.8 tidak boleh tepercaya")
	}
}

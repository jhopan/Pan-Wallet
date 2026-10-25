package paypan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ping harus melaporkan sehat kalau PayPan merespons.
func TestPingReportsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("path = %q, want /", r.URL.Path)
		}
		// PayPan asli membalas teks biasa, bukan JSON.
		w.Write([]byte("Paypan payment gateway OK"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test")
	status := client.Ping(context.Background())

	if !status.OK {
		t.Fatalf("harus sehat, dapat: %s", status.Message)
	}
	if status.LatencyMS < 0 {
		t.Fatalf("latensi tidak masuk akal: %d", status.LatencyMS)
	}
}

// Server mati harus terdeteksi, bukan dianggap sehat.
func TestPingDetectsDownServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // matikan sebelum dipakai

	client := NewClient(url, "pp_test")
	status := client.Ping(context.Background())

	if status.OK {
		t.Fatal("server mati tidak boleh dianggap sehat")
	}
	if status.Message == "" {
		t.Fatal("harus ada pesan kesalahan")
	}
}

// Token ditolak (401) berarti konfigurasi salah — jangan lapor sehat.
func TestPingDetectsBadToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"ok":false,"error":"token tidak valid"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_salah")
	status := client.Ping(context.Background())

	if status.OK {
		t.Fatal("token salah tidak boleh dianggap sehat")
	}
}

// Latensi harus terukur supaya admin tahu PayPan lambat atau tidak.
func TestPingMeasuresLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test")
	status := client.Ping(context.Background())

	if status.LatencyMS < 0 {
		t.Fatalf("latensi = %d", status.LatencyMS)
	}
}

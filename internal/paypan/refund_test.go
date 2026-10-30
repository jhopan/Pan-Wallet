package paypan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRefundInvoiceCallsPaypan(t *testing.T) {
	var gotPath, gotMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"ok":true,"data":{"id":"inv-1","status":"refunded"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test")
	if err := client.RefundInvoice(context.Background(), "inv-1"); err != nil {
		t.Fatalf("RefundInvoice: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/invoice/inv-1/refund" {
		t.Fatalf("path = %q, want /api/invoice/inv-1/refund", gotPath)
	}
}

func TestRefundInvoiceReportsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"ok":false,"error":"bukan paid / tidak ditemukan"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test")
	if err := client.RefundInvoice(context.Background(), "inv-1"); err == nil {
		t.Fatal("harus melaporkan error")
	}
}

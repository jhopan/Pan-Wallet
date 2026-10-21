package paypan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateInvoiceSendsAuthAndParsesResponse(t *testing.T) {
	var gotPath, gotAuth, gotPrice string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")

		var body struct {
			Price int64  `json:"price"`
			Label string `json:"label"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		gotPrice = string(rune(body.Price))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":{"id":"inv_123","price":50000,"code":12,"total":50012,"status":"pending","expires_at":1788966000,"pay_url":"/pay/inv_123","qr_url":"/pay/inv_123/qr.png","qr":"000201010212"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test_token")
	invoice, err := client.CreateInvoice(context.Background(), 50000, "topup wallet")
	if err != nil {
		t.Fatalf("CreateInvoice error = %v", err)
	}

	if gotPath != "/api/invoice" {
		t.Fatalf("path = %q, want /api/invoice", gotPath)
	}
	if gotAuth != "Bearer pp_test_token" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotPrice == "" {
		t.Fatal("price tidak terkirim")
	}
	if invoice.ID != "inv_123" || invoice.Total != 50012 || invoice.Code != 12 {
		t.Fatalf("invoice = %+v", invoice)
	}
	if invoice.Status != "pending" {
		t.Fatalf("status = %q", invoice.Status)
	}
}

func TestCreateInvoiceReturnsErrorFromPaypan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ok":false,"error":"minimal Rp1.000"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test")
	_, err := client.CreateInvoice(context.Background(), 500, "")
	if err == nil {
		t.Fatal("error = nil, want error dari PayPan")
	}
	if err.Error() != "minimal Rp1.000" {
		t.Fatalf("error = %q, want pesan PayPan", err.Error())
	}
}

func TestGetInvoiceParsesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/invoice/inv_123" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Write([]byte(`{"ok":true,"data":{"id":"inv_123","status":"paid","price":50000,"code":12,"total":50012,"paid_at":1788965650}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "pp_test")
	invoice, err := client.GetInvoice(context.Background(), "inv_123")
	if err != nil {
		t.Fatalf("GetInvoice error = %v", err)
	}
	if invoice.Status != "paid" {
		t.Fatalf("status = %q, want paid", invoice.Status)
	}
}

func TestVerifyWebhookAcceptsValidSignature(t *testing.T) {
	body := []byte(`{"event":"order.paid","order":{"id":"inv_123"}}`)
	secret := "pp_secret123"
	signature := Sign(body, secret)

	if !VerifyWebhook(body, signature, secret) {
		t.Fatal("signature valid harus diterima")
	}
	if VerifyWebhook(body, "deadbeef", secret) {
		t.Fatal("signature salah harus ditolak")
	}
	if VerifyWebhook(body, signature, "secret_lain") {
		t.Fatal("secret salah harus ditolak")
	}
}

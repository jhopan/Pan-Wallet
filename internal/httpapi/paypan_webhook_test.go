package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jhopan/PanWallet/internal/paypan"
	"github.com/jhopan/PanWallet/internal/sqlite"
	"github.com/jhopan/PanWallet/internal/topup"
)

// Order milik aplikasi lain (mis. AgenPulsa) harus diabaikan dengan 200.
// Kalau dibalas error, PayPan menganggap gagal dan mencoba ulang 5x
// dengan backoff 30s/2m/5m/15m — sia-sia dan memenuhi antrean retry.
func TestWebhookUnknownOrderIsIgnored(t *testing.T) {
	const secret = "secret-uji"
	store := &fakeTopups{err: sqlite.ErrInvoiceNotFound}
	server, signature := newWebhookServer(store, secret)

	body := `{"event":"order.paid","order":{"id":"order-orang-lain"}}`
	request := httptest.NewRequest(http.MethodPost, "/webhook/paypan", strings.NewReader(body))
	request.Header.Set("X-Paypan-Signature", paypan.Sign([]byte(body), secret))
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()
	_ = signature

	server.paypanWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (order asing harus diabaikan)", response.Code)
	}
	if !strings.Contains(response.Body.String(), "ignored") {
		t.Fatalf("body = %q, harus menandai diabaikan", response.Body.String())
	}
}

// Error lain (mis. database gagal) tetap harus dilaporkan sebagai error.
func TestWebhookRealErrorStillReported(t *testing.T) {
	const secret = "secret-uji"
	store := &fakeTopups{err: errors.New("koneksi database gagal")}
	server, _ := newWebhookServer(store, secret)

	body := `{"event":"order.paid","order":{"id":"order-x"}}`
	request := httptest.NewRequest(http.MethodPost, "/webhook/paypan", strings.NewReader(body))
	request.Header.Set("X-Paypan-Signature", paypan.Sign([]byte(body), secret))
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()

	server.paypanWebhook(response, request)

	if response.Code == http.StatusOK {
		t.Fatalf("error asli tidak boleh dibalas 200: %s", response.Body.String())
	}
}

// Order milik PanWallet sendiri tetap diproses seperti biasa.
func TestWebhookOwnOrderIsProcessed(t *testing.T) {
	store := &fakeTopups{invoice: topup.Invoice{
		ID: "inv-1", WalletUserID: "w1", Total: 25004, FreshlyPaid: true,
	}}
	const secret = "secret-uji"
	server, _ := newWebhookServer(store, secret)

	body := `{"event":"order.paid","order":{"id":"order-milik-saya"}}`
	request := httptest.NewRequest(http.MethodPost, "/webhook/paypan", strings.NewReader(body))
	request.Header.Set("X-Paypan-Signature", paypan.Sign([]byte(body), secret))
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()

	server.paypanWebhook(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "credited") {
		t.Fatalf("body = %q, harus menyatakan kredit", response.Body.String())
	}
}

// newWebhookServer membuat server dengan secret webhook dan body yang ditandatangani.
func newWebhookServer(store TopupStore, secret string) (*Server, string) {
	server := &Server{deps: Deps{
		Topups:   store,
		Settings: fakeSettings{"paypan_webhook_secret": secret},
	}}
	body := `{"event":"order.paid","order":{"id":"order-x"}}`
	signature := paypan.Sign([]byte(body), secret)
	return server, signature
}

// fakeTopups adalah stub untuk menguji handler webhook.
type fakeTopups struct {
	invoice topup.Invoice
	err     error
}

func (fake *fakeTopups) CreateTopupInvoice(ctx context.Context, client *paypan.Client, walletUserID string, price int64) (topup.Invoice, error) {
	return topup.Invoice{}, nil
}

func (fake *fakeTopups) TopupInvoice(ctx context.Context, invoiceID string) (topup.Invoice, error) {
	return topup.Invoice{}, nil
}

func (fake *fakeTopups) MarkTopupPaidByPaypan(ctx context.Context, paypanID, source string) (topup.Invoice, error) {
	if fake.err != nil {
		return topup.Invoice{}, fake.err
	}
	return fake.invoice, nil
}

func (fake *fakeTopups) MarkTopupPaidByLocalID(ctx context.Context, localID, source string) (topup.Invoice, error) {
	return topup.Invoice{}, nil
}

func (fake *fakeTopups) RecentTopupInvoices(ctx context.Context, limit int) ([]topup.Invoice, error) {
	return nil, nil
}

func (fake *fakeTopups) ExpireTopupInvoice(ctx context.Context, invoiceID string) error {
	return nil
}

package sqlite

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhopan/PanWallet/internal/paypan"
)

// fakePaypan meniru POST /api/invoice dan GET /api/invoice/{id}.
func fakePaypan(t *testing.T, total int64) (*httptest.Server, *paypan.Client) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/invoice":
			var body struct {
				Price int64 `json:"price"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			w.Write([]byte(`{"ok":true,"data":{"id":"pp_inv_1","price":` +
				itoa(body.Price) + `,"code":12,"total":` + itoa(total) +
				`,"status":"pending","expires_at":1900000000,"pay_url":"/pay/pp_inv_1"}}`))
		case r.Method == http.MethodGet:
			w.Write([]byte(`{"ok":true,"data":{"id":"pp_inv_1","status":"paid","price":50000,"code":12,"total":` + itoa(total) + `}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	return server, paypan.NewClient(server.URL, "pp_test")
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}

func TestCreateTopupInvoiceUsesPaypan(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 0)
	_, client := fakePaypan(t, 50_012)

	invoice, err := store.CreateTopupInvoice(context.Background(), client, userID, 50_000)
	if err != nil {
		t.Fatalf("CreateTopupInvoice: %v", err)
	}
	if invoice.PaypanInvoice != "pp_inv_1" {
		t.Fatalf("paypan invoice = %q", invoice.PaypanInvoice)
	}
	if invoice.Total != 50_012 {
		t.Fatalf("total = %d, want 50012 dari PayPan", invoice.Total)
	}
	if invoice.Status != "pending" {
		t.Fatalf("status = %q", invoice.Status)
	}
}

func TestMarkPaidByPaypanCreditsOnce(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 10_000)
	_, client := fakePaypan(t, 50_012)
	ctx := context.Background()

	if _, err := store.CreateTopupInvoice(ctx, client, userID, 50_000); err != nil {
		t.Fatalf("buat invoice: %v", err)
	}

	paid, err := store.MarkTopupPaidByPaypan(ctx, "pp_inv_1", "paypan-webhook")
	if err != nil {
		t.Fatalf("MarkTopupPaidByPaypan: %v", err)
	}
	if paid.Status != "paid" {
		t.Fatalf("status = %q, want paid", paid.Status)
	}

	overview, _ := store.Overview(ctx)
	if overview.Wallets[0].Balance != 60_012 {
		t.Fatalf("Balance = %d, want 60012", overview.Wallets[0].Balance)
	}
	if overview.LedgerCount != 1 {
		t.Fatalf("LedgerCount = %d, want 1", overview.LedgerCount)
	}

	// Webhook dobel: tidak boleh kredit lagi.
	if _, err := store.MarkTopupPaidByPaypan(ctx, "pp_inv_1", "paypan-webhook"); err != nil {
		t.Fatalf("webhook ulang harus idempotent: %v", err)
	}
	overview, _ = store.Overview(ctx)
	if overview.Wallets[0].Balance != 60_012 {
		t.Fatalf("Balance setelah webhook ulang = %d, want 60012", overview.Wallets[0].Balance)
	}
	if overview.LedgerCount != 1 {
		t.Fatalf("LedgerCount setelah webhook ulang = %d, want 1", overview.LedgerCount)
	}
}

func TestTopupInvoiceLookupByPaypanID(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 0)
	_, client := fakePaypan(t, 20_005)
	ctx := context.Background()

	created, err := store.CreateTopupInvoice(ctx, client, userID, 25_000)
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}

	byLocal, err := store.TopupInvoice(ctx, created.ID)
	if err != nil {
		t.Fatalf("TopupInvoice: %v", err)
	}
	if byLocal.Total != created.Total {
		t.Fatalf("total tidak cocok")
	}

	byPaypan, err := store.TopupInvoiceByPaypan(ctx, "pp_inv_1")
	if err != nil {
		t.Fatalf("TopupInvoiceByPaypan: %v", err)
	}
	if byPaypan.ID != created.ID {
		t.Fatalf("invoice id = %q, want %q", byPaypan.ID, created.ID)
	}
}

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/paypan"
	"github.com/jhopan/Pan-Wallet/internal/topup"
)

type fakeReconcileStore struct {
	pending   []topup.Invoice
	paidCalls []string
	expired   []string
}

func (store *fakeReconcileStore) PendingTopupInvoices(context.Context, time.Time) ([]topup.Invoice, error) {
	return store.pending, nil
}
func (store *fakeReconcileStore) MarkTopupPaidByPaypan(_ context.Context, id, _ string) (topup.Invoice, error) {
	store.paidCalls = append(store.paidCalls, id)
	return topup.Invoice{ID: "local-" + id, PaypanInvoice: id, Total: 50001}, nil
}
func (store *fakeReconcileStore) ExpireTopupInvoice(_ context.Context, id string) error {
	store.expired = append(store.expired, id)
	return nil
}

func TestReconcileMarksPaidInvoice(t *testing.T) {
	paypanServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"data":{"id":"pp_1","status":"paid","total":50001}}`))
	}))
	defer paypanServer.Close()

	store := &fakeReconcileStore{pending: []topup.Invoice{
		{ID: "local-1", PaypanInvoice: "pp_1", Status: "pending"},
	}}
	server := &Server{
		deps: Deps{
			Reconcile: store,
			Settings:  fakeSettings{"paypan_base_url": paypanServer.URL, "paypan_token": "pp_x"},
		},
	}

	server.reconcileOnce(context.Background())

	if len(store.paidCalls) != 1 || store.paidCalls[0] != "pp_1" {
		t.Fatalf("paidCalls = %v, want [pp_1]", store.paidCalls)
	}
}

func TestReconcileExpiresStaleInvoice(t *testing.T) {
	paypanServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"data":{"id":"pp_2","status":"expired","total":50001}}`))
	}))
	defer paypanServer.Close()

	store := &fakeReconcileStore{pending: []topup.Invoice{
		{ID: "local-2", PaypanInvoice: "pp_2", Status: "pending"},
	}}
	server := &Server{
		deps: Deps{
			Reconcile: store,
			Settings:  fakeSettings{"paypan_base_url": paypanServer.URL, "paypan_token": "pp_x"},
		},
	}

	server.reconcileOnce(context.Background())

	if len(store.expired) != 1 || store.expired[0] != "local-2" {
		t.Fatalf("expired = %v, want [local-2]", store.expired)
	}
	if len(store.paidCalls) != 0 {
		t.Fatal("invoice expired tidak boleh dikredit")
	}
}

func TestReconcileLeavesPendingAlone(t *testing.T) {
	paypanServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true,"data":{"id":"pp_3","status":"pending","total":50001}}`))
	}))
	defer paypanServer.Close()

	store := &fakeReconcileStore{pending: []topup.Invoice{
		{ID: "local-3", PaypanInvoice: "pp_3", Status: "pending"},
	}}
	server := &Server{
		deps: Deps{
			Reconcile: store,
			Settings:  fakeSettings{"paypan_base_url": paypanServer.URL, "paypan_token": "pp_x"},
		},
	}

	server.reconcileOnce(context.Background())

	if len(store.paidCalls) != 0 || len(store.expired) != 0 {
		t.Fatal("invoice pending tidak boleh diubah")
	}
}

type fakeSettings map[string]string

func (settings fakeSettings) Setting(_ context.Context, key string) (string, error) {
	return settings[key], nil
}
func (fakeSettings) SetSetting(context.Context, string, string) error { return nil }

var _ = paypan.NewClient

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jhopan/Pan-Wallet/internal/report"
)

type fakeOverviewStore struct{}

func (fakeOverviewStore) Overview(context.Context) (report.Overview, error) {
	return report.Overview{
		WalletCount: 1,
		HoldCount:   2,
		LedgerCount: 3,
		Wallets: []report.OverviewWallet{{
			WalletUserID: "e469f462-5040-450f-ac2a-48987fbbbe7f",
			Balance:      100000,
			HeldActive:   25000,
			Available:    75000,
			Currency:     "IDR",
		}},
	}, nil
}

// Halaman publik "/" hanya menyapa: tidak boleh membocorkan saldo tanpa login.
func TestPublicRootDoesNotLeakBalance(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	New(Deps{Overview: fakeOverviewStore{}}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "100.000") || strings.Contains(body, "e469f462") {
		t.Fatal("halaman publik membocorkan data saldo")
	}
}

// Panel admin (setelah login) menampilkan saldo terformat.
func TestAdminHomeRendersWalletRows(t *testing.T) {
	auth := NewAdminAuth("admin", "rahasia123")
	token := auth.StartSession()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.AddCookie(&http.Cookie{Name: adminCookie, Value: token})

	New(Deps{Overview: fakeOverviewStore{}, Admin: auth}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{"100.000", "75.000", "e469f462"} {
		if !strings.Contains(body, want) {
			t.Fatalf("panel admin tidak memuat %q", want)
		}
	}
}

// Tanpa sesi, /admin harus mengalihkan ke halaman login.
func TestAdminRequiresSession(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)

	New(Deps{Admin: NewAdminAuth("admin", "rahasia123")}).ServeHTTP(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", response.Code)
	}
	if location := response.Header().Get("Location"); location != "/admin/login" {
		t.Fatalf("Location = %q, want /admin/login", location)
	}
}

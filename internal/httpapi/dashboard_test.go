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

func TestDashboardRendersWalletRows(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	New(nil, nil, fakeOverviewStore{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{"Pan-Wallet", "100.000", "75.000", "e469f462"} {
		if !strings.Contains(body, want) {
			t.Fatalf("dashboard body missing %q", want)
		}
	}
}

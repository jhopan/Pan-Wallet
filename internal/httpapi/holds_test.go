package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/jhopan/PanWallet/internal/report"
	"github.com/jhopan/PanWallet/internal/wallet"
)

type fakeHoldStore struct {
	request wallet.CreateHoldRequest
}

func (store *fakeHoldStore) CreateHold(_ context.Context, request wallet.CreateHoldRequest, _ time.Time) (wallet.Hold, error) {
	store.request = request
	return wallet.Hold{ID: "11111111-1111-1111-1111-111111111111", Product: request.Product, Status: "active"}, nil
}

func (store *fakeHoldStore) CaptureHold(context.Context, wallet.TransitionRequest) (wallet.Hold, error) {
	return wallet.Hold{Status: "captured"}, nil
}

func (store *fakeHoldStore) ReleaseHold(context.Context, wallet.TransitionRequest) (wallet.Hold, error) {
	return wallet.Hold{Status: "released"}, nil
}

func (store *fakeHoldStore) WalletBalance(context.Context, string) (report.OverviewWallet, error) {
	return report.OverviewWallet{}, nil
}

func TestHoldUsesAuthenticatedProductScope(t *testing.T) {
	store := &fakeHoldStore{}
	request := httptest.NewRequest(http.MethodPost, "/v1/holds", strings.NewReader(`{
		"wallet_user_id":"22222222-2222-2222-2222-222222222222",
		"reference_id":"AP-1001",
		"amount":25000,
		"expires_at":"2030-01-01T00:00:00Z"
	}`))
	request.Header.Set("Authorization", "Bearer pw_live_example")
	request.Header.Set("Idempotency-Key", "hold:agenpulsa:AP-1001")
	response := httptest.NewRecorder()

	New(Deps{Auth: fakeAuthenticator{}, Holds: store}).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if store.request.Product != "agenpulsa" {
		t.Fatalf("product = %q, want agenpulsa", store.request.Product)
	}
}

// Wallet diblokir harus ditolak dengan 403, bukan 422 generik.
func TestHoldRejectsBlockedWallet(t *testing.T) {
	store := &blockedHoldStore{}
	request := httptest.NewRequest(http.MethodPost, "/v1/holds", strings.NewReader(`{
		"wallet_user_id":"22222222-2222-2222-2222-222222222222",
		"reference_id":"AP-1","amount":1000,"expires_at":"2030-01-01T00:00:00Z"
	}`))
	request.Header.Set("Authorization", "Bearer pw_live_test")
	request.Header.Set("Idempotency-Key", "hold:AP-1")
	response := httptest.NewRecorder()

	New(Deps{Auth: fakeAuthenticator{}, Holds: store}).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

type blockedHoldStore struct{}

func (blockedHoldStore) CreateHold(context.Context, wallet.CreateHoldRequest, time.Time) (wallet.Hold, error) {
	return wallet.Hold{}, wallet.ErrWalletBlocked
}
func (blockedHoldStore) CaptureHold(context.Context, wallet.TransitionRequest) (wallet.Hold, error) {
	return wallet.Hold{}, nil
}
func (blockedHoldStore) ReleaseHold(context.Context, wallet.TransitionRequest) (wallet.Hold, error) {
	return wallet.Hold{}, nil
}
func (blockedHoldStore) WalletBalance(context.Context, string) (report.OverviewWallet, error) {
	return report.OverviewWallet{}, nil
}

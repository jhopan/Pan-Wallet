package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/report"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

type fakeTransitionStore struct {
	captured wallet.TransitionRequest
	released wallet.TransitionRequest
}

func (store *fakeTransitionStore) CreateHold(context.Context, wallet.CreateHoldRequest, time.Time) (wallet.Hold, error) {
	return wallet.Hold{ID: "hold-1", Status: "active"}, nil
}

func (store *fakeTransitionStore) CaptureHold(_ context.Context, request wallet.TransitionRequest) (wallet.Hold, error) {
	store.captured = request
	return wallet.Hold{ID: request.HoldID, Status: "captured"}, nil
}

func (store *fakeTransitionStore) ReleaseHold(_ context.Context, request wallet.TransitionRequest) (wallet.Hold, error) {
	store.released = request
	return wallet.Hold{ID: request.HoldID, Status: "released"}, nil
}

func (store *fakeTransitionStore) WalletBalance(context.Context, string) (report.OverviewWallet, error) {
	return report.OverviewWallet{WalletUserID: "w1", Balance: 100000, HeldActive: 25000, Available: 75000, Currency: "IDR"}, nil
}

func TestCaptureRequiresAuth(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/holds/hold-1/capture", strings.NewReader("{}"))

	New(Deps{Auth: fakeAuthenticator{}, Holds: &fakeTransitionStore{}}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}

func TestCaptureUsesAuthenticatedProduct(t *testing.T) {
	store := &fakeTransitionStore{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/holds/hold-1/capture", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer pw_live_test")
	request.Header.Set("Idempotency-Key", "capture:AP-1")

	New(Deps{Auth: fakeAuthenticator{}, Holds: store}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", response.Code, response.Body.String())
	}
	if store.captured.HoldID != "hold-1" {
		t.Fatalf("hold id = %q", store.captured.HoldID)
	}
	if store.captured.Product != "agenpulsa" {
		t.Fatalf("product = %q, want dari token", store.captured.Product)
	}
	if store.captured.IdempotencyKey != "capture:AP-1" {
		t.Fatalf("idempotency = %q", store.captured.IdempotencyKey)
	}
}

func TestReleaseUsesAuthenticatedProduct(t *testing.T) {
	store := &fakeTransitionStore{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/holds/hold-2/release", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer pw_live_test")
	request.Header.Set("Idempotency-Key", "release:AP-2")

	New(Deps{Auth: fakeAuthenticator{}, Holds: store}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if store.released.HoldID != "hold-2" || store.released.Product != "agenpulsa" {
		t.Fatalf("release request = %+v", store.released)
	}
}

func TestWalletBalanceReturnsAvailable(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/wallets/w1", nil)
	request.Header.Set("Authorization", "Bearer pw_live_test")

	New(Deps{Auth: fakeAuthenticator{}, Holds: &fakeTransitionStore{}}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{"100000", "25000", "75000"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body tidak memuat %q: %s", want, body)
		}
	}
}

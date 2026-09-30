package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

type fakeHoldStore struct {
	request wallet.CreateHoldRequest
}

func (store *fakeHoldStore) CreateHold(_ context.Context, request wallet.CreateHoldRequest, _ time.Time) (wallet.Hold, error) {
	store.request = request
	return wallet.Hold{ID: "11111111-1111-1111-1111-111111111111", Product: request.Product, Status: "active"}, nil
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

	New(fakeAuthenticator{}, store).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if store.request.Product != "agenpulsa" {
		t.Fatalf("product = %q, want agenpulsa", store.request.Product)
	}
}

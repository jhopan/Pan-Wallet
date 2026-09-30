package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeAuthenticator struct{}

func (fakeAuthenticator) Authenticate(context.Context, string) (string, error) {
	return "agenpulsa", nil
}

func TestBearerTokenRejectsMissingHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/holds", nil)
	if _, err := bearerToken(request); err == nil {
		t.Fatal("bearerToken() error = nil, want unauthorized")
	}
}

func TestBearerTokenReturnsToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/holds", nil)
	request.Header.Set("Authorization", "Bearer pw_live_example")

	token, err := bearerToken(request)
	if err != nil {
		t.Fatalf("bearerToken() error = %v", err)
	}
	if token != "pw_live_example" {
		t.Fatalf("token = %q", token)
	}
}

func TestProductEndpointRejectsUnauthenticatedRequest(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/holds", nil)

	New(fakeAuthenticator{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

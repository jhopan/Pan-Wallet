package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
)

type fakeProductKeyStore struct {
	product string
	err     error
	hash    []byte
}

func (store *fakeProductKeyStore) ProductForTokenHash(_ context.Context, hash []byte) (string, error) {
	store.hash = hash
	return store.product, store.err
}

func TestAuthenticateReturnsScopedProduct(t *testing.T) {
	store := &fakeProductKeyStore{product: "agenpulsa"}
	authenticator := NewProductAuthenticator(store)

	product, err := authenticator.Authenticate(context.Background(), "pw_live_example")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if product != "agenpulsa" {
		t.Fatalf("product = %q", product)
	}
	expected := sha256.Sum256([]byte("pw_live_example"))
	if string(store.hash) != string(expected[:]) {
		t.Fatal("token hash mismatch")
	}
}

func TestAuthenticateHidesStoreFailure(t *testing.T) {
	authenticator := NewProductAuthenticator(&fakeProductKeyStore{err: errors.New("database down")})
	_, err := authenticator.Authenticate(context.Background(), "pw_live_example")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want unauthorized", err)
	}
}

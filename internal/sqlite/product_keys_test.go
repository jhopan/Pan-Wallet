package sqlite

import (
	"context"
	"testing"

	"github.com/jhopan/Pan-Wallet/internal/auth"
)

func TestProductKeyAuthenticatesToProduct(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	token, err := store.CreateProductKey(ctx, "agenpulsa")
	if err != nil {
		t.Fatalf("CreateProductKey: %v", err)
	}
	if len(token) < 16 {
		t.Fatalf("token terlalu pendek: %q", token)
	}

	authenticator := auth.NewProductAuthenticator(store)
	product, err := authenticator.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if product != "agenpulsa" {
		t.Fatalf("product = %q, want agenpulsa", product)
	}
}

func TestWrongTokenRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if _, err := store.CreateProductKey(ctx, "agenpulsa"); err != nil {
		t.Fatalf("CreateProductKey: %v", err)
	}

	authenticator := auth.NewProductAuthenticator(store)
	if _, err := authenticator.Authenticate(ctx, "pw_live_salah"); err == nil {
		t.Fatal("token salah harus ditolak")
	}
}

func TestRotateReplacesOldToken(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	oldToken, err := store.CreateProductKey(ctx, "agenpulsa")
	if err != nil {
		t.Fatalf("CreateProductKey: %v", err)
	}
	newToken, err := store.CreateProductKey(ctx, "agenpulsa")
	if err != nil {
		t.Fatalf("rotasi: %v", err)
	}
	if oldToken == newToken {
		t.Fatal("rotasi harus menghasilkan token berbeda")
	}

	authenticator := auth.NewProductAuthenticator(store)
	if _, err := authenticator.Authenticate(ctx, oldToken); err == nil {
		t.Fatal("token lama harus mati setelah rotasi")
	}
	if _, err := authenticator.Authenticate(ctx, newToken); err != nil {
		t.Fatalf("token baru harus valid, error = %v", err)
	}
}

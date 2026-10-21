package sqlite

import (
	"context"
	"strings"
	"testing"
)

func TestPublicIDIsShortAndUnique(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	first, err := store.CreateWalletUser(ctx, 0)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}
	second, err := store.CreateWalletUser(ctx, 0)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}

	wallets, err := store.ListWallets(ctx, 10)
	if err != nil {
		t.Fatalf("ListWallets: %v", err)
	}
	ids := map[string]bool{}
	for _, wallet := range wallets {
		if wallet.PublicID == "" {
			t.Fatal("public_id tidak boleh kosong")
		}
		if len(wallet.PublicID) != 8 {
			t.Fatalf("public_id = %q, panjang %d, want 8", wallet.PublicID, len(wallet.PublicID))
		}
		if wallet.PublicID != strings.ToUpper(wallet.PublicID) {
			t.Fatalf("public_id harus huruf besar: %q", wallet.PublicID)
		}
		if ids[wallet.PublicID] {
			t.Fatalf("public_id duplikat: %q", wallet.PublicID)
		}
		ids[wallet.PublicID] = true
	}
	if len(ids) != 2 {
		t.Fatalf("jumlah id unik = %d, want 2", len(ids))
	}
	_ = first
	_ = second
}

func TestResolvePublicIDReturnsWallet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	userID, err := store.CreateWalletUser(ctx, 75_000)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}
	wallets, _ := store.ListWallets(ctx, 10)
	publicID := wallets[0].PublicID

	// Cari pakai ID publik, huruf kecil pun boleh.
	resolved, err := store.ResolvePublicID(ctx, strings.ToLower(publicID))
	if err != nil {
		t.Fatalf("ResolvePublicID: %v", err)
	}
	if resolved.WalletUserID != userID {
		t.Fatalf("wallet = %q, want %q", resolved.WalletUserID, userID)
	}
	if resolved.Balance != 75_000 {
		t.Fatalf("balance = %d, want 75000", resolved.Balance)
	}
}

func TestResolvePublicIDRejectsUnknown(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.ResolvePublicID(context.Background(), "ZZZZZZZZ"); err == nil {
		t.Fatal("id tidak dikenal harus error")
	}
}

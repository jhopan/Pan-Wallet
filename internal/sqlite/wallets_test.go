package sqlite

import (
	"context"
	"testing"
)

func TestListWalletsReturnsCreatedUsers(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	first, err := store.CreateWalletUser(ctx, 50_000)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}
	second, err := store.CreateWalletUser(ctx, 0)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}

	wallets, err := store.ListWallets(ctx, 50)
	if err != nil {
		t.Fatalf("ListWallets: %v", err)
	}
	if len(wallets) != 2 {
		t.Fatalf("jumlah wallet = %d, want 2", len(wallets))
	}

	// Terbaru dulu.
	if wallets[0].WalletUserID != second {
		t.Fatalf("wallet terbaru = %q, want %q", wallets[0].WalletUserID, second)
	}
	if wallets[1].WalletUserID != first || wallets[1].Balance != 50_000 {
		t.Fatalf("wallet pertama = %+v", wallets[1])
	}
}

func TestListWalletsShowsStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	userID, err := store.CreateWalletUser(ctx, 10_000)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}
	if err := store.SetWalletStatus(ctx, userID, "blocked"); err != nil {
		t.Fatalf("SetWalletStatus: %v", err)
	}

	wallets, _ := store.ListWallets(ctx, 10)
	if len(wallets) != 1 || wallets[0].Status != "blocked" {
		t.Fatalf("status = %+v", wallets)
	}

	if err := store.SetWalletStatus(ctx, userID, "active"); err != nil {
		t.Fatalf("SetWalletStatus aktif: %v", err)
	}
	wallets, _ = store.ListWallets(ctx, 10)
	if wallets[0].Status != "active" {
		t.Fatalf("status setelah aktif = %q", wallets[0].Status)
	}
}

func TestSetWalletStatusRejectsUnknown(t *testing.T) {
	store := newTestStore(t)
	userID, _ := store.CreateWalletUser(context.Background(), 0)
	if err := store.SetWalletStatus(context.Background(), userID, "hilang"); err == nil {
		t.Fatal("status asing harus ditolak")
	}
}

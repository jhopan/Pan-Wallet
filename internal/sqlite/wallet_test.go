package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jhopan/PanWallet/internal/wallet"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "wallet.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

// seedWallet membuat wallet user dengan saldo awal dan mengembalikan UUID-nya.
func seedWallet(t *testing.T, store *Store, balance int64) string {
	t.Helper()
	id, err := store.CreateWalletUser(context.Background(), balance)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}
	return id
}

func TestCreateWalletUserSetsBalance(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 100_000)

	overview, err := store.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if overview.WalletCount != 1 {
		t.Fatalf("WalletCount = %d, want 1", overview.WalletCount)
	}
	if overview.Wallets[0].WalletUserID != userID {
		t.Fatalf("WalletUserID = %q, want %q", overview.Wallets[0].WalletUserID, userID)
	}
	if overview.Wallets[0].Balance != 100_000 {
		t.Fatalf("Balance = %d, want 100000", overview.Wallets[0].Balance)
	}
	if overview.Wallets[0].Available != 100_000 {
		t.Fatalf("Available = %d, want 100000", overview.Wallets[0].Available)
	}
}

func TestHoldThenCaptureReducesBalanceOnce(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 50_000)
	ctx := context.Background()

	hold, err := store.CreateHold(ctx, wallet.CreateHoldRequest{
		WalletUserID: userID, Product: "agenpulsa", ReferenceID: "AP-1001",
		Amount: 25_000, IdempotencyKey: "hold:AP-1001",
	}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateHold: %v", err)
	}
	if hold.Status != "active" {
		t.Fatalf("status = %q, want active", hold.Status)
	}

	overview, _ := store.Overview(ctx)
	if overview.Wallets[0].Available != 25_000 {
		t.Fatalf("Available setelah hold = %d, want 25000", overview.Wallets[0].Available)
	}

	captured, err := store.CaptureHold(ctx, wallet.TransitionRequest{
		HoldID: hold.ID, Product: "agenpulsa", IdempotencyKey: "capture:AP-1001",
	})
	if err != nil {
		t.Fatalf("CaptureHold: %v", err)
	}
	if captured.Status != "captured" {
		t.Fatalf("status = %q, want captured", captured.Status)
	}

	overview, _ = store.Overview(ctx)
	if overview.Wallets[0].Balance != 25_000 {
		t.Fatalf("Balance setelah capture = %d, want 25000", overview.Wallets[0].Balance)
	}
	if overview.LedgerCount != 1 {
		t.Fatalf("LedgerCount = %d, want 1", overview.LedgerCount)
	}

	// Capture kedua dengan key sama = idempotent, tidak menambah ledger.
	if _, err := store.CaptureHold(ctx, wallet.TransitionRequest{
		HoldID: hold.ID, Product: "agenpulsa", IdempotencyKey: "capture:AP-1001",
	}); err != nil {
		t.Fatalf("capture ulang harus idempotent, error = %v", err)
	}
	overview, _ = store.Overview(ctx)
	if overview.LedgerCount != 1 {
		t.Fatalf("LedgerCount setelah capture ulang = %d, want 1", overview.LedgerCount)
	}
}

func TestHoldRejectsInsufficientBalance(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 10_000)

	_, err := store.CreateHold(context.Background(), wallet.CreateHoldRequest{
		WalletUserID: userID, Product: "agenpulsa", ReferenceID: "AP-2",
		Amount: 50_000, IdempotencyKey: "hold:AP-2",
	}, time.Now().Add(time.Hour))
	if !errors.Is(err, wallet.ErrInsufficientBalance) {
		t.Fatalf("error = %v, want ErrInsufficientBalance", err)
	}
}

func TestCaptureRejectsOtherProductScope(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 50_000)
	ctx := context.Background()

	hold, err := store.CreateHold(ctx, wallet.CreateHoldRequest{
		WalletUserID: userID, Product: "agenpulsa", ReferenceID: "AP-3",
		Amount: 10_000, IdempotencyKey: "hold:AP-3",
	}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateHold: %v", err)
	}

	_, err = store.CaptureHold(ctx, wallet.TransitionRequest{
		HoldID: hold.ID, Product: "tembak-paket", IdempotencyKey: "capture:AP-3",
	})
	if !errors.Is(err, wallet.ErrHoldNotFound) {
		t.Fatalf("error = %v, want ErrHoldNotFound", err)
	}
}

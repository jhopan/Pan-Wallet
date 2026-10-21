package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

func TestLedgerRecordsEveryBalanceChange(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 100_000)
	ctx := context.Background()

	// Top-up manual lewat jalur kredit.
	if _, err := store.CreditWallet(ctx, userID, 25_000, "topup-manual", "TU-1", "topup:TU-1"); err != nil {
		t.Fatalf("CreditWallet: %v", err)
	}

	// Hold lalu capture.
	hold, err := store.CreateHold(ctx, wallet.CreateHoldRequest{
		WalletUserID: userID, Product: "agenpulsa", ReferenceID: "AP-1",
		Amount: 10_000, IdempotencyKey: "hold:AP-1",
	}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateHold: %v", err)
	}
	if _, err := store.CaptureHold(ctx, wallet.TransitionRequest{
		HoldID: hold.ID, Product: "agenpulsa", IdempotencyKey: "capture:AP-1",
	}); err != nil {
		t.Fatalf("CaptureHold: %v", err)
	}

	entries, err := store.LedgerEntries(ctx, userID, 50)
	if err != nil {
		t.Fatalf("LedgerEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("jumlah mutasi = %d, want 2", len(entries))
	}

	// Terbaru dulu: capture (-10000), lalu topup (+25000).
	if entries[0].EntryType != "hold_capture" || entries[0].Amount != -10_000 {
		t.Fatalf("mutasi terbaru = %+v", entries[0])
	}
	if entries[1].EntryType != "topup" || entries[1].Amount != 25_000 {
		t.Fatalf("mutasi kedua = %+v", entries[1])
	}
	if entries[1].BalanceAfter != 125_000 {
		t.Fatalf("balance_after = %d, want 125000", entries[1].BalanceAfter)
	}
}

func TestLedgerIsEmptyForNewWallet(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 5_000)

	entries, err := store.LedgerEntries(context.Background(), userID, 50)
	if err != nil {
		t.Fatalf("LedgerEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("wallet baru harus tanpa mutasi, dapat %d", len(entries))
	}
}

func TestProductKeyListingAndRevoke(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	token, err := store.CreateProductKey(ctx, "agenpulsa")
	if err != nil {
		t.Fatalf("CreateProductKey: %v", err)
	}
	keys, err := store.ListProductKeys(ctx)
	if err != nil {
		t.Fatalf("ListProductKeys: %v", err)
	}
	if len(keys) != 1 || keys[0].Product != "agenpulsa" || !keys[0].Active {
		t.Fatalf("daftar key = %+v", keys)
	}
	if keys[0].TokenPrefix == "" {
		t.Fatal("prefix token harus terlihat untuk identifikasi")
	}
	if keys[0].TokenPrefix == token {
		t.Fatal("token penuh tidak boleh dikembalikan")
	}

	if err := store.RevokeProductKey(ctx, "agenpulsa"); err != nil {
		t.Fatalf("RevokeProductKey: %v", err)
	}
	keys, _ = store.ListProductKeys(ctx)
	if keys[0].Active {
		t.Fatal("key harus nonaktif setelah revoke")
	}
}

package sqlite

import (
	"context"
	"testing"
)

func TestAuditRecordsAdminAction(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	userID, err := store.CreateWalletUser(ctx, 0)
	if err != nil {
		t.Fatalf("CreateWalletUser: %v", err)
	}

	err = store.RecordAudit(ctx, AuditEntry{
		Actor:        "admin",
		Action:       "wallet.block",
		WalletUserID: userID,
		ReferenceID:  userID,
		Details:      "diblokir karena penyalahgunaan",
	})
	if err != nil {
		t.Fatalf("RecordAudit: %v", err)
	}

	entries, err := store.AuditLog(ctx, 10)
	if err != nil {
		t.Fatalf("AuditLog: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("jumlah audit = %d, want 1", len(entries))
	}
	if entries[0].Action != "wallet.block" || entries[0].Actor != "admin" {
		t.Fatalf("audit = %+v", entries[0])
	}
	if entries[0].WalletUserID != userID {
		t.Fatalf("wallet = %q, want %q", entries[0].WalletUserID, userID)
	}
}

func TestAuditNewestFirst(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for _, action := range []string{"pertama", "kedua", "ketiga"} {
		if err := store.RecordAudit(ctx, AuditEntry{Actor: "admin", Action: action, ReferenceID: action}); err != nil {
			t.Fatalf("RecordAudit: %v", err)
		}
	}

	entries, _ := store.AuditLog(ctx, 10)
	if len(entries) != 3 {
		t.Fatalf("jumlah = %d", len(entries))
	}
	if entries[0].Action != "ketiga" {
		t.Fatalf("audit terbaru = %q, want ketiga", entries[0].Action)
	}
}

func TestAdjustBalanceCreditsAndDebits(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	userID, _ := store.CreateWalletUser(ctx, 10_000)

	// Tambah 5.000
	after, err := store.AdjustBalance(ctx, userID, 5_000, "bonus admin", "admin", "ADJ-1")
	if err != nil {
		t.Fatalf("AdjustBalance tambah: %v", err)
	}
	if after != 15_000 {
		t.Fatalf("saldo = %d, want 15000", after)
	}

	// Kurangi 3.000
	after, err = store.AdjustBalance(ctx, userID, -3_000, "koreksi", "admin", "ADJ-2")
	if err != nil {
		t.Fatalf("AdjustBalance kurangi: %v", err)
	}
	if after != 12_000 {
		t.Fatalf("saldo = %d, want 12000", after)
	}

	// Ledger tercatat dua baris adjustment.
	entries, _ := store.LedgerEntries(ctx, userID, 10)
	if len(entries) != 2 {
		t.Fatalf("mutasi = %d, want 2", len(entries))
	}
	if entries[0].EntryType != "adjustment" {
		t.Fatalf("entry type = %q", entries[0].EntryType)
	}
}

func TestAdjustBalanceRejectsNegativeResult(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	userID, _ := store.CreateWalletUser(ctx, 1_000)

	if _, err := store.AdjustBalance(ctx, userID, -5_000, "koreksi", "admin", "ADJ-3"); err == nil {
		t.Fatal("saldo tidak boleh jadi negatif")
	}
}

func TestAdjustBalanceIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	userID, _ := store.CreateWalletUser(ctx, 0)

	first, _ := store.AdjustBalance(ctx, userID, 5_000, "bonus", "admin", "ADJ-SAMA")
	second, _ := store.AdjustBalance(ctx, userID, 5_000, "bonus", "admin", "ADJ-SAMA")

	if first != second {
		t.Fatalf("idempotensi gagal: %d vs %d", first, second)
	}
	entries, _ := store.LedgerEntries(ctx, userID, 10)
	if len(entries) != 1 {
		t.Fatalf("mutasi = %d, want 1 (tidak dobel)", len(entries))
	}
}

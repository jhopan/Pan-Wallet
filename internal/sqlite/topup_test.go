package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestCreateTopupInvoiceClaimsUniqueTotal(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 0)
	ctx := context.Background()

	first, err := store.CreateTopupInvoice(ctx, userID, 25_000, 15*time.Minute)
	if err != nil {
		t.Fatalf("invoice pertama: %v", err)
	}
	if first.Code != 1 || first.Total != 25_001 {
		t.Fatalf("invoice pertama code=%d total=%d, want code=1 total=25001", first.Code, first.Total)
	}

	second, err := store.CreateTopupInvoice(ctx, userID, 25_000, 15*time.Minute)
	if err != nil {
		t.Fatalf("invoice kedua: %v", err)
	}
	if second.Code != 2 || second.Total != 25_002 {
		t.Fatalf("invoice kedua code=%d total=%d, want code=2 total=25002", second.Code, second.Total)
	}
	if first.Total == second.Total {
		t.Fatal("dua invoice tidak boleh punya total sama")
	}
}

func TestMarkTopupPaidCreditsOnce(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 10_000)
	ctx := context.Background()

	invoice, err := store.CreateTopupInvoice(ctx, userID, 50_000, 15*time.Minute)
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}

	paid, err := store.MarkTopupPaid(ctx, invoice.ID, "manual-admin")
	if err != nil {
		t.Fatalf("MarkTopupPaid: %v", err)
	}
	if paid.Status != "paid" {
		t.Fatalf("status = %q, want paid", paid.Status)
	}

	overview, _ := store.Overview(ctx)
	if overview.Wallets[0].Balance != 60_001 {
		t.Fatalf("Balance = %d, want 60001 (10000 + 50001)", overview.Wallets[0].Balance)
	}
	if overview.LedgerCount != 1 {
		t.Fatalf("LedgerCount = %d, want 1", overview.LedgerCount)
	}

	// Klik lunas kedua: tidak boleh menambah saldo/ledger.
	if _, err := store.MarkTopupPaid(ctx, invoice.ID, "manual-admin"); err != nil {
		t.Fatalf("pelunasan ulang harus idempotent: %v", err)
	}
	overview, _ = store.Overview(ctx)
	if overview.Wallets[0].Balance != 60_001 {
		t.Fatalf("Balance setelah pelunasan ulang = %d, want 60001", overview.Wallets[0].Balance)
	}
	if overview.LedgerCount != 1 {
		t.Fatalf("LedgerCount setelah pelunasan ulang = %d, want 1", overview.LedgerCount)
	}
}

func TestTopupInvoiceLookupAndList(t *testing.T) {
	store := newTestStore(t)
	userID := seedWallet(t, store, 0)
	ctx := context.Background()

	created, err := store.CreateTopupInvoice(ctx, userID, 20_000, 15*time.Minute)
	if err != nil {
		t.Fatalf("buat invoice: %v", err)
	}

	found, err := store.TopupInvoice(ctx, created.ID)
	if err != nil {
		t.Fatalf("TopupInvoice: %v", err)
	}
	if found.Total != created.Total || found.Status != "pending" {
		t.Fatalf("invoice tidak cocok: %+v", found)
	}

	list, err := store.RecentTopupInvoices(ctx, 10)
	if err != nil {
		t.Fatalf("RecentTopupInvoices: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("daftar invoice tidak memuat invoice baru: %+v", list)
	}
}

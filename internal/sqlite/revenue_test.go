package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestRevenueSummaryCountsOnlyPaid(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	userID, _ := store.CreateWalletUser(ctx, 0)

	// Satu invoice lunas, satu masih pending.
	paidInvoice := seedTopupInvoice(t, store, userID, 50_000, "paid")
	seedTopupInvoice(t, store, userID, 25_000, "pending")

	summary, err := store.RevenueSummary(ctx)
	if err != nil {
		t.Fatalf("RevenueSummary: %v", err)
	}

	if summary.TotalRevenue != paidInvoice {
		t.Fatalf("pendapatan = %d, want %d (hanya lunas)", summary.TotalRevenue, paidInvoice)
	}
	if summary.PaidCount != 1 {
		t.Fatalf("jumlah lunas = %d, want 1", summary.PaidCount)
	}
	if summary.PendingCount != 1 {
		t.Fatalf("jumlah pending = %d, want 1", summary.PendingCount)
	}
}

func TestRevenueByDayGroupsCorrectly(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	userID, _ := store.CreateWalletUser(ctx, 0)

	seedTopupInvoice(t, store, userID, 10_000, "paid")
	seedTopupInvoice(t, store, userID, 20_000, "paid")

	days, err := store.RevenueByDay(ctx, 7)
	if err != nil {
		t.Fatalf("RevenueByDay: %v", err)
	}
	if len(days) == 0 {
		t.Fatal("harus ada minimal satu hari")
	}
	if days[0].Count != 2 {
		t.Fatalf("jumlah transaksi hari ini = %d, want 2", days[0].Count)
	}
	if days[0].Revenue < 30_000 {
		t.Fatalf("pendapatan hari ini = %d, want >= 30000", days[0].Revenue)
	}
}

func TestTopupStatsTotals(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	userID, _ := store.CreateWalletUser(ctx, 0)

	seedTopupInvoice(t, store, userID, 100_000, "paid")
	seedTopupInvoice(t, store, userID, 50_000, "expired")

	stats, err := store.TopupStats(ctx)
	if err != nil {
		t.Fatalf("TopupStats: %v", err)
	}
	if stats.PaidCount != 1 || stats.ExpiredCount != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

// seedTopupInvoice menyisipkan invoice lunas/pending langsung ke database uji.
func seedTopupInvoice(t *testing.T, store *Store, walletUserID string, price int64, status string) int64 {
	t.Helper()
	ctx := context.Background()
	total := price + 1
	now := time.Now().Unix()

	var paidAt any
	if status == "paid" {
		paidAt = now
	}
	_, err := store.db.ExecContext(ctx, `
		insert into topup_invoices (id, wallet_user_id, paypan_invoice_id, price, code, total, status, source, expires_at, paid_at, created_at)
		values (?, ?, ?, ?, 1, ?, ?, 'paypan-webhook', ?, ?, ?)`,
		newID(), walletUserID, newID(), price, total, status, now+900, paidAt, now)
	if err != nil {
		t.Fatalf("seed invoice: %v", err)
	}
	return total
}

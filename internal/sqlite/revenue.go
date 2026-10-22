package sqlite

import (
	"context"
	"fmt"
	"time"
)

// RevenueSummary adalah ringkasan pendapatan dari invoice top-up.
type RevenueSummary struct {
	TotalRevenue int64
	PaidCount    int
	PendingCount int
	ExpiredCount int
	TodayRevenue int64
	TodayCount   int
}

// RevenueDay adalah pendapatan satu hari.
type RevenueDay struct {
	Date    string
	Count   int
	Revenue int64
}

// RevenueSummary menghitung ringkasan pendapatan. Hanya status 'paid' yang
// dihitung sebagai pendapatan nyata.
func (store *Store) RevenueSummary(ctx context.Context) (RevenueSummary, error) {
	var summary RevenueSummary

	if err := store.db.QueryRowContext(ctx, `
		select coalesce(sum(total),0), count(*) from topup_invoices where status = 'paid'`,
	).Scan(&summary.TotalRevenue, &summary.PaidCount); err != nil {
		return RevenueSummary{}, fmt.Errorf("hitung pendapatan: %w", err)
	}

	if err := store.db.QueryRowContext(ctx,
		`select count(*) from topup_invoices where status = 'pending'`,
	).Scan(&summary.PendingCount); err != nil {
		return RevenueSummary{}, fmt.Errorf("hitung pending: %w", err)
	}

	if err := store.db.QueryRowContext(ctx,
		`select count(*) from topup_invoices where status = 'expired'`,
	).Scan(&summary.ExpiredCount); err != nil {
		return RevenueSummary{}, fmt.Errorf("hitung expired: %w", err)
	}

	startOfDay := time.Now().Truncate(24 * time.Hour).Unix()
	if err := store.db.QueryRowContext(ctx, `
		select coalesce(sum(total),0), count(*) from topup_invoices
		where status = 'paid' and coalesce(paid_at, created_at) >= ?`, startOfDay,
	).Scan(&summary.TodayRevenue, &summary.TodayCount); err != nil {
		return RevenueSummary{}, fmt.Errorf("hitung hari ini: %w", err)
	}
	return summary, nil
}

// RevenueByDay menghitung pendapatan per hari untuk N hari terakhir.
func (store *Store) RevenueByDay(ctx context.Context, days int) ([]RevenueDay, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days).Unix()

	rows, err := store.db.QueryContext(ctx, `
		select date(coalesce(paid_at, created_at), 'unixepoch', 'localtime') as hari,
		       count(*), coalesce(sum(total),0)
		from topup_invoices
		where status = 'paid' and coalesce(paid_at, created_at) >= ?
		group by hari order by hari desc`, since)
	if err != nil {
		return nil, fmt.Errorf("pendapatan harian: %w", err)
	}
	defer rows.Close()

	var result []RevenueDay
	for rows.Next() {
		var day RevenueDay
		if err := rows.Scan(&day.Date, &day.Count, &day.Revenue); err != nil {
			return nil, fmt.Errorf("baca harian: %w", err)
		}
		result = append(result, day)
	}
	return result, rows.Err()
}

// TopupStats adalah hitungan invoice per status.
type TopupStats struct {
	PaidCount      int
	PendingCount   int
	ExpiredCount   int
	CancelledCount int
}

// TopupStats menghitung jumlah invoice per status.
func (store *Store) TopupStats(ctx context.Context) (TopupStats, error) {
	var stats TopupStats
	err := store.db.QueryRowContext(ctx, `
		select
			sum(case when status='paid' then 1 else 0 end),
			sum(case when status='pending' then 1 else 0 end),
			sum(case when status='expired' then 1 else 0 end),
			sum(case when status='cancelled' then 1 else 0 end)
		from topup_invoices`,
	).Scan(&stats.PaidCount, &stats.PendingCount, &stats.ExpiredCount, &stats.CancelledCount)
	if err != nil {
		return TopupStats{}, fmt.Errorf("statistik topup: %w", err)
	}
	return stats, nil
}

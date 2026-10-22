package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

// RevenueSummary menghitung ringkasan pendapatan.
func (store *Store) RevenueSummary(ctx context.Context) (sqlite.RevenueSummary, error) {
	var summary sqlite.RevenueSummary
	err := store.pool.QueryRow(ctx, `
		select
			coalesce(sum(total) filter (where status='paid'),0),
			count(*) filter (where status='paid'),
			count(*) filter (where status='pending'),
			count(*) filter (where status='expired'),
			coalesce(sum(total) filter (where status='paid' and coalesce(paid_at, created_at) >= date_trunc('day', now())),0),
			count(*) filter (where status='paid' and coalesce(paid_at, created_at) >= date_trunc('day', now()))
		from topup_invoices`,
	).Scan(&summary.TotalRevenue, &summary.PaidCount, &summary.PendingCount,
		&summary.ExpiredCount, &summary.TodayRevenue, &summary.TodayCount)
	if err != nil {
		return sqlite.RevenueSummary{}, fmt.Errorf("hitung pendapatan: %w", err)
	}
	return summary, nil
}

// RevenueByDay menghitung pendapatan per hari.
func (store *Store) RevenueByDay(ctx context.Context, days int) ([]sqlite.RevenueDay, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	rows, err := store.pool.Query(ctx, `
		select to_char(coalesce(paid_at, created_at), 'YYYY-MM-DD') as hari,
		       count(*), coalesce(sum(total),0)
		from topup_invoices
		where status = 'paid' and coalesce(paid_at, created_at) >= $1
		group by hari order by hari desc`, since)
	if err != nil {
		return nil, fmt.Errorf("pendapatan harian: %w", err)
	}
	defer rows.Close()

	var result []sqlite.RevenueDay
	for rows.Next() {
		var day sqlite.RevenueDay
		if err := rows.Scan(&day.Date, &day.Count, &day.Revenue); err != nil {
			return nil, fmt.Errorf("baca harian: %w", err)
		}
		result = append(result, day)
	}
	return result, rows.Err()
}

// TopupStats menghitung invoice per status.
func (store *Store) TopupStats(ctx context.Context) (sqlite.TopupStats, error) {
	var stats sqlite.TopupStats
	err := store.pool.QueryRow(ctx, `
		select
			count(*) filter (where status='paid'),
			count(*) filter (where status='pending'),
			count(*) filter (where status='expired'),
			count(*) filter (where status='cancelled')
		from topup_invoices`,
	).Scan(&stats.PaidCount, &stats.PendingCount, &stats.ExpiredCount, &stats.CancelledCount)
	if err != nil {
		return sqlite.TopupStats{}, fmt.Errorf("statistik topup: %w", err)
	}
	return stats, nil
}

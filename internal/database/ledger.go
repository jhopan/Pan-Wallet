package database

import (
	"context"
	"fmt"

	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

// LedgerEntries mengambil mutasi saldo terbaru untuk satu wallet user.
func (store *Store) LedgerEntries(ctx context.Context, walletUserID string, limit int) ([]sqlite.LedgerEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := store.pool.Query(ctx, `
		select l.id::text, l.entry_type, l.amount, l.balance_after, l.product, l.reference_id, l.idempotency_key, l.created_at
		from wallet_ledger l join wallets w on w.id = l.wallet_id
		where w.wallet_user_id = $1::uuid
		order by l.created_at desc limit $2`, walletUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("baca mutasi: %w", err)
	}
	defer rows.Close()

	var entries []sqlite.LedgerEntry
	for rows.Next() {
		var entry sqlite.LedgerEntry
		if err := rows.Scan(&entry.ID, &entry.EntryType, &entry.Amount, &entry.BalanceAfter,
			&entry.Product, &entry.ReferenceID, &entry.IdempotencyKey, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("baca mutasi: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

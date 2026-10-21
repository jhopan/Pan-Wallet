package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// LedgerEntry adalah satu baris mutasi saldo.
type LedgerEntry struct {
	ID             string
	EntryType      string
	Amount         int64
	BalanceAfter   int64
	Product        string
	ReferenceID    string
	IdempotencyKey string
	CreatedAt      time.Time
}

// ProductKey adalah ringkasan API key produk (tanpa token asli).
type ProductKey struct {
	Product     string
	TokenPrefix string
	Active      bool
	CreatedAt   time.Time
	RotatedAt   time.Time
}

// LedgerEntries mengambil mutasi saldo terbaru untuk satu wallet user.
func (store *Store) LedgerEntries(ctx context.Context, walletUserID string, limit int) ([]LedgerEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := store.db.QueryContext(ctx, `
		select l.id, l.entry_type, l.amount, l.balance_after, l.product, l.reference_id, l.idempotency_key, l.created_at
		from wallet_ledger l join wallets w on w.id = l.wallet_id
		where w.wallet_user_id = ?
		order by l.created_at desc, l.rowid desc limit ?`, walletUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("baca mutasi: %w", err)
	}
	defer rows.Close()

	var entries []LedgerEntry
	for rows.Next() {
		var entry LedgerEntry
		var createdAt int64
		if err := rows.Scan(&entry.ID, &entry.EntryType, &entry.Amount, &entry.BalanceAfter,
			&entry.Product, &entry.ReferenceID, &entry.IdempotencyKey, &createdAt); err != nil {
			return nil, fmt.Errorf("baca mutasi: %w", err)
		}
		entry.CreatedAt = time.Unix(createdAt, 0)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// CreditWallet menambah saldo lewat jalur admin (top-up manual / koreksi).
// Selalu mencatat ledger; idempotency key mencegah kredit ganda.
func (store *Store) CreditWallet(ctx context.Context, walletUserID string, amount int64, product, referenceID, idempotencyKey string) (int64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("nominal kredit harus lebih dari nol")
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("mulai kredit: %w", err)
	}
	defer tx.Rollback()

	var existing int
	if err := tx.QueryRowContext(ctx,
		`select count(*) from wallet_ledger where idempotency_key = ?`, idempotencyKey).Scan(&existing); err != nil {
		return 0, fmt.Errorf("cek idempotensi: %w", err)
	}
	if existing > 0 {
		var balance int64
		tx.QueryRowContext(ctx, `select balance from wallets where wallet_user_id = ?`, walletUserID).Scan(&balance)
		return balance, nil
	}

	var walletID string
	var balance int64
	err = tx.QueryRowContext(ctx,
		`select id, balance from wallets where wallet_user_id = ?`, walletUserID).Scan(&walletID, &balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvoiceNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("baca wallet: %w", err)
	}

	balance += amount
	if _, err := tx.ExecContext(ctx,
		`update wallets set balance = ?, updated_at = ? where id = ?`, balance, nowUnix(), walletID); err != nil {
		return 0, fmt.Errorf("kredit saldo: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		insert into wallet_ledger (id, wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key, created_at)
		values (?, ?, ?, ?, 'topup', ?, ?, ?, ?)`,
		newID(), walletID, amount, balance, product, referenceID, idempotencyKey, nowUnix()); err != nil {
		return 0, fmt.Errorf("catat ledger: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("simpan kredit: %w", err)
	}
	return balance, nil
}

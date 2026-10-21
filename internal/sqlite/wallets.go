package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// WalletRow adalah satu baris daftar wallet untuk panel.
type WalletRow struct {
	WalletUserID string
	PublicID     string
	Status       string
	Balance      int64
	HeldActive   int64
	Available    int64
	Currency     string
	CreatedAt    time.Time
}

// ListWallets mengambil daftar wallet terbaru.
func (store *Store) ListWallets(ctx context.Context, limit int) ([]WalletRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.db.QueryContext(ctx, `
		select u.id, u.public_id, u.status, w.balance, w.currency, u.created_at,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallet_users u join wallets w on w.wallet_user_id = u.id
		order by u.created_at desc, u.rowid desc limit ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("daftar wallet: %w", err)
	}
	defer rows.Close()

	var wallets []WalletRow
	for rows.Next() {
		var row WalletRow
		var createdAt int64
		if err := rows.Scan(&row.WalletUserID, &row.PublicID, &row.Status, &row.Balance, &row.Currency, &createdAt, &row.HeldActive); err != nil {
			return nil, fmt.Errorf("baca wallet: %w", err)
		}
		row.Available = row.Balance - row.HeldActive
		row.CreatedAt = time.Unix(createdAt, 0)
		wallets = append(wallets, row)
	}
	return wallets, rows.Err()
}

// SetWalletStatus mengubah status wallet user: active atau blocked.
// User blocked tidak boleh bertransaksi.
func (store *Store) SetWalletStatus(ctx context.Context, walletUserID, status string) error {
	if status != "active" && status != "blocked" {
		return fmt.Errorf("status harus active atau blocked")
	}
	result, err := store.db.ExecContext(ctx,
		`update wallet_users set status = ? where id = ?`, status, walletUserID)
	if err != nil {
		return fmt.Errorf("ubah status wallet: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("wallet user tidak ditemukan")
	}
	return nil
}

// WalletStatus membaca status satu wallet user.
func (store *Store) WalletStatus(ctx context.Context, walletUserID string) (string, error) {
	var status string
	err := store.db.QueryRowContext(ctx,
		`select status from wallet_users where id = ?`, walletUserID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvoiceNotFound
	}
	if err != nil {
		return "", fmt.Errorf("baca status: %w", err)
	}
	return status, nil
}

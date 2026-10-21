package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

// ListWallets mengambil daftar wallet terbaru.
func (store *Store) ListWallets(ctx context.Context, limit int) ([]sqlite.WalletRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.pool.Query(ctx, `
		select u.id::text, u.status, w.balance, w.currency, u.created_at,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallet_users u join wallets w on w.wallet_user_id = u.id
		order by u.created_at desc limit $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("daftar wallet: %w", err)
	}
	defer rows.Close()

	var wallets []sqlite.WalletRow
	for rows.Next() {
		var row sqlite.WalletRow
		if err := rows.Scan(&row.WalletUserID, &row.Status, &row.Balance, &row.Currency, &row.CreatedAt, &row.HeldActive); err != nil {
			return nil, fmt.Errorf("baca wallet: %w", err)
		}
		row.Available = row.Balance - row.HeldActive
		wallets = append(wallets, row)
	}
	return wallets, rows.Err()
}

// CreateWalletUser membuat wallet user beserta wallet.
func (store *Store) CreateWalletUser(ctx context.Context, balance int64) (string, error) {
	if balance < 0 {
		return "", fmt.Errorf("saldo awal tidak boleh negatif")
	}
	var userID string
	err := store.pool.QueryRow(ctx, `
		with created as (insert into wallet_users default values returning id)
		insert into wallets (wallet_user_id, balance) select id, $1 from created returning wallet_user_id::text`, balance,
	).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("buat wallet user: %w", err)
	}
	return userID, nil
}

// SetWalletStatus mengubah status wallet user.
func (store *Store) SetWalletStatus(ctx context.Context, walletUserID, status string) error {
	if status != "active" && status != "blocked" {
		return fmt.Errorf("status harus active atau blocked")
	}
	result, err := store.pool.Exec(ctx,
		`update wallet_users set status = $2 where id = $1::uuid`, walletUserID, status)
	if err != nil {
		return fmt.Errorf("ubah status wallet: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("wallet user tidak ditemukan")
	}
	return nil
}

// WalletStatus membaca status satu wallet user.
func (store *Store) WalletStatus(ctx context.Context, walletUserID string) (string, error) {
	var status string
	err := store.pool.QueryRow(ctx,
		`select status from wallet_users where id = $1::uuid`, walletUserID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvoiceNotFound
	}
	if err != nil {
		return "", fmt.Errorf("baca status: %w", err)
	}
	return status, nil
}

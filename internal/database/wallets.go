package database

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/PanWallet/internal/sqlite"
)

// ListWallets mengambil daftar wallet terbaru.
func (store *Store) ListWallets(ctx context.Context, limit int) ([]sqlite.WalletRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.pool.Query(ctx, `
		select u.id::text, u.public_id, u.status, w.balance, w.currency, u.created_at,
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
		if err := rows.Scan(&row.WalletUserID, &row.PublicID, &row.Status, &row.Balance, &row.Currency, &row.CreatedAt, &row.HeldActive); err != nil {
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
		with created as (
			insert into wallet_users (public_id) values ($2) returning id
		)
		insert into wallets (wallet_user_id, balance) select id, $1 from created returning wallet_user_id::text`,
		balance, generatePublicID(),
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

// generatePublicID membuat ID pendek 8 karakter tanpa huruf ambigu.
func generatePublicID() string {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	var builder strings.Builder
	for _, b := range buffer {
		builder.WriteByte(alphabet[int(b)%len(alphabet)])
	}
	return builder.String()
}

// ResolvePublicID mencari wallet lewat public_id (case-insensitive).
func (store *Store) ResolvePublicID(ctx context.Context, publicID string) (sqlite.WalletRow, error) {
	normalized := strings.ToUpper(strings.TrimSpace(publicID))
	var row sqlite.WalletRow
	err := store.pool.QueryRow(ctx, `
		select u.id::text, u.public_id, u.status, w.balance, w.currency, u.created_at,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallet_users u join wallets w on w.wallet_user_id = u.id
		where u.public_id = $1`, normalized,
	).Scan(&row.WalletUserID, &row.PublicID, &row.Status, &row.Balance, &row.Currency, &row.CreatedAt, &row.HeldActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlite.WalletRow{}, ErrInvoiceNotFound
	}
	if err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("cari public id: %w", err)
	}
	row.Available = row.Balance - row.HeldActive
	return row, nil
}

// PublicIDByWalletUserID mencari public_id dari UUID internal wallet user.
func (store *Store) PublicIDByWalletUserID(ctx context.Context, walletUserID string) (string, error) {
	var publicID string
	err := store.pool.QueryRow(ctx,
		`select public_id from wallet_users where id = $1`, walletUserID).Scan(&publicID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvoiceNotFound
	}
	if err != nil {
		return "", fmt.Errorf("cari public id wallet: %w", err)
	}
	return publicID, nil
}

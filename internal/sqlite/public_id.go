package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Huruf yang dipakai public_id. Sengaja tanpa 0/O/1/I/L supaya tidak salah baca
// saat pelanggan mengetik atau menyebutkannya lewat chat.
const publicIDAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const publicIDLength = 8

// generatePublicID membuat ID pendek acak 8 karakter.
func generatePublicID() (string, error) {
	buffer := make([]byte, publicIDLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("buat public id: %w", err)
	}
	var builder strings.Builder
	for _, b := range buffer {
		builder.WriteByte(publicIDAlphabet[int(b)%len(publicIDAlphabet)])
	}
	return builder.String(), nil
}

// ResolvePublicID mencari wallet lewat public_id (case-insensitive).
func (store *Store) ResolvePublicID(ctx context.Context, publicID string) (WalletRow, error) {
	normalized := strings.ToUpper(strings.TrimSpace(publicID))
	if normalized == "" {
		return WalletRow{}, ErrInvoiceNotFound
	}

	var row WalletRow
	var createdAt int64
	err := store.db.QueryRowContext(ctx, `
		select u.id, u.public_id, u.status, w.balance, w.currency, u.created_at,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallet_users u join wallets w on w.wallet_user_id = u.id
		where u.public_id = ?`, normalized,
	).Scan(&row.WalletUserID, &row.PublicID, &row.Status, &row.Balance, &row.Currency, &createdAt, &row.HeldActive)
	if errors.Is(err, sql.ErrNoRows) {
		return WalletRow{}, ErrInvoiceNotFound
	}
	if err != nil {
		return WalletRow{}, fmt.Errorf("cari public id: %w", err)
	}
	row.CreatedAt = time.Unix(createdAt, 0)
	row.Available = row.Balance - row.HeldActive
	return row, nil
}

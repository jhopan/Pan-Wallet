package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ProductForTokenHash mencari produk dari hash token yang aktif.
func (store *Store) ProductForTokenHash(ctx context.Context, tokenHash []byte) (string, error) {
	var product string
	err := store.db.QueryRowContext(ctx,
		`select product from product_api_keys where token_hash = ? and active = 1`, tokenHash).Scan(&product)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("product key tidak ditemukan")
	}
	if err != nil {
		return "", fmt.Errorf("cek product key: %w", err)
	}
	return product, nil
}

// CreateProductKey membuat API key baru untuk sebuah produk dan mengembalikan
// token mentah SEKALI SAJA (yang disimpan hanya hash SHA-256).
func (store *Store) CreateProductKey(ctx context.Context, product string) (string, error) {
	product = strings.TrimSpace(product)
	if product == "" {
		return "", fmt.Errorf("nama produk wajib diisi")
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(token))

	if _, err := store.db.ExecContext(ctx, `
		insert into product_api_keys (id, product, token_hash, active, created_at, rotated_at)
		values (?, ?, ?, 1, ?, ?)
		on conflict (product) do update set token_hash = excluded.token_hash, active = 1, rotated_at = excluded.rotated_at`,
		newID(), product, hash[:], nowUnix(), nowUnix()); err != nil {
		return "", fmt.Errorf("simpan product key: %w", err)
	}
	return token, nil
}

// generateToken membuat token acak berformat pw_live_<24 hex>.
func generateToken() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("buat token: %w", err)
	}
	return "pw_live_" + hex.EncodeToString(buffer), nil
}

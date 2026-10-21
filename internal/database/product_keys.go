package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (store *Store) ProductForTokenHash(ctx context.Context, tokenHash []byte) (string, error) {
	var product string
	err := store.pool.QueryRow(ctx,
		`select product from product_api_keys where token_hash = $1 and active = true`, tokenHash,
	).Scan(&product)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("product key tidak ditemukan")
	}
	if err != nil {
		return "", fmt.Errorf("cek product key: %w", err)
	}
	return product, nil
}

// ListProductKeys menampilkan API key produk tanpa membocorkan token asli.
func (store *Store) ListProductKeys(ctx context.Context) ([]sqlite.ProductKey, error) {
	rows, err := store.pool.Query(ctx, `
		select product, token_hash, active, created_at, rotated_at
		from product_api_keys order by product`)
	if err != nil {
		return nil, fmt.Errorf("daftar product key: %w", err)
	}
	defer rows.Close()

	var keys []sqlite.ProductKey
	for rows.Next() {
		var key sqlite.ProductKey
		var hash []byte
		if err := rows.Scan(&key.Product, &hash, &key.Active, &key.CreatedAt, &key.RotatedAt); err != nil {
			return nil, fmt.Errorf("baca product key: %w", err)
		}
		key.TokenPrefix = fmt.Sprintf("%x", hash[:3])
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// CreateProductKey membuat atau merotasi key produk; token mentah dikembalikan sekali.
func (store *Store) CreateProductKey(ctx context.Context, product string) (string, error) {
	product = strings.TrimSpace(product)
	if product == "" {
		return "", fmt.Errorf("nama produk wajib diisi")
	}
	token, err := generateProductToken()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(token))
	if _, err := store.pool.Exec(ctx, `
		insert into product_api_keys (product, token_hash, active, rotated_at)
		values ($1, $2, true, now())
		on conflict (product) do update set token_hash = excluded.token_hash, active = true, rotated_at = now()`,
		product, hash[:]); err != nil {
		return "", fmt.Errorf("simpan product key: %w", err)
	}
	return token, nil
}

// RevokeProductKey menonaktifkan key produk.
func (store *Store) RevokeProductKey(ctx context.Context, product string) error {
	result, err := store.pool.Exec(ctx,
		`update product_api_keys set active = false, rotated_at = now() where product = $1`, product)
	if err != nil {
		return fmt.Errorf("revoke product key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("produk %s tidak ditemukan", product)
	}
	return nil
}

func generateProductToken() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("buat token: %w", err)
	}
	return "pw_live_" + hex.EncodeToString(buffer), nil
}

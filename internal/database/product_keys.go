package database

import (
	"context"
	"errors"
	"fmt"

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

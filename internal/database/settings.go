package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Setting membaca satu nilai konfigurasi. Kosong bila belum ada.
func (store *Store) Setting(ctx context.Context, key string) (string, error) {
	var value string
	err := store.pool.QueryRow(ctx, `select value from app_settings where key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("baca setting %s: %w", key, err)
	}
	return value, nil
}

// SetSetting menyimpan nilai konfigurasi (upsert).
func (store *Store) SetSetting(ctx context.Context, key, value string) error {
	if _, err := store.pool.Exec(ctx, `
		insert into app_settings (key, value, updated_at) values ($1, $2, now())
		on conflict (key) do update set value = excluded.value, updated_at = now()`,
		key, value); err != nil {
		return fmt.Errorf("simpan setting %s: %w", key, err)
	}
	return nil
}

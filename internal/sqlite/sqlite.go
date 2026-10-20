// Package sqlite menyediakan penyimpanan Pan-Wallet berbasis SQLite: satu file,
// tanpa server database. Semua penulisan saldo berjalan serial (satu koneksi)
// sehingga invarian saldo tetap terjaga tanpa row lock seperti PostgreSQL.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store adalah handle database SQLite untuk seluruh operasi wallet.
type Store struct {
	db *sql.DB
}

// Open membuka (atau membuat) database di path dan menerapkan skema.
func Open(ctx context.Context, path string) (*Store, error) {
	if directory := filepath.Dir(path); directory != "" && directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return nil, fmt.Errorf("buat direktori database: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("buka SQLite: %w", err)
	}
	// Satu koneksi: penulisan saldo jadi serial, hilangkan SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping SQLite: %w", err)
	}

	store := &Store{db: db}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("terapkan skema: %w", err)
	}
	return store, nil
}

// Close menutup koneksi database.
func (store *Store) Close() {
	store.db.Close()
}

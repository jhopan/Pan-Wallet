package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenCreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.db")

	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	var tables int
	if err := store.db.QueryRow(`select count(*) from sqlite_master where type='table'
		and name in ('wallets','wallet_holds','wallet_ledger','topup_invoices','product_api_keys')`).Scan(&tables); err != nil {
		t.Fatalf("query tables: %v", err)
	}
	if tables != 5 {
		t.Fatalf("tabel inti = %d, want 5", tables)
	}
}

func TestOpenCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "wallet.db")

	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open() harus membuat direktori induk, error = %v", err)
	}
	defer store.Close()
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.db")

	first, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open pertama: %v", err)
	}
	first.Close()

	second, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open kedua harus aman, error = %v", err)
	}
	second.Close()
}

package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("buat pool PostgreSQL: %w", err)
	}

	store := &Store{pool: pool}
	if err := store.pool.Ping(ctx); err != nil {
		store.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if err := store.ApplyMigrations(ctx); err != nil {
		store.Close()
		return nil, err
	}
	return store, nil
}

func (store *Store) Close() {
	store.pool.Close()
}

func (store *Store) ApplyMigrations(ctx context.Context) error {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai migrasi: %w", err)
	}
	defer transaction.Rollback(ctx)

	if _, err := transaction.Exec(ctx, `
		create table if not exists schema_migrations (
			name text primary key,
			applied_at timestamptz not null default now()
		)`); err != nil {
		return fmt.Errorf("siapkan catatan migrasi: %w", err)
	}

	for _, migration := range Migrations() {
		var applied bool
		if err := transaction.QueryRow(ctx,
			`select exists(select 1 from schema_migrations where name = $1)`, migration.Name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("cek migrasi %s: %w", migration.Name, err)
		}
		if applied {
			continue
		}
		if _, err := transaction.Exec(ctx, migration.SQL); err != nil {
			return fmt.Errorf("jalankan migrasi %s: %w", migration.Name, err)
		}
		if _, err := transaction.Exec(ctx,
			`insert into schema_migrations (name) values ($1)`, migration.Name,
		); err != nil {
			return fmt.Errorf("catat migrasi %s: %w", migration.Name, err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("simpan migrasi: %w", err)
	}
	return nil
}

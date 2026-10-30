package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// migrations berisi perbaikan skema untuk database yang SUDAH ADA.
//
// Skema utama (schema.sql) memakai CREATE TABLE IF NOT EXISTS, jadi kalau
// sebuah tabel sudah ada, definisi BARU di schema.sql TIDAK ikut diterapkan.
// Migration di bawah menutup celah itu.
//
// Setiap migration harus:
//   - idempoten (aman dijalankan berkali-kali)
//   - tidak menghapus data
//   - punya nomor urut supaya urutannya jelas

// migration adalah satu perbaikan skema.
type migration struct {
	// Name identitas singkat, dipakai pesan log.
	Name string
	// SQL adalah pernyataan yang dijalankan.
	SQL string
}

// migrations adalah daftar perbaikan skema, dijalankan berurutan.
var migrations = []migration{
	{
		// Status invoice "refunded" baru dipakai saat PayPan dikonfirmasi
		// refund. Constraint lama hanya mengizinkan pending/paid/expired/
		// cancelled, jadi update refunded gagal dengan CHECK constraint.
		//
		// SQLite tidak bisa menambah nilai ke CHECK constraint yang sudah
		// ada, jadi tabel harus dibangun ulang: salin data ke tabel baru
		// dengan constraint yang benar, lalu ganti.
		Name: "topup_invoices_allow_refunded",
		SQL: `
create table if not exists topup_invoices_migrated (
  id text primary key,
  wallet_user_id text not null,
  paypan_invoice_id text,
  price integer not null,
  code integer not null,
  total integer not null,
  status text not null default 'pending' check (status in ('pending','paid','expired','cancelled','refunded')),
  source text not null default '',
  created_at integer not null,
  expires_at integer not null,
  paid_at integer
);
insert into topup_invoices_migrated (id, wallet_user_id, paypan_invoice_id, price, code, total, status, source, created_at, expires_at, paid_at)
  select id, wallet_user_id, paypan_invoice_id, price, code, total, status, source, created_at, expires_at, paid_at
  from topup_invoices where id not in (select id from topup_invoices_migrated);
drop table topup_invoices;
alter table topup_invoices_migrated rename to topup_invoices;
create index if not exists idx_topup_wallet on topup_invoices(wallet_user_id);
create index if not exists idx_topup_paypan on topup_invoices(paypan_invoice_id);
create unique index if not exists idx_topup_paypan_unique on topup_invoices(paypan_invoice_id) where paypan_invoice_id is not null;
`,
	},
}

// applyMigrations menjalankan perbaikan skema pada database yang sudah ada.
//
// Dijalankan sekali saat database dibuka, sebelum aplikasi melayani
// permintaan. Kalau sebuah migration sudah applied (tercatat di tabel
// schema_migrations), migration itu dilewati.
func applyMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
create table if not exists schema_migrations (
  name text primary key,
  applied_at integer not null
)`); err != nil {
		return fmt.Errorf("siapkan tabel migrasi: %w", err)
	}

	for _, item := range migrations {
		var applied int
		err := db.QueryRowContext(ctx,
			`select count(*) from schema_migrations where name = ?`, item.Name).Scan(&applied)
		if err != nil {
			return fmt.Errorf("cek migrasi %s: %w", item.Name, err)
		}
		if applied > 0 {
			continue
		}

		if _, err := db.ExecContext(ctx, item.SQL); err != nil {
			return fmt.Errorf("jalankan migrasi %s: %w", item.Name, err)
		}
		if _, err := db.ExecContext(ctx,
			`insert into schema_migrations(name, applied_at) values(?, strftime('%s','now'))`,
			item.Name); err != nil {
			return fmt.Errorf("catat migrasi %s: %w", item.Name, err)
		}
	}
	return nil
}

package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// Database lama dibuat dengan constraint status tanpa "refunded".
// Setelah migrasi, status refunded harus bisa dipakai.
func TestMigrasiIzinkanStatusRefunded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lama.db")

	// Simulasikan database lama: constraint tanpa refunded.
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("buka: %v", err)
	}
	oldSchema := `
create table topup_invoices (
  id text primary key,
  wallet_user_id text not null,
  paypan_invoice_id text,
  price integer not null,
  code integer not null,
  total integer not null,
  status text not null default 'pending' check (status in ('pending','paid','expired','cancelled')),
  source text not null default '',
  created_at integer not null,
  expires_at integer not null,
  paid_at integer
);
insert into topup_invoices values ('inv-1','w1','pp_1',25000,1,25001,'paid','webhook',100,200,150);
`
	if _, err := db.Exec(oldSchema); err != nil {
		t.Fatalf("skema lama: %v", err)
	}
	// Pastikan refunded memang ditolak sebelum migrasi.
	if _, err := db.Exec(`update topup_invoices set status='refunded' where id='inv-1'`); err == nil {
		t.Fatal("constraint lama seharusnya menolak refunded")
	}
	db.Close()

	// Buka lewat store: migrasi harus jalan.
	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	// Data lama harus tetap utuh.
	var status string
	var total int64
	err = store.db.QueryRow(`select status, total from topup_invoices where id='inv-1'`).Scan(&status, &total)
	if err != nil {
		t.Fatalf("baca data: %v", err)
	}
	if status != "paid" || total != 25001 {
		t.Fatalf("data berubah: status=%q total=%d", status, total)
	}

	// Sekarang refunded harus bisa dipakai.
	if _, err := store.db.Exec(`update topup_invoices set status='refunded' where id='inv-1'`); err != nil {
		t.Fatalf("refunded harus bisa dipakai setelah migrasi: %v", err)
	}
}

// Migrasi harus idempoten: dibuka dua kali tidak merusak apa pun.
func TestMigrasiIdempoten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idem.db")

	first, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open pertama: %v", err)
	}
	if _, err := first.db.Exec(`insert into topup_invoices values ('inv-2','w1','pp_2',10000,1,10001,'paid','x',1,2,3)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	first.Close()

	second, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open kedua: %v", err)
	}
	defer second.Close()

	var count int
	if err := second.db.QueryRow(`select count(*) from topup_invoices where id='inv-2'`).Scan(&count); err != nil {
		t.Fatalf("cek data: %v", err)
	}
	if count != 1 {
		t.Fatalf("baris setelah migrasi ulang = %d, want 1", count)
	}
}

// Database kosong harus tetap bisa dibuka (skema utama sudah cukup).
func TestBukaDatabaseKosong(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baru.db")
	store, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file db tidak dibuat: %v", err)
	}
}

package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// makeSource membuat database SQLite asli berisi tabel uji.
// Backup harus memakai database sungguhan karena prosesnya lewat VACUUM INTO.
func makeSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wallet.db")

	connection, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("buka db: %v", err)
	}
	defer connection.Close()

	if _, err := connection.Exec(`create table wallet_users (id text primary key, public_id text)`); err != nil {
		t.Fatalf("buat tabel: %v", err)
	}
	if _, err := connection.Exec(`insert into wallet_users values ('1','ABC12345')`); err != nil {
		t.Fatalf("isi tabel: %v", err)
	}
	return path
}

// countUsers membaca jumlah baris dari file backup.
func countUsers(t *testing.T, path string) int {
	t.Helper()
	connection, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		t.Fatalf("buka backup: %v", err)
	}
	defer connection.Close()

	var total int
	if err := connection.QueryRow(`select count(*) from wallet_users`).Scan(&total); err != nil {
		t.Fatalf("hitung baris: %v", err)
	}
	return total
}

func TestOnceCreatesUsableBackup(t *testing.T) {
	source := makeSource(t)
	dir := filepath.Join(t.TempDir(), "backup")

	runner := New(source, dir, 5, time.Hour)
	result, err := runner.Once()
	if err != nil {
		t.Fatalf("Once: %v", err)
	}
	if result.SizeBytes <= 0 {
		t.Fatalf("ukuran backup = %d", result.SizeBytes)
	}

	// Yang penting bukan ukuran, tapi isinya benar-benar bisa dibuka.
	if total := countUsers(t, filepath.Join(dir, result.Name)); total != 1 {
		t.Fatalf("backup berisi %d baris, want 1", total)
	}
}

// Regresi: dengan mode WAL, menyalin file mentah kehilangan data terbaru.
// Backup harus tetap memuat semua baris yang sudah di-commit.
func TestOnceIncludesCommittedRowsInWAL(t *testing.T) {
	source := makeSource(t)
	dir := filepath.Join(t.TempDir(), "backup")

	connection, err := sql.Open("sqlite", "file:"+filepath.ToSlash(source))
	if err != nil {
		t.Fatalf("buka db: %v", err)
	}
	if _, err := connection.Exec(`pragma journal_mode=wal`); err != nil {
		t.Fatalf("set wal: %v", err)
	}
	for i := 2; i <= 20; i++ {
		if _, err := connection.Exec(`insert into wallet_users values (?, 'X')`, string(rune('0'+i))); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	// Jangan tutup: biarkan WAL tetap ada, meniru server yang sedang jalan.
	defer connection.Close()

	runner := New(source, dir, 5, time.Hour)
	result, err := runner.Once()
	if err != nil {
		t.Fatalf("Once: %v", err)
	}

	total := countUsers(t, filepath.Join(dir, result.Name))
	if total != 20 {
		t.Fatalf("backup berisi %d baris, want 20 (data WAL harus ikut)", total)
	}
}

func TestOnceCreatesMissingDirectory(t *testing.T) {
	source := makeSource(t)
	dir := filepath.Join(t.TempDir(), "a", "b", "c")

	runner := New(source, dir, 5, time.Hour)
	if _, err := runner.Once(); err != nil {
		t.Fatalf("folder induk harus dibuat otomatis: %v", err)
	}
}

func TestPruneKeepsOnlyNewest(t *testing.T) {
	source := makeSource(t)
	dir := filepath.Join(t.TempDir(), "backup")

	runner := New(source, dir, 2, time.Hour)
	for i := 0; i < 4; i++ {
		if _, err := runner.Once(); err != nil {
			t.Fatalf("Once ke-%d: %v", i, err)
		}
		// Nama memakai detik; beri jeda agar unik dan urutan jelas.
		time.Sleep(1100 * time.Millisecond)
	}

	list, err := runner.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("jumlah backup = %d, want 2 (keep)", len(list))
	}
}

func TestListEmptyWhenDirectoryMissing(t *testing.T) {
	runner := New(makeSource(t), filepath.Join(t.TempDir(), "tidak-ada"), 3, time.Hour)
	list, err := runner.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("harus kosong, dapat %d", len(list))
	}
}

func TestListIgnoresForeignFiles(t *testing.T) {
	source := makeSource(t)
	dir := filepath.Join(t.TempDir(), "backup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "catatan.txt"), []byte("x"), 0o600)
	os.WriteFile(filepath.Join(dir, "lain.db"), []byte("x"), 0o600)

	runner := New(source, dir, 3, time.Hour)
	if _, err := runner.Once(); err != nil {
		t.Fatal(err)
	}

	list, _ := runner.List()
	if len(list) != 1 {
		t.Fatalf("hanya berkas pan-wallet-*.db yang dihitung, dapat %d", len(list))
	}
}

func TestStartRunsImmediately(t *testing.T) {
	source := makeSource(t)
	dir := filepath.Join(t.TempDir(), "backup")

	runner := New(source, dir, 3, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner.Start(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		list, _ := runner.List()
		if len(list) > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("backup saat start tidak terjadi")
}

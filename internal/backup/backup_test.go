package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeSource(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wallet.db")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("tulis sumber: %v", err)
	}
	return path
}

func TestOnceCreatesBackupFile(t *testing.T) {
	source := makeSource(t, "isi-database-asli")
	dir := filepath.Join(t.TempDir(), "backup")

	runner := New(source, dir, 5, time.Hour)
	backup, err := runner.Once()
	if err != nil {
		t.Fatalf("Once: %v", err)
	}

	if backup.SizeBytes != int64(len("isi-database-asli")) {
		t.Fatalf("ukuran = %d", backup.SizeBytes)
	}

	written, err := os.ReadFile(filepath.Join(dir, backup.Name))
	if err != nil {
		t.Fatalf("baca backup: %v", err)
	}
	if string(written) != "isi-database-asli" {
		t.Fatalf("isi backup = %q", string(written))
	}
}

func TestOnceCreatesMissingDirectory(t *testing.T) {
	source := makeSource(t, "x")
	dir := filepath.Join(t.TempDir(), "a", "b", "c")

	runner := New(source, dir, 5, time.Hour)
	if _, err := runner.Once(); err != nil {
		t.Fatalf("folder induk harus dibuat otomatis: %v", err)
	}
}

func TestPruneKeepsOnlyNewest(t *testing.T) {
	source := makeSource(t, "data")
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
	runner := New(makeSource(t, "x"), filepath.Join(t.TempDir(), "tidak-ada"), 3, time.Hour)
	list, err := runner.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("harus kosong, dapat %d", len(list))
	}
}

func TestListIgnoresForeignFiles(t *testing.T) {
	source := makeSource(t, "data")
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
	source := makeSource(t, "data")
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

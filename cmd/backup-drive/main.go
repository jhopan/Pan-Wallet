// Command backup-drive membuat snapshot database lalu mengunggahnya ke
// Google Drive memakai rclone. Dirancang untuk dijalankan dari cron/Task
// Scheduler, dan aman walau server PanWallet sedang melayani permintaan.
//
// Pemakaian:
//
//	backup-drive -remote jhopan
//	backup-drive -remote jhopan -keep-days 30
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhopan/PanWallet/internal/backup"
)

func main() {
	remote := flag.String("remote", envOr("RCLONE_REMOTE", "jhopan"), "nama remote rclone")
	dbPath := flag.String("db", envOr("PANWALLET_DB_PATH", "data/panwallet.db"), "lokasi database")
	backupDir := flag.String("dir", envOr("PANWALLET_BACKUP_DIR", "backup"), "folder backup lokal")
	keepDays := flag.Int("keep-days", 30, "hapus backup di Drive yang lebih tua dari N hari")
	dryRun := flag.Bool("dry-run", false, "hanya tampilkan rencana, tidak mengunggah")
	flag.Parse()

	stamp := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("panwallet-%s.db", stamp)
	target := filepath.Join(*backupDir, name)

	// 1. Snapshot konsisten (VACUUM INTO, menyertakan data WAL).
	fmt.Printf("[1/4] snapshot %s\n", *dbPath)
	size, err := backup.Snapshot(*dbPath, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal snapshot: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("      %s (%.1f KB)\n", target, float64(size)/1024)

	// 2. Verifikasi isi backup benar-benar bisa dibaca.
	fmt.Println("[2/4] verifikasi isi")
	count, err := countUsers(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backup rusak: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("      pelanggan: %d\n", count)

	if *dryRun {
		fmt.Println("[3/4] dry-run: tidak mengunggah")
		fmt.Println("[4/4] dry-run: tidak merapikan")
		return
	}

	// 3. Unggah ke Drive.
	destination := fmt.Sprintf("%s:PanWallet-Backup/", *remote)
	fmt.Printf("[3/4] unggah ke %s\n", destination)
	if err := runRclone("copy", target, destination, "--no-traverse"); err != nil {
		fmt.Fprintf(os.Stderr, "gagal unggah: %v\n", err)
		os.Exit(1)
	}

	// 4. Rapikan backup lama di Drive.
	fmt.Printf("[4/4] hapus backup > %d hari\n", *keepDays)
	if err := runRclone("delete", destination,
		"--min-age", fmt.Sprintf("%dd", *keepDays)); err != nil {
		// Bukan kegagalan fatal: backup baru sudah terunggah.
		fmt.Fprintf(os.Stderr, "peringatan saat merapikan: %v\n", err)
	}

	fmt.Println("selesai.")
}

// runRclone menjalankan perintah rclone dan menampilkan keluarannya.
func runRclone(args ...string) error {
	path, err := exec.LookPath("rclone")
	if err != nil {
		return fmt.Errorf("rclone tidak ditemukan di PATH")
	}
	command := exec.Command(path, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// countUsers memastikan backup dapat dibuka dan berisi tabel utama.
func countUsers(path string) (int, error) {
	return backup.CountRows(path, "wallet_users")
}

// envOr mengambil env, atau nilai cadangan kalau kosong.
func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

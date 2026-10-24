package backup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Snapshot membuat salinan database yang KONSISTEN.
//
// Penting: menyalin file .db dengan io.Copy TIDAK CUKUP saat mode WAL aktif.
// Data terbaru bisa masih berada di file -wal dan belum masuk ke .db,
// sehingga hasil salinan kehilangan transaksi terakhir (pernah terbukti:
// backup hanya berisi 1 dari 17 wallet).
//
// VACUUM INTO membuat salinan logis lewat SQLite sendiri, sehingga seluruh
// data yang sudah di-commit ikut terbawa dan hasilnya satu file utuh.
func Snapshot(sourcePath, targetPath string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return 0, fmt.Errorf("buat folder backup: %w", err)
	}
	// VACUUM INTO menolak menimpa file yang sudah ada.
	_ = os.Remove(targetPath)

	// Mode read-only: jangan sampai proses backup mengubah database aktif.
	dsn := "file:" + filepath.ToSlash(sourcePath) + "?mode=ro"
	connection, err := sql.Open("sqlite", dsn)
	if err != nil {
		return 0, fmt.Errorf("buka database: %w", err)
	}
	defer connection.Close()

	// Satu koneksi saja: VACUUM INTO tidak boleh berjalan paralel.
	connection.SetMaxOpenConns(1)

	// Nama target di-escape sebagai literal SQL.
	quoted := "'" + escapeSQL(targetPath) + "'"
	if _, err := connection.Exec("VACUUM INTO " + quoted); err != nil {
		return 0, fmt.Errorf("salin database: %w", err)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// escapeSQL menggandakan tanda kutip tunggal.
func escapeSQL(value string) string {
	escaped := ""
	for _, character := range value {
		if character == '\'' {
			escaped += "''"
			continue
		}
		escaped += string(character)
	}
	return escaped
}

// timestampedName membuat nama berkas backup dengan waktu lokal.
func timestampedName(now time.Time) string {
	return fmt.Sprintf("pan-wallet-%s.db", now.Format("20060102-150405"))
}

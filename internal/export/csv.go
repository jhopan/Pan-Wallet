package export

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Table adalah satu tabel yang bisa diekspor.
type Table struct {
	Name        string
	Description string
}

// Tables adalah daftar tabel yang aman diekspor untuk pembukuan.
// Tabel konfigurasi (app_settings) sengaja TIDAK ikut karena berisi
// token PayPan dan secret webhook — jangan sampai masuk spreadsheet.
var Tables = []Table{
	{Name: "wallet_users", Description: "Daftar pelanggan (public_id)"},
	{Name: "wallets", Description: "Saldo tiap pelanggan"},
	{Name: "wallet_ledger", Description: "Mutasi saldo (top-up, potong, koreksi)"},
	{Name: "topup_invoices", Description: "Invoice top-up QRIS"},
	{Name: "wallet_holds", Description: "Saldo yang ditahan transaksi"},
	{Name: "wallet_admin_audit", Description: "Jejak tindakan admin"},
}

// ExportCSV menulis satu tabel ke berkas CSV.
func ExportCSV(db *sql.DB, table, targetPath string) (int, error) {
	allowed := false
	for _, candidate := range Tables {
		if candidate.Name == table {
			allowed = true
			break
		}
	}
	if !allowed {
		return 0, fmt.Errorf("tabel %q tidak boleh diekspor", table)
	}

	rows, err := db.Query("select * from " + table)
	if err != nil {
		return 0, fmt.Errorf("baca %s: %w", table, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return 0, err
	}
	file, err := os.Create(targetPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(columns); err != nil {
		return 0, err
	}

	count := 0
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return count, err
		}

		record := make([]string, len(columns))
		for i, value := range values {
			record[i] = formatValue(value)
		}
		if err := writer.Write(record); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

// ExportAll menulis semua tabel ke satu folder, ditambah satu ringkasan.
func ExportAll(db *sql.DB, dir string) (map[string]int, error) {
	stamp := time.Now().Format("20060102-150405")
	folder := filepath.Join(dir, "export-"+stamp)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return nil, err
	}

	result := map[string]int{}
	for _, table := range Tables {
		path := filepath.Join(folder, table.Name+".csv")
		count, err := ExportCSV(db, table.Name, path)
		if err != nil {
			return result, err
		}
		result[table.Name] = count
	}
	return result, nil
}

// formatValue mengubah nilai database menjadi teks yang ramah spreadsheet.
func formatValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case string:
		return typed
	case time.Time:
		return typed.Format(time.RFC3339)
	default:
		return fmt.Sprint(typed)
	}
}

// Ringkasan menyusun teks ringkasan untuk laporan harian.
func Ringkasan(db *sql.DB) (string, error) {
	var pelanggan, saldo, invoiceLunas, nilaiLunas int64
	if err := db.QueryRow(`select count(*) from wallet_users`).Scan(&pelanggan); err != nil {
		return "", err
	}
	if err := db.QueryRow(`select coalesce(sum(balance),0) from wallets`).Scan(&saldo); err != nil {
		return "", err
	}
	if err := db.QueryRow(`select count(*) from topup_invoices where status = 'paid'`).Scan(&invoiceLunas); err != nil {
		return "", err
	}
	if err := db.QueryRow(`select coalesce(sum(total),0) from topup_invoices where status = 'paid'`).Scan(&nilaiLunas); err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("Ringkasan PanWallet\n")
	builder.WriteString("Waktu: " + time.Now().Format("02 Jan 2006 15:04") + "\n\n")
	builder.WriteString(fmt.Sprintf("Pelanggan      : %d\n", pelanggan))
	builder.WriteString(fmt.Sprintf("Total saldo    : Rp %s\n", formatRupiah(saldo)))
	builder.WriteString(fmt.Sprintf("Top-up lunas   : %d invoice\n", invoiceLunas))
	builder.WriteString(fmt.Sprintf("Nilai top-up   : Rp %s\n", formatRupiah(nilaiLunas)))
	return builder.String(), nil
}

// formatRupiah memberi pemisah ribuan titik.
func formatRupiah(value int64) string {
	text := strconv.FormatInt(value, 10)
	negative := strings.HasPrefix(text, "-")
	if negative {
		text = text[1:]
	}

	var parts []string
	for len(text) > 3 {
		parts = append([]string{text[len(text)-3:]}, parts...)
		text = text[:len(text)-3]
	}
	parts = append([]string{text}, parts...)

	result := strings.Join(parts, ".")
	if negative {
		return "-" + result
	}
	return result
}

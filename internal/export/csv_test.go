package export

import (
	"database/sql"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func seed(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "t.db")))
	if err != nil {
		t.Fatalf("buka: %v", err)
	}
	db.SetMaxOpenConns(1)
	statements := []string{
		`create table wallet_users (id text, public_id text, created_at integer)`,
		`create table wallets (id text, wallet_user_id text, balance integer)`,
		`create table wallet_ledger (id text, wallet_id text, amount integer, entry_type text)`,
		`create table topup_invoices (id text, total integer, status text)`,
		`create table wallet_holds (id text, amount integer, status text)`,
		`create table wallet_admin_audit (id text, action text)`,
		`create table app_settings (key text, value text)`,
		`insert into wallet_users values ('w1','ABC12345',1790000000)`,
		`insert into wallets values ('x1','w1',50000)`,
		`insert into topup_invoices values ('i1',25001,'paid')`,
		`insert into app_settings values ('paypan_token','pp_RAHASIA')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	return db
}

func TestExportCSVWritesHeaderAndRows(t *testing.T) {
	db := seed(t)
	defer db.Close()

	target := filepath.Join(t.TempDir(), "users.csv")
	count, err := ExportCSV(db, "wallet_users", target)
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if count != 1 {
		t.Fatalf("baris = %d, want 1", count)
	}

	file, err := os.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("total baris csv = %d, want 2 (header + data)", len(records))
	}
	if records[0][1] != "public_id" {
		t.Fatalf("header = %v", records[0])
	}
	if records[1][1] != "ABC12345" {
		t.Fatalf("data = %v", records[1])
	}
}

// Tabel konfigurasi berisi token — harus DITOLAK supaya rahasia tidak bocor
// ke spreadsheet yang mungkin dibagikan.
func TestExportRejectsSettingsTable(t *testing.T) {
	db := seed(t)
	defer db.Close()

	_, err := ExportCSV(db, "app_settings", filepath.Join(t.TempDir(), "x.csv"))
	if err == nil {
		t.Fatal("app_settings harus ditolak")
	}
}

func TestExportAllSkipsSettings(t *testing.T) {
	db := seed(t)
	defer db.Close()

	result, err := ExportAll(db, t.TempDir())
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if _, exists := result["app_settings"]; exists {
		t.Fatal("app_settings tidak boleh ikut diekspor")
	}
	if _, exists := result["wallet_users"]; !exists {
		t.Fatal("wallet_users harus ikut")
	}
}

func TestRingkasanHasKeyNumbers(t *testing.T) {
	db := seed(t)
	defer db.Close()

	text, err := Ringkasan(db)
	if err != nil {
		t.Fatalf("Ringkasan: %v", err)
	}
	for _, want := range []string{"Pelanggan", "Total saldo", "50.000", "Top-up lunas"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ringkasan tidak memuat %q:\n%s", want, text)
		}
	}
}

func TestFormatRupiah(t *testing.T) {
	cases := map[int64]string{
		0:       "0",
		999:     "999",
		1000:    "1.000",
		25000:   "25.000",
		1465017: "1.465.017",
		-5000:   "-5.000",
	}
	for input, want := range cases {
		if got := formatRupiah(input); got != want {
			t.Fatalf("formatRupiah(%d) = %q, want %q", input, got, want)
		}
	}
}

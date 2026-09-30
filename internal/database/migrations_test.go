package database

import "testing"

func TestMigrationsContainWalletCoreSchema(t *testing.T) {
	migrations := Migrations()
	if len(migrations) != 1 {
		t.Fatalf("migration count = %d, want 1", len(migrations))
	}
	if migrations[0].Name != "000001_wallet_core.sql" {
		t.Fatalf("migration name = %q", migrations[0].Name)
	}
	if migrations[0].SQL == "" {
		t.Fatal("migration SQL is empty")
	}
}

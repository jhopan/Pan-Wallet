package database

import "testing"

func TestMigrationsContainWalletSchemaAndAPISecurity(t *testing.T) {
	migrations := Migrations()
	if len(migrations) != 4 {
		t.Fatalf("migration count = %d, want 4", len(migrations))
	}
	for index, name := range []string{"000001_wallet_core.sql", "000002_api_security.sql", "000003_hold_transitions.sql", "000004_qris_topup.sql"} {
		if migrations[index].Name != name {
			t.Fatalf("migration %d name = %q, want %q", index, migrations[index].Name, name)
		}
		if migrations[index].SQL == "" {
			t.Fatalf("migration %q SQL is empty", name)
		}
	}
}

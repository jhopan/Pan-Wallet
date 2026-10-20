package config

import "testing"

func TestValidateRejectsSQLiteWithoutPath(t *testing.T) {
	settings := Settings{Driver: DriverSQLite, AdminUsername: "admin", AdminPassword: "rahasia123"}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, path sqlite kosong harus ditolak")
	}
}

func TestValidateAcceptsSQLite(t *testing.T) {
	settings := Settings{
		Driver:        DriverSQLite,
		DatabasePath:  "data/pan-wallet.db",
		AdminUsername: "admin",
		AdminPassword: "rahasia123",
	}
	if err := settings.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsPostgres(t *testing.T) {
	settings := Settings{
		Driver:        DriverPostgres,
		DatabaseURL:   "postgres://wallet:secret@example.test:5432/wallet?sslmode=require",
		AdminUsername: "admin",
		AdminPassword: "rahasia123",
	}
	if err := settings.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsPostgresWithoutURL(t *testing.T) {
	settings := Settings{Driver: DriverPostgres, AdminUsername: "admin", AdminPassword: "rahasia123"}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, URL postgres kosong harus ditolak")
	}
}

func TestValidateRejectsNonPostgresURL(t *testing.T) {
	settings := Settings{
		Driver:        DriverPostgres,
		DatabaseURL:   "mysql://host/db",
		AdminUsername: "admin",
		AdminPassword: "rahasia123",
	}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, URL non-Postgres harus ditolak")
	}
}

func TestValidateRejectsUnknownDriver(t *testing.T) {
	settings := Settings{Driver: "mysql", AdminUsername: "admin", AdminPassword: "rahasia123"}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, driver asing harus ditolak")
	}
}

func TestValidateRejectsShortAdminPassword(t *testing.T) {
	settings := Settings{
		Driver:        DriverSQLite,
		DatabasePath:  "data/pan-wallet.db",
		AdminUsername: "admin",
		AdminPassword: "pendek",
	}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, password < 8 karakter harus ditolak")
	}
}

func TestLoadDefaultsToSQLite(t *testing.T) {
	t.Setenv("PAN_WALLET_DB_DRIVER", "")
	t.Setenv("PAN_WALLET_DB_PATH", "")
	t.Setenv("PAN_WALLET_ADDR", "")

	settings := Load()

	if settings.Driver != DriverSQLite {
		t.Fatalf("Driver = %q, want sqlite", settings.Driver)
	}
	if settings.DatabasePath != "data/pan-wallet.db" {
		t.Fatalf("DatabasePath = %q, want data/pan-wallet.db", settings.DatabasePath)
	}
	if settings.Address != ":8080" {
		t.Fatalf("Address = %q, want :8080", settings.Address)
	}
}

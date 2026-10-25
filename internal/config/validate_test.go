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
		DatabasePath:  "data/panwallet.db",
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
		DatabasePath:  "data/panwallet.db",
		AdminUsername: "admin",
		AdminPassword: "pendek",
	}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, password < 8 karakter harus ditolak")
	}
}

func TestLoadDefaultsToSQLite(t *testing.T) {
	t.Setenv("PANWALLET_DB_DRIVER", "")
	t.Setenv("PANWALLET_DB_PATH", "")
	t.Setenv("PANWALLET_ADDR", "")
	t.Setenv("PANWALLET_HOST", "")
	t.Setenv("PANWALLET_PORT", "")

	settings := Load()

	if settings.Driver != DriverSQLite {
		t.Fatalf("Driver = %q, want sqlite", settings.Driver)
	}
	if settings.DatabasePath != "data/panwallet.db" {
		t.Fatalf("DatabasePath = %q, want data/panwallet.db", settings.DatabasePath)
	}
	if settings.Address != "127.0.0.1:8080" {
		t.Fatalf("Address = %q, want 127.0.0.1:8080", settings.Address)
	}
}

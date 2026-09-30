package config

import "testing"

func TestSettingsValidateRejectsMissingDatabaseURL(t *testing.T) {
	err := (Settings{Address: ":8080"}).Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing database URL error")
	}
}

func TestSettingsValidateAcceptsPostgresURL(t *testing.T) {
	settings := Settings{
		Address:     ":8080",
		DatabaseURL: "postgres://wallet:secret@example.test:5432/wallet?sslmode=require",
	}

	if err := settings.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSettingsValidateRejectsNonPostgresURL(t *testing.T) {
	settings := Settings{Address: ":8080", DatabaseURL: "sqlite:///wallet.db"}
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want PostgreSQL URL error")
	}
}

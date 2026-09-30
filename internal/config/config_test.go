package config

import "testing"

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("PAN_WALLET_ADDR", "127.0.0.1:9090")
	t.Setenv("PAN_WALLET_DATABASE_URL", "postgres://wallet:secret@example.test:5432/wallet")

	settings := Load()

	if settings.Address != "127.0.0.1:9090" {
		t.Fatalf("Address = %q", settings.Address)
	}
	if settings.DatabaseURL != "postgres://wallet:secret@example.test:5432/wallet" {
		t.Fatalf("DatabaseURL = %q", settings.DatabaseURL)
	}
}

func TestLoadUsesDefaultAddress(t *testing.T) {
	t.Setenv("PAN_WALLET_ADDR", "")
	t.Setenv("PAN_WALLET_DATABASE_URL", "")

	settings := Load()

	if settings.Address != ":8080" {
		t.Fatalf("Address = %q, want :8080", settings.Address)
	}
	if settings.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", settings.DatabaseURL)
	}
}

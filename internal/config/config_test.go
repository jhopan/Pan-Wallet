package config

import "testing"

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("PANWALLET_ADDR", "127.0.0.1:9090")
	t.Setenv("PANWALLET_DATABASE_URL", "postgres://wallet:secret@example.test:5432/wallet")

	settings := Load()

	if settings.Address != "127.0.0.1:9090" {
		t.Fatalf("Address = %q", settings.Address)
	}
	if settings.DatabaseURL != "postgres://wallet:secret@example.test:5432/wallet" {
		t.Fatalf("DatabaseURL = %q", settings.DatabaseURL)
	}
}

func TestLoadUsesDefaultAddress(t *testing.T) {
	t.Setenv("PANWALLET_ADDR", "")
	t.Setenv("PANWALLET_HOST", "")
	t.Setenv("PANWALLET_PORT", "")
	t.Setenv("PANWALLET_DATABASE_URL", "")

	settings := Load()

	// Default 127.0.0.1 (bukan :8080) supaya tidak terbuka ke jaringan.
	if settings.Address != "127.0.0.1:8080" {
		t.Fatalf("Address = %q, want 127.0.0.1:8080", settings.Address)
	}
	if settings.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", settings.DatabaseURL)
	}
}

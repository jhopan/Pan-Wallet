package config

import "testing"

func TestLoadReadsHostAndPort(t *testing.T) {
	t.Setenv("PAN_WALLET_HOST", "0.0.0.0")
	t.Setenv("PAN_WALLET_PORT", "9090")
	t.Setenv("PAN_WALLET_ADDR", "")

	settings := Load()

	if settings.Host != "0.0.0.0" {
		t.Fatalf("Host = %q, want 0.0.0.0", settings.Host)
	}
	if settings.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", settings.Port)
	}
	if settings.Address != "0.0.0.0:9090" {
		t.Fatalf("Address = %q, want 0.0.0.0:9090", settings.Address)
	}
}

func TestLoadDefaultsToLocalhost(t *testing.T) {
	t.Setenv("PAN_WALLET_HOST", "")
	t.Setenv("PAN_WALLET_PORT", "")
	t.Setenv("PAN_WALLET_ADDR", "")

	settings := Load()

	// Default harus 127.0.0.1, bukan :8080 — supaya tidak terbuka ke jaringan.
	if settings.Host != "127.0.0.1" {
		t.Fatalf("Host default = %q, want 127.0.0.1", settings.Host)
	}
	if settings.Address != "127.0.0.1:8080" {
		t.Fatalf("Address default = %q", settings.Address)
	}
}

func TestLoadAddrOverridesHostPort(t *testing.T) {
	t.Setenv("PAN_WALLET_HOST", "127.0.0.1")
	t.Setenv("PAN_WALLET_PORT", "8080")
	t.Setenv("PAN_WALLET_ADDR", ":3000")

	settings := Load()

	if settings.Address != ":3000" {
		t.Fatalf("Address = %q, want :3000 (ADDR harus menang)", settings.Address)
	}
}

func TestLoadParsesTrustedProxies(t *testing.T) {
	t.Setenv("PAN_WALLET_TRUSTED_PROXIES", "127.0.0.1, ::1 ,10.0.0.0/8")

	settings := Load()

	if len(settings.TrustedProxies) != 3 {
		t.Fatalf("TrustedProxies = %v, want 3 entri", settings.TrustedProxies)
	}
	if settings.TrustedProxies[1] != "::1" {
		t.Fatalf("entri kedua = %q, want ::1 (spasi harus dibuang)", settings.TrustedProxies[1])
	}
}

func TestLoadReadsPublicURL(t *testing.T) {
	t.Setenv("PAN_WALLET_PUBLIC_URL", "https://wallet.contoh.com/")

	settings := Load()

	// trailing slash harus dibuang supaya penyusunan URL tidak dobel
	if settings.PublicBaseURL != "https://wallet.contoh.com" {
		t.Fatalf("PublicBaseURL = %q", settings.PublicBaseURL)
	}
}

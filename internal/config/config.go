package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// Driver adalah mesin database yang dipakai Pan-Wallet.
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
)

type Settings struct {
	// Host dan Port dipisah supaya jelas: Host=127.0.0.1 berarti hanya
	// bisa diakses dari mesin sendiri (cocok untuk cloudflared tunnel).
	Host string
	Port string
	// Address adalah Host:Port hasil gabungan, dipakai ListenAndServe.
	Address       string
	Driver        Driver
	DatabaseURL   string // postgres: connection URL
	DatabasePath  string // sqlite: path file .db
	BackupDir     string // folder cadangan SQLite
	AdminUsername string
	AdminPassword string
	// TrustedProxies adalah daftar IP/CIDR reverse proxy yang boleh
	// mengirim header X-Forwarded-For. Kosong = tidak ada yang dipercaya.
	TrustedProxies []string
	// PublicBaseURL dipakai untuk menyusun link (mis. URL webhook PayPan
	// dan pay_url) saat berjalan di balik tunnel/domain.
	PublicBaseURL string
}

func (settings Settings) Validate() error {
	switch settings.Driver {
	case DriverSQLite:
		if strings.TrimSpace(settings.DatabasePath) == "" {
			return fmt.Errorf("PAN_WALLET_DB_PATH wajib diisi untuk driver sqlite")
		}
	case DriverPostgres:
		if settings.DatabaseURL == "" {
			return fmt.Errorf("PAN_WALLET_DATABASE_URL wajib diisi untuk driver postgres")
		}
		databaseURL, err := url.Parse(settings.DatabaseURL)
		if err != nil || databaseURL.Scheme == "" {
			return fmt.Errorf("PAN_WALLET_DATABASE_URL tidak valid")
		}
		if databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql" {
			return fmt.Errorf("PAN_WALLET_DATABASE_URL harus URL PostgreSQL")
		}
	default:
		return fmt.Errorf("PAN_WALLET_DB_DRIVER harus 'sqlite' atau 'postgres'")
	}

	if settings.AdminUsername == "" {
		return fmt.Errorf("PAN_WALLET_ADMIN_USER wajib diisi")
	}
	if len(settings.AdminPassword) < 8 {
		return fmt.Errorf("PAN_WALLET_ADMIN_PASS minimal 8 karakter")
	}
	return nil
}

func Load() Settings {
	// Host default 127.0.0.1: aman secara bawaan. Dengan cloudflared,
	// tunnel yang menghubungi localhost, jadi port tidak perlu dibuka.
	host := strings.TrimSpace(os.Getenv("PAN_WALLET_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}

	port := strings.TrimSpace(os.Getenv("PAN_WALLET_PORT"))
	if port == "" {
		port = "8080"
	}

	// PAN_WALLET_ADDR menang kalau diisi eksplisit (kompatibilitas lama).
	address := strings.TrimSpace(os.Getenv("PAN_WALLET_ADDR"))
	if address == "" {
		address = net.JoinHostPort(host, port)
	}

	driver := Driver(strings.ToLower(strings.TrimSpace(os.Getenv("PAN_WALLET_DB_DRIVER"))))
	if driver == "" {
		// Default SQLite: tidak butuh server database.
		driver = DriverSQLite
	}

	databasePath := os.Getenv("PAN_WALLET_DB_PATH")
	if databasePath == "" && driver == DriverSQLite {
		databasePath = "data/pan-wallet.db"
	}

	backupDir := os.Getenv("PAN_WALLET_BACKUP_DIR")
	if backupDir == "" {
		backupDir = "backup"
	}

	return Settings{
		Host:           host,
		Port:           port,
		Address:        address,
		BackupDir:      backupDir,
		Driver:         driver,
		DatabaseURL:    os.Getenv("PAN_WALLET_DATABASE_URL"),
		DatabasePath:   databasePath,
		AdminUsername:  os.Getenv("PAN_WALLET_ADMIN_USER"),
		AdminPassword:  os.Getenv("PAN_WALLET_ADMIN_PASS"),
		TrustedProxies: splitList(os.Getenv("PAN_WALLET_TRUSTED_PROXIES")),
		PublicBaseURL:  strings.TrimRight(strings.TrimSpace(os.Getenv("PAN_WALLET_PUBLIC_URL")), "/"),
	}
}

// splitList memecah daftar dipisah koma, membuang entri kosong.
func splitList(value string) []string {
	var items []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

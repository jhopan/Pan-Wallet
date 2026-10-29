package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Driver adalah mesin database yang dipakai PanWallet.
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
	// BackupKeep adalah jumlah berkas backup lokal yang disimpan.
	// Default 3: cukup untuk pemulihan, tidak memakan disk.
	BackupKeep int
}

func (settings Settings) Validate() error {
	switch settings.Driver {
	case DriverSQLite:
		if strings.TrimSpace(settings.DatabasePath) == "" {
			return fmt.Errorf("PANWALLET_DB_PATH wajib diisi untuk driver sqlite")
		}
	case DriverPostgres:
		if settings.DatabaseURL == "" {
			return fmt.Errorf("PANWALLET_DATABASE_URL wajib diisi untuk driver postgres")
		}
		databaseURL, err := url.Parse(settings.DatabaseURL)
		if err != nil || databaseURL.Scheme == "" {
			return fmt.Errorf("PANWALLET_DATABASE_URL tidak valid")
		}
		if databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql" {
			return fmt.Errorf("PANWALLET_DATABASE_URL harus URL PostgreSQL")
		}
	default:
		return fmt.Errorf("PANWALLET_DB_DRIVER harus 'sqlite' atau 'postgres'")
	}

	if settings.AdminUsername == "" {
		return fmt.Errorf("PANWALLET_ADMIN_USER wajib diisi")
	}
	if len(settings.AdminPassword) < 8 {
		return fmt.Errorf("PANWALLET_ADMIN_PASS minimal 8 karakter")
	}
	return nil
}

func Load() Settings {
	// Host default 127.0.0.1: aman secara bawaan. Dengan cloudflared,
	// tunnel yang menghubungi localhost, jadi port tidak perlu dibuka.
	host := strings.TrimSpace(os.Getenv("PANWALLET_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}

	port := strings.TrimSpace(os.Getenv("PANWALLET_PORT"))
	if port == "" {
		port = "8080"
	}

	// PANWALLET_ADDR menang kalau diisi eksplisit (kompatibilitas lama).
	address := strings.TrimSpace(os.Getenv("PANWALLET_ADDR"))
	if address == "" {
		address = net.JoinHostPort(host, port)
	}

	driver := Driver(strings.ToLower(strings.TrimSpace(os.Getenv("PANWALLET_DB_DRIVER"))))
	if driver == "" {
		// Default SQLite: tidak butuh server database.
		driver = DriverSQLite
	}

	databasePath := os.Getenv("PANWALLET_DB_PATH")
	if databasePath == "" && driver == DriverSQLite {
		databasePath = "data/panwallet.db"
	}

	backupDir := os.Getenv("PANWALLET_BACKUP_DIR")
	if backupDir == "" {
		backupDir = "backup"
	}

	return Settings{
		Host:           host,
		Port:           port,
		Address:        address,
		BackupDir:      backupDir,
		Driver:         driver,
		DatabaseURL:    os.Getenv("PANWALLET_DATABASE_URL"),
		DatabasePath:   databasePath,
		AdminUsername:  os.Getenv("PANWALLET_ADMIN_USER"),
		AdminPassword:  os.Getenv("PANWALLET_ADMIN_PASS"),
		TrustedProxies: splitList(os.Getenv("PANWALLET_TRUSTED_PROXIES")),
		PublicBaseURL:  strings.TrimRight(strings.TrimSpace(os.Getenv("PANWALLET_PUBLIC_URL")), "/"),
		BackupKeep:     intEnv("PANWALLET_BACKUP_KEEP", 3),
	}
}

// intEnv membaca bilangan bulat dari env, dengan nilai cadangan.
func intEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
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

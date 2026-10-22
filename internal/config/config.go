package config

import (
	"fmt"
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
	Address       string
	Driver        Driver
	DatabaseURL   string // postgres: connection URL
	DatabasePath  string // sqlite: path file .db
	BackupDir     string // folder cadangan SQLite
	AdminUsername string
	AdminPassword string
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
	address := os.Getenv("PAN_WALLET_ADDR")
	if address == "" {
		address = ":8080"
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
		Address:       address,
		BackupDir:     backupDir,
		Driver:        driver,
		DatabaseURL:   os.Getenv("PAN_WALLET_DATABASE_URL"),
		DatabasePath:  databasePath,
		AdminUsername: os.Getenv("PAN_WALLET_ADMIN_USER"),
		AdminPassword: os.Getenv("PAN_WALLET_ADMIN_PASS"),
	}
}

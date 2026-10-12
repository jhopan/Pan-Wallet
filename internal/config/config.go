package config

import (
	"fmt"
	"net/url"
	"os"
)

type Settings struct {
	Address       string
	DatabaseURL   string
	AdminUsername string
	AdminPassword string
}

func (settings Settings) Validate() error {
	if settings.DatabaseURL == "" {
		return fmt.Errorf("PAN_WALLET_DATABASE_URL wajib diisi")
	}

	databaseURL, err := url.Parse(settings.DatabaseURL)
	if err != nil || databaseURL.Scheme == "" {
		return fmt.Errorf("PAN_WALLET_DATABASE_URL tidak valid")
	}
	if databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql" {
		return fmt.Errorf("PAN_WALLET_DATABASE_URL harus URL PostgreSQL")
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

	return Settings{
		Address:       address,
		DatabaseURL:   os.Getenv("PAN_WALLET_DATABASE_URL"),
		AdminUsername: os.Getenv("PAN_WALLET_ADMIN_USER"),
		AdminPassword: os.Getenv("PAN_WALLET_ADMIN_PASS"),
	}
}

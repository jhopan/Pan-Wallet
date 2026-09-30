package config

import (
	"fmt"
	"net/url"
	"os"
)

type Settings struct {
	Address     string
	DatabaseURL string
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
	return nil
}

func Load() Settings {
	address := os.Getenv("PAN_WALLET_ADDR")
	if address == "" {
		address = ":8080"
	}

	return Settings{
		Address:     address,
		DatabaseURL: os.Getenv("PAN_WALLET_DATABASE_URL"),
	}
}

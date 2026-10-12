package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/auth"
	"github.com/jhopan/Pan-Wallet/internal/config"
	"github.com/jhopan/Pan-Wallet/internal/database"
	"github.com/jhopan/Pan-Wallet/internal/httpapi"
)

func main() {
	settings := config.Load()
	if err := settings.Validate(); err != nil {
		log.Fatal(err)
	}

	store, err := database.Open(context.Background(), settings.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	adminAuth := httpapi.NewAdminAuth(settings.AdminUsername, settings.AdminPassword)
	go func() {
		for range time.Tick(30 * time.Minute) {
			adminAuth.Sweep()
		}
	}()

	server := httpapi.New(httpapi.Deps{
		Auth:     auth.NewProductAuthenticator(store),
		Holds:    store,
		Overview: store,
		Topups:   store,
		Settings: store,
		Admin:    adminAuth,
	})

	log.Printf("Pan-Wallet listening on %s", settings.Address)
	if err := http.ListenAndServe(settings.Address, server); err != nil {
		log.Fatal(err)
	}
}

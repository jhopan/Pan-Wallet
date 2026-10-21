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
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

func main() {
	settings := config.Load()
	if err := settings.Validate(); err != nil {
		log.Fatal(err)
	}

	adminAuth := httpapi.NewAdminAuth(settings.AdminUsername, settings.AdminPassword)
	deps := httpapi.Deps{Admin: adminAuth}
	var closeStore func()

	switch settings.Driver {
	case config.DriverSQLite:
		store, err := sqlite.Open(context.Background(), settings.DatabasePath)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("database: sqlite %s", settings.DatabasePath)
		deps.Auth = auth.NewProductAuthenticator(store)
		deps.Holds = store
		deps.Overview = store
		deps.Topups = store
		deps.Settings = store
		deps.Ledger = store
		deps.Keys = store
		deps.Wallets = store
		deps.PublicIDs = store
		deps.Identities = store
		deps.Reconcile = store
		closeStore = store.Close

	default:
		store, err := database.Open(context.Background(), settings.DatabaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("database: postgres")
		deps.Auth = auth.NewProductAuthenticator(store)
		deps.Holds = store
		deps.Overview = store
		deps.Topups = store
		deps.Settings = store
		deps.Ledger = store
		deps.Keys = store
		deps.Wallets = store
		deps.PublicIDs = store
		deps.Identities = store
		deps.Reconcile = store
		closeStore = store.Close
	}
	defer closeStore()

	go func() {
		for range time.Tick(30 * time.Minute) {
			adminAuth.Sweep()
		}
	}()

	server := httpapi.New(deps)
	server.StartReconciler(context.Background())

	log.Printf("Pan-Wallet listening on %s", settings.Address)
	if err := http.ListenAndServe(settings.Address, server); err != nil {
		log.Fatal(err)
	}
}

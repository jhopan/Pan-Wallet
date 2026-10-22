package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/auth"
	"github.com/jhopan/Pan-Wallet/internal/backup"
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
		deps.Pinger = store
		deps.Audit = store
		deps.Revenue = store
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
		deps.Pinger = store
		deps.Audit = store
		deps.Revenue = store
		closeStore = store.Close
	}
	defer closeStore()

	// Backup otomatis: hanya untuk SQLite (satu berkas).
	if settings.Driver == config.DriverSQLite {
		runner := backup.New(settings.DatabasePath, settings.BackupDir, 24, time.Hour)
		runner.Start(context.Background())
		deps.Backup = runner
		log.Printf("backup: %s (setiap jam, simpan 24 terakhir)", settings.BackupDir)
	}

	go func() {
		for range time.Tick(30 * time.Minute) {
			adminAuth.Sweep()
		}
	}()

	server := httpapi.New(deps)
	server.StartReconciler(context.Background())

	httpServer := &http.Server{
		Addr:              settings.Address,
		Handler:           server,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown: selesaikan permintaan berjalan sebelum berhenti,
	// supaya transaksi tidak terputus di tengah jalan saat restart.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Pan-Wallet listening on %s", settings.Address)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-stop
	log.Println("sinyal berhenti diterima, menutup dengan rapi...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("berhenti.")
}

package main

import (
	"context"
	"log"
	"net/http"

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

	productAuthenticator := auth.NewProductAuthenticator(store)

	log.Printf("Pan-Wallet listening on %s", settings.Address)
	if err := http.ListenAndServe(settings.Address, httpapi.New(productAuthenticator, store)); err != nil {
		log.Fatal(err)
	}
}

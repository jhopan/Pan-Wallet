package main

import (
	"log"
	"net/http"

	"github.com/jhopan/Pan-Wallet/internal/config"
	"github.com/jhopan/Pan-Wallet/internal/httpapi"
)

func main() {
	settings := config.Load()
	if err := settings.Validate(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Pan-Wallet listening on %s", settings.Address)
	if err := http.ListenAndServe(settings.Address, httpapi.New()); err != nil {
		log.Fatal(err)
	}
}

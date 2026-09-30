package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jhopan/Pan-Wallet/internal/httpapi"
)

func main() {
	address := os.Getenv("PAN_WALLET_ADDR")
	if address == "" {
		address = ":8080"
	}

	log.Printf("Pan-Wallet listening on %s", address)
	if err := http.ListenAndServe(address, httpapi.New()); err != nil {
		log.Fatal(err)
	}
}

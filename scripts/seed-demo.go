//go:build ignore

// Seed demo: membuat satu wallet user bersaldo awal + product API key.
// Jalankan: go run scripts/seed-demo.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhopan/PanWallet/internal/sqlite"
)

func main() {
	store, err := sqlite.Open(context.Background(), "data/panwallet.db")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	userID, err := store.CreateWalletUser(context.Background(), 100_000)
	if err != nil {
		log.Fatal(err)
	}
	token, err := store.CreateProductKey(context.Background(), "agenpulsa")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("wallet_user_id :", userID)
	fmt.Println("saldo awal     : 100000")
	fmt.Println("product        : agenpulsa")
	fmt.Println("api_key        :", token)
}

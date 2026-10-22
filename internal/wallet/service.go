package wallet

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInsufficientBalance = errors.New("saldo tidak cukup")
	ErrHoldNotFound        = errors.New("hold tidak ditemukan")
	ErrHoldProcessed       = errors.New("hold sudah diproses")
	ErrHoldExpired         = errors.New("hold sudah kadaluarsa")
	ErrIdempotencyConflict = errors.New("idempotency key konflik")
	ErrWalletBlocked       = errors.New("wallet diblokir")
)

type CreateHoldRequest struct {
	WalletUserID   string
	Product        string
	ReferenceID    string
	Amount         int64
	IdempotencyKey string
}

func (request CreateHoldRequest) Validate() error {
	if strings.TrimSpace(request.WalletUserID) == "" ||
		strings.TrimSpace(request.Product) == "" ||
		strings.TrimSpace(request.ReferenceID) == "" ||
		strings.TrimSpace(request.IdempotencyKey) == "" ||
		request.Amount <= 0 {
		return fmt.Errorf("permintaan hold tidak valid")
	}
	return nil
}

// Hold adalah saldo yang dikunci sementara untuk satu transaksi.
// Tag JSON memakai snake_case supaya konsisten dengan endpoint lain —
// bot lain cukup membaca .id, bukan .ID.
type Hold struct {
	ID             string `json:"id"`
	WalletUserID   string `json:"wallet_user_id"`
	Product        string `json:"product"`
	ReferenceID    string `json:"reference_id"`
	Amount         int64  `json:"amount"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
}

type TransitionRequest struct {
	HoldID         string
	Product        string
	IdempotencyKey string
}

func (request TransitionRequest) Validate() error {
	if strings.TrimSpace(request.HoldID) == "" ||
		strings.TrimSpace(request.Product) == "" ||
		strings.TrimSpace(request.IdempotencyKey) == "" {
		return fmt.Errorf("permintaan transisi hold tidak valid")
	}
	return nil
}

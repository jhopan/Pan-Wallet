package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

func (store *Store) CreateHold(ctx context.Context, request wallet.CreateHoldRequest, expiresAt time.Time) (wallet.Hold, error) {
	if err := request.Validate(); err != nil {
		return wallet.Hold{}, err
	}

	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("mulai hold: %w", err)
	}
	defer transaction.Rollback(ctx)

	var existing wallet.Hold
	err = transaction.QueryRow(ctx, `
		select h.id::text, w.wallet_user_id::text, h.product, h.reference_id, h.amount, h.status, h.idempotency_key
		from wallet_holds h join wallets w on w.id = h.wallet_id
		where h.idempotency_key = $1`, request.IdempotencyKey,
	).Scan(&existing.ID, &existing.WalletUserID, &existing.Product, &existing.ReferenceID, &existing.Amount, &existing.Status, &existing.IdempotencyKey)
	if err == nil {
		if existing.WalletUserID != request.WalletUserID ||
			existing.Product != request.Product ||
			existing.ReferenceID != request.ReferenceID ||
			existing.Amount != request.Amount {
			return wallet.Hold{}, wallet.ErrIdempotencyConflict
		}
		return existing, transaction.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return wallet.Hold{}, fmt.Errorf("cek idempotensi hold: %w", err)
	}

	var walletID string
	var balance, held int64
	err = transaction.QueryRow(ctx, `
		select id::text, balance from wallets
		where wallet_user_id = $1::uuid for update`, request.WalletUserID,
	).Scan(&walletID, &balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return wallet.Hold{}, wallet.ErrHoldNotFound
	}
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("kunci wallet: %w", err)
	}
	if err := transaction.QueryRow(ctx,
		`select coalesce(sum(amount), 0) from wallet_holds where wallet_id = $1::uuid and status = 'active'`, walletID,
	).Scan(&held); err != nil {
		return wallet.Hold{}, fmt.Errorf("hitung hold aktif: %w", err)
	}
	if request.Amount > balance-held {
		return wallet.Hold{}, wallet.ErrInsufficientBalance
	}

	var hold wallet.Hold
	err = transaction.QueryRow(ctx, `
		insert into wallet_holds (wallet_id, amount, product, reference_id, expires_at, idempotency_key)
		values ($1::uuid, $2, $3, $4, $5, $6)
		returning id::text, $7::text, product, reference_id, amount, status, idempotency_key`,
		walletID, request.Amount, request.Product, request.ReferenceID, expiresAt, request.IdempotencyKey, request.WalletUserID,
	).Scan(&hold.ID, &hold.WalletUserID, &hold.Product, &hold.ReferenceID, &hold.Amount, &hold.Status, &hold.IdempotencyKey)
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("buat hold: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return wallet.Hold{}, fmt.Errorf("simpan hold: %w", err)
	}
	return hold, nil
}

func (store *Store) CaptureHold(ctx context.Context, request wallet.TransitionRequest) (wallet.Hold, error) {
	return store.transitionHold(ctx, request, "captured")
}

func (store *Store) ReleaseHold(ctx context.Context, request wallet.TransitionRequest) (wallet.Hold, error) {
	return store.transitionHold(ctx, request, "released")
}

func (store *Store) transitionHold(ctx context.Context, request wallet.TransitionRequest, targetStatus string) (wallet.Hold, error) {
	if err := request.Validate(); err != nil {
		return wallet.Hold{}, err
	}

	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("mulai transisi hold: %w", err)
	}
	defer transaction.Rollback(ctx)

	var hold wallet.Hold
	var walletID string
	var expiresAt time.Time
	err = transaction.QueryRow(ctx, `
		select h.id::text, w.wallet_user_id::text, h.product, h.reference_id, h.amount, h.status, h.idempotency_key, h.wallet_id::text, h.expires_at
		from wallet_holds h join wallets w on w.id = h.wallet_id
		where h.id = $1::uuid and h.product = $2 for update`, request.HoldID, request.Product,
	).Scan(&hold.ID, &hold.WalletUserID, &hold.Product, &hold.ReferenceID, &hold.Amount, &hold.Status, &hold.IdempotencyKey, &walletID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return wallet.Hold{}, wallet.ErrHoldNotFound
	}
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("kunci hold: %w", err)
	}
	action := "capture"
	if targetStatus == "released" {
		action = "release"
	}

	var existingKey string
	err = transaction.QueryRow(ctx,
		`select idempotency_key from wallet_hold_transitions where hold_id = $1::uuid and action = $2`, hold.ID, action,
	).Scan(&existingKey)
	if err == nil {
		if existingKey != request.IdempotencyKey {
			return wallet.Hold{}, wallet.ErrIdempotencyConflict
		}
		return hold, transaction.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return wallet.Hold{}, fmt.Errorf("cek idempotensi transisi: %w", err)
	}
	if hold.Status != "active" {
		return wallet.Hold{}, wallet.ErrHoldProcessed
	}
	if !expiresAt.After(time.Now()) {
		if _, err := transaction.Exec(ctx, `update wallet_holds set status = 'expired' where id = $1::uuid`, hold.ID); err != nil {
			return wallet.Hold{}, fmt.Errorf("tandai hold kadaluarsa: %w", err)
		}
		return wallet.Hold{}, wallet.ErrHoldExpired
	}
	if _, err := transaction.Exec(ctx,
		`insert into wallet_hold_transitions (hold_id, action, idempotency_key) values ($1::uuid, $2, $3)`,
		hold.ID, action, request.IdempotencyKey,
	); err != nil {
		return wallet.Hold{}, fmt.Errorf("catat transisi hold: %w", err)
	}

	if targetStatus == "captured" {
		var balanceAfter int64
		err = transaction.QueryRow(ctx, `
			update wallets set balance = balance - $1, updated_at = now()
			where id = $2::uuid and balance >= $1
			returning balance`, hold.Amount, walletID).Scan(&balanceAfter)
		if errors.Is(err, pgx.ErrNoRows) {
			return wallet.Hold{}, wallet.ErrInsufficientBalance
		}
		if err != nil {
			return wallet.Hold{}, fmt.Errorf("kurangi saldo: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			insert into wallet_ledger (wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key)
			values ($1::uuid, $2, $3, 'hold_capture', $4, $5, $6)`,
			walletID, -hold.Amount, balanceAfter, hold.Product, hold.ReferenceID, request.IdempotencyKey,
		); err != nil {
			return wallet.Hold{}, fmt.Errorf("catat capture: %w", err)
		}
	}

	if _, err := transaction.Exec(ctx, `update wallet_holds set status = $1 where id = $2::uuid`, targetStatus, hold.ID); err != nil {
		return wallet.Hold{}, fmt.Errorf("ubah status hold: %w", err)
	}
	hold.Status = targetStatus
	if err := transaction.Commit(ctx); err != nil {
		return wallet.Hold{}, fmt.Errorf("simpan transisi hold: %w", err)
	}
	return hold, nil
}

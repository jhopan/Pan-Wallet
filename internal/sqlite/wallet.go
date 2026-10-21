package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/report"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

// newID membuat id hex acak (pengganti uuid PostgreSQL).
func newID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}

// CreateWalletUser membuat wallet user beserta wallet dengan saldo awal.
// Dipakai admin/panel dan seeding; bukan jalur kredit normal.
func (store *Store) CreateWalletUser(ctx context.Context, balance int64) (string, error) {
	if balance < 0 {
		return "", fmt.Errorf("saldo awal tidak boleh negatif")
	}
	userID := newID()
	walletID := newID()
	now := time.Now().Unix()

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("mulai transaksi: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`insert into wallet_users (id, status, created_at) values (?, 'active', ?)`, userID, now); err != nil {
		return "", fmt.Errorf("buat wallet user: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`insert into wallets (id, wallet_user_id, balance, currency, updated_at) values (?, ?, ?, 'IDR', ?)`,
		walletID, userID, balance, now); err != nil {
		return "", fmt.Errorf("buat wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("simpan wallet user: %w", err)
	}
	return userID, nil
}

// CreateHold mengunci saldo tersedia untuk satu order produk.
func (store *Store) CreateHold(ctx context.Context, request wallet.CreateHoldRequest, expiresAt time.Time) (wallet.Hold, error) {
	if err := request.Validate(); err != nil {
		return wallet.Hold{}, err
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("mulai hold: %w", err)
	}
	defer tx.Rollback()

	// Idempotensi: key sama mengembalikan hold lama, payload beda = konflik.
	var existing wallet.Hold
	err = tx.QueryRowContext(ctx, `
		select h.id, w.wallet_user_id, h.product, h.reference_id, h.amount, h.status, h.idempotency_key
		from wallet_holds h join wallets w on w.id = h.wallet_id
		where h.idempotency_key = ?`, request.IdempotencyKey,
	).Scan(&existing.ID, &existing.WalletUserID, &existing.Product, &existing.ReferenceID,
		&existing.Amount, &existing.Status, &existing.IdempotencyKey)
	if err == nil {
		if existing.WalletUserID != request.WalletUserID ||
			existing.Product != request.Product ||
			existing.ReferenceID != request.ReferenceID ||
			existing.Amount != request.Amount {
			return wallet.Hold{}, wallet.ErrIdempotencyConflict
		}
		return existing, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return wallet.Hold{}, fmt.Errorf("cek idempotensi hold: %w", err)
	}

	var walletID string
	var balance int64
	var status string
	err = tx.QueryRowContext(ctx, `
		select w.id, w.balance, u.status from wallets w
		join wallet_users u on u.id = w.wallet_user_id
		where w.wallet_user_id = ?`, request.WalletUserID).Scan(&walletID, &balance, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return wallet.Hold{}, wallet.ErrHoldNotFound
	}
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("baca wallet: %w", err)
	}
	if status != "active" {
		return wallet.Hold{}, wallet.ErrWalletBlocked
	}

	var held int64
	if err := tx.QueryRowContext(ctx,
		`select coalesce(sum(amount), 0) from wallet_holds where wallet_id = ? and status = 'active'`, walletID,
	).Scan(&held); err != nil {
		return wallet.Hold{}, fmt.Errorf("hitung hold aktif: %w", err)
	}
	if request.Amount > balance-held {
		return wallet.Hold{}, wallet.ErrInsufficientBalance
	}

	hold := wallet.Hold{
		ID:             newID(),
		WalletUserID:   request.WalletUserID,
		Product:        request.Product,
		ReferenceID:    request.ReferenceID,
		Amount:         request.Amount,
		Status:         "active",
		IdempotencyKey: request.IdempotencyKey,
	}
	if _, err := tx.ExecContext(ctx, `
		insert into wallet_holds (id, wallet_id, amount, product, reference_id, status, expires_at, idempotency_key, created_at)
		values (?, ?, ?, ?, ?, 'active', ?, ?, ?)`,
		hold.ID, walletID, hold.Amount, hold.Product, hold.ReferenceID,
		expiresAt.Unix(), hold.IdempotencyKey, time.Now().Unix()); err != nil {
		return wallet.Hold{}, fmt.Errorf("buat hold: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return wallet.Hold{}, fmt.Errorf("simpan hold: %w", err)
	}
	return hold, nil
}

// CaptureHold mendebit saldo sekali dan mencatat ledger.
func (store *Store) CaptureHold(ctx context.Context, request wallet.TransitionRequest) (wallet.Hold, error) {
	return store.transitionHold(ctx, request, "captured")
}

// ReleaseHold melepas hold tanpa mengubah saldo.
func (store *Store) ReleaseHold(ctx context.Context, request wallet.TransitionRequest) (wallet.Hold, error) {
	return store.transitionHold(ctx, request, "released")
}

func (store *Store) transitionHold(ctx context.Context, request wallet.TransitionRequest, target string) (wallet.Hold, error) {
	if err := request.Validate(); err != nil {
		return wallet.Hold{}, err
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("mulai transisi: %w", err)
	}
	defer tx.Rollback()

	var hold wallet.Hold
	var walletID string
	var expiresAt int64
	err = tx.QueryRowContext(ctx, `
		select h.id, w.wallet_user_id, h.product, h.reference_id, h.amount, h.status, h.idempotency_key, h.wallet_id, h.expires_at
		from wallet_holds h join wallets w on w.id = h.wallet_id
		where h.id = ? and h.product = ?`, request.HoldID, request.Product,
	).Scan(&hold.ID, &hold.WalletUserID, &hold.Product, &hold.ReferenceID, &hold.Amount,
		&hold.Status, &hold.IdempotencyKey, &walletID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return wallet.Hold{}, wallet.ErrHoldNotFound
	}
	if err != nil {
		return wallet.Hold{}, fmt.Errorf("baca hold: %w", err)
	}

	action := "capture"
	if target == "released" {
		action = "release"
	}

	var existingKey string
	err = tx.QueryRowContext(ctx,
		`select idempotency_key from wallet_hold_transitions where hold_id = ? and action = ?`, hold.ID, action,
	).Scan(&existingKey)
	if err == nil {
		if existingKey != request.IdempotencyKey {
			return wallet.Hold{}, wallet.ErrIdempotencyConflict
		}
		return hold, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return wallet.Hold{}, fmt.Errorf("cek idempotensi transisi: %w", err)
	}

	if hold.Status != "active" {
		return wallet.Hold{}, wallet.ErrHoldProcessed
	}
	if time.Now().Unix() >= expiresAt {
		if _, err := tx.ExecContext(ctx, `update wallet_holds set status = 'expired' where id = ?`, hold.ID); err != nil {
			return wallet.Hold{}, fmt.Errorf("tandai expired: %w", err)
		}
		return wallet.Hold{}, wallet.ErrHoldExpired
	}

	if _, err := tx.ExecContext(ctx,
		`insert into wallet_hold_transitions (id, hold_id, action, idempotency_key, created_at) values (?, ?, ?, ?, ?)`,
		newID(), hold.ID, action, request.IdempotencyKey, time.Now().Unix()); err != nil {
		return wallet.Hold{}, fmt.Errorf("catat transisi: %w", err)
	}

	if target == "captured" {
		var balanceAfter int64
		if err := tx.QueryRowContext(ctx,
			`select balance from wallets where id = ?`, walletID).Scan(&balanceAfter); err != nil {
			return wallet.Hold{}, fmt.Errorf("baca saldo: %w", err)
		}
		if balanceAfter < hold.Amount {
			return wallet.Hold{}, wallet.ErrInsufficientBalance
		}
		balanceAfter -= hold.Amount
		if _, err := tx.ExecContext(ctx,
			`update wallets set balance = ?, updated_at = ? where id = ?`,
			balanceAfter, time.Now().Unix(), walletID); err != nil {
			return wallet.Hold{}, fmt.Errorf("kurangi saldo: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			insert into wallet_ledger (id, wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key, created_at)
			values (?, ?, ?, ?, 'hold_capture', ?, ?, ?, ?)`,
			newID(), walletID, -hold.Amount, balanceAfter, hold.Product, hold.ReferenceID,
			request.IdempotencyKey, time.Now().Unix()); err != nil {
			return wallet.Hold{}, fmt.Errorf("catat ledger: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`update wallet_holds set status = ? where id = ?`, target, hold.ID); err != nil {
		return wallet.Hold{}, fmt.Errorf("ubah status hold: %w", err)
	}
	hold.Status = target
	if err := tx.Commit(); err != nil {
		return wallet.Hold{}, fmt.Errorf("simpan transisi: %w", err)
	}
	return hold, nil
}

// WalletBalance mengembalikan saldo tersimpan, tertahan, dan tersedia.
func (store *Store) WalletBalance(ctx context.Context, walletUserID string) (report.OverviewWallet, error) {
	var item report.OverviewWallet
	err := store.db.QueryRowContext(ctx, `
		select w.wallet_user_id, w.balance, w.currency,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallets w where w.wallet_user_id = ?`, walletUserID,
	).Scan(&item.WalletUserID, &item.Balance, &item.Currency, &item.HeldActive)
	if errors.Is(err, sql.ErrNoRows) {
		return report.OverviewWallet{}, wallet.ErrHoldNotFound
	}
	if err != nil {
		return report.OverviewWallet{}, fmt.Errorf("baca saldo: %w", err)
	}
	item.Available = item.Balance - item.HeldActive
	return item, nil
}

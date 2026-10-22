package database

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

// RecordAudit menyimpan tindakan admin.
func (store *Store) RecordAudit(ctx context.Context, entry sqlite.AuditEntry) error {
	var walletUserID any
	if strings.TrimSpace(entry.WalletUserID) != "" {
		walletUserID = entry.WalletUserID
	}
	actor := entry.Actor
	if strings.TrimSpace(actor) == "" {
		actor = "admin"
	}
	_, err := store.pool.Exec(ctx, `
		insert into wallet_admin_audit (actor, action, wallet_user_id, reference_id, details)
		values ($1, $2, $3, $4, $5)`,
		actor, entry.Action, walletUserID, entry.ReferenceID, entry.Details)
	if err != nil {
		return fmt.Errorf("catat audit: %w", err)
	}
	return nil
}

// AuditLog mengambil catatan audit terbaru.
func (store *Store) AuditLog(ctx context.Context, limit int) ([]sqlite.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.pool.Query(ctx, `
		select id::text, actor, action, coalesce(wallet_user_id::text,''), reference_id, details, created_at
		from wallet_admin_audit order by created_at desc limit $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("baca audit: %w", err)
	}
	defer rows.Close()

	var entries []sqlite.AuditEntry
	for rows.Next() {
		var entry sqlite.AuditEntry
		if err := rows.Scan(&entry.ID, &entry.Actor, &entry.Action, &entry.WalletUserID,
			&entry.ReferenceID, &entry.Details, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("baca audit: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// AdjustBalance menambah/mengurangi saldo atas perintah admin.
func (store *Store) AdjustBalance(ctx context.Context, walletUserID string, amount int64, reason, actor, referenceID string) (int64, error) {
	if amount == 0 {
		return 0, fmt.Errorf("nominal penyesuaian tidak boleh nol")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return 0, fmt.Errorf("alasan wajib diisi")
	}
	if strings.TrimSpace(referenceID) == "" {
		referenceID = "adjust-" + randomSuffix()
	}

	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("mulai penyesuaian: %w", err)
	}
	defer transaction.Rollback(ctx)

	idempotencyKey := "adjust:" + referenceID
	var existing int
	if err := transaction.QueryRow(ctx,
		`select count(*) from wallet_ledger where idempotency_key = $1`, idempotencyKey).Scan(&existing); err != nil {
		return 0, fmt.Errorf("cek idempotensi: %w", err)
	}
	if existing > 0 {
		var balance int64
		transaction.QueryRow(ctx, `select balance from wallets where wallet_user_id = $1::uuid`, walletUserID).Scan(&balance)
		return balance, transaction.Commit(ctx)
	}

	var walletID string
	var balance int64
	err = transaction.QueryRow(ctx,
		`select id::text, balance from wallets where wallet_user_id = $1::uuid for update`, walletUserID,
	).Scan(&walletID, &balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInvoiceNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("baca wallet: %w", err)
	}

	if balance+amount < 0 {
		return 0, fmt.Errorf("saldo tidak cukup: saldo %d, penyesuaian %d", balance, amount)
	}
	balance += amount

	if _, err := transaction.Exec(ctx,
		`update wallets set balance = $1, updated_at = now() where id = $2::uuid`, balance, walletID); err != nil {
		return 0, fmt.Errorf("ubah saldo: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		insert into wallet_ledger (wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key)
		values ($1::uuid, $2, $3, 'adjustment', 'admin', $4, $5)`,
		walletID, amount, balance, referenceID, idempotencyKey); err != nil {
		return 0, fmt.Errorf("catat ledger: %w", err)
	}
	if err := store.RecordAudit(ctx, sqlite.AuditEntry{
		Actor: actor, Action: "wallet.adjust", WalletUserID: walletUserID,
		ReferenceID: referenceID, Details: fmt.Sprintf("nominal %d; alasan: %s", amount, reason),
	}); err != nil {
		return 0, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return 0, fmt.Errorf("simpan penyesuaian: %w", err)
	}
	return balance, nil
}

func randomSuffix() string {
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		return "000000"
	}
	return fmt.Sprintf("%x", buffer)
}

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AuditEntry adalah satu catatan tindakan admin.
type AuditEntry struct {
	ID           string
	Actor        string
	Action       string
	WalletUserID string
	ReferenceID  string
	Details      string
	CreatedAt    time.Time
}

// RecordAudit menyimpan tindakan admin. Selalu dipanggil setelah aksi berhasil
// supaya jejak "siapa melakukan apa" tidak pernah hilang.
func (store *Store) RecordAudit(ctx context.Context, entry AuditEntry) error {
	var walletUserID any
	if strings.TrimSpace(entry.WalletUserID) != "" {
		walletUserID = entry.WalletUserID
	}
	_, err := store.db.ExecContext(ctx, `
		insert into wallet_admin_audit (id, actor, action, wallet_user_id, reference_id, details, created_at)
		values (?, ?, ?, ?, ?, ?, ?)`,
		newID(), defaultActor(entry.Actor), entry.Action, walletUserID, entry.ReferenceID, entry.Details, nowUnix())
	if err != nil {
		return fmt.Errorf("catat audit: %w", err)
	}
	return nil
}

// AuditLog mengambil catatan audit terbaru.
func (store *Store) AuditLog(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := store.db.QueryContext(ctx, `
		select id, actor, action, coalesce(wallet_user_id,''), reference_id, details, created_at
		from wallet_admin_audit order by created_at desc, rowid desc limit ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("baca audit: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var createdAt int64
		if err := rows.Scan(&entry.ID, &entry.Actor, &entry.Action, &entry.WalletUserID,
			&entry.ReferenceID, &entry.Details, &createdAt); err != nil {
			return nil, fmt.Errorf("baca audit: %w", err)
		}
		entry.CreatedAt = time.Unix(createdAt, 0)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// AdjustBalance menambah atau mengurangi saldo atas perintah admin.
// amount positif = tambah, negatif = kurangi. Selalu mencatat ledger dan audit.
func (store *Store) AdjustBalance(ctx context.Context, walletUserID string, amount int64, reason, actor, referenceID string) (int64, error) {
	if amount == 0 {
		return 0, fmt.Errorf("nominal penyesuaian tidak boleh nol")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return 0, fmt.Errorf("alasan wajib diisi")
	}
	if strings.TrimSpace(referenceID) == "" {
		referenceID = "adjust-" + newID()[:12]
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("mulai penyesuaian: %w", err)
	}
	defer tx.Rollback()

	// Idempotensi: key sama tidak menambah saldo dua kali.
	idempotencyKey := "adjust:" + referenceID
	var existing int
	if err := tx.QueryRowContext(ctx,
		`select count(*) from wallet_ledger where idempotency_key = ?`, idempotencyKey).Scan(&existing); err != nil {
		return 0, fmt.Errorf("cek idempotensi: %w", err)
	}
	if existing > 0 {
		var balance int64
		if err := tx.QueryRowContext(ctx,
			`select balance from wallets where wallet_user_id = ?`, walletUserID).Scan(&balance); err != nil {
			return 0, fmt.Errorf("baca saldo: %w", err)
		}
		return balance, nil
	}

	var walletID string
	var balance int64
	err = tx.QueryRowContext(ctx,
		`select id, balance from wallets where wallet_user_id = ?`, walletUserID).Scan(&walletID, &balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvoiceNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("baca wallet: %w", err)
	}

	if balance+amount < 0 {
		return 0, fmt.Errorf("saldo tidak cukup: saldo %d, penyesuaian %d", balance, amount)
	}
	balance += amount

	if _, err := tx.ExecContext(ctx,
		`update wallets set balance = ?, updated_at = ? where id = ?`, balance, nowUnix(), walletID); err != nil {
		return 0, fmt.Errorf("ubah saldo: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		insert into wallet_ledger (id, wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key, created_at)
		values (?, ?, ?, ?, 'adjustment', 'admin', ?, ?, ?)`,
		newID(), walletID, amount, balance, referenceID, idempotencyKey, nowUnix()); err != nil {
		return 0, fmt.Errorf("catat ledger: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		insert into wallet_admin_audit (id, actor, action, wallet_user_id, reference_id, details, created_at)
		values (?, ?, 'wallet.adjust', ?, ?, ?, ?)`,
		newID(), defaultActor(actor), walletUserID, referenceID,
		fmt.Sprintf("nominal %d; alasan: %s", amount, reason), nowUnix()); err != nil {
		return 0, fmt.Errorf("catat audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("simpan penyesuaian: %w", err)
	}
	return balance, nil
}

func defaultActor(actor string) string {
	if strings.TrimSpace(actor) == "" {
		return "admin"
	}
	return strings.TrimSpace(actor)
}

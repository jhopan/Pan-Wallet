package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/topup"
)

// ErrNoCodeAvailable muncul bila seluruh kode 1-999 untuk harga itu terpakai.
var ErrNoCodeAvailable = errors.New("kode invoice habis untuk nominal ini")

// ErrInvoiceNotFound dipakai saat invoice tidak ada.
var ErrInvoiceNotFound = errors.New("invoice tidak ditemukan")

// CreateTopupInvoice mengklaim total unik lalu menyimpan invoice top-up.
// Klaim atomik memanfaatkan unique index parsial pada total status pending.
func (store *Store) CreateTopupInvoice(ctx context.Context, walletUserID string, price int64, ttl time.Duration) (topup.Invoice, error) {
	if price < topup.MinPrice {
		return topup.Invoice{}, fmt.Errorf("nominal top-up minimal Rp%d", topup.MinPrice)
	}

	for attempt := 0; attempt < 30; attempt++ {
		invoice, err := store.tryCreateInvoice(ctx, walletUserID, price, ttl)
		if err == nil {
			return invoice, nil
		}
		if errors.Is(err, ErrNoCodeAvailable) {
			return topup.Invoice{}, err
		}
		// Bentrok klaim total (unique index) -> coba kode berikutnya.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			continue
		}
		return topup.Invoice{}, err
	}
	return topup.Invoice{}, ErrNoCodeAvailable
}

func (store *Store) tryCreateInvoice(ctx context.Context, walletUserID string, price int64, ttl time.Duration) (topup.Invoice, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("mulai invoice: %w", err)
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx,
		`select count(*) from wallets where wallet_user_id = ?`, walletUserID).Scan(&exists); err != nil {
		return topup.Invoice{}, fmt.Errorf("cek wallet: %w", err)
	}
	if exists == 0 {
		return topup.Invoice{}, ErrInvoiceNotFound
	}

	var used []int
	rows, err := tx.QueryContext(ctx,
		`select code from topup_invoices where price = ? and status = 'pending'`, price)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca kode terpakai: %w", err)
	}
	for rows.Next() {
		var code int
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return topup.Invoice{}, err
		}
		used = append(used, code)
	}
	rows.Close()

	var claimed []int64
	totalRows, err := tx.QueryContext(ctx, `select total from topup_invoices where status = 'pending'`)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca total terklaim: %w", err)
	}
	for totalRows.Next() {
		var total int64
		if err := totalRows.Scan(&total); err != nil {
			totalRows.Close()
			return topup.Invoice{}, err
		}
		claimed = append(claimed, total)
	}
	totalRows.Close()

	code := topup.AllocateCode(price, used, claimed)
	if code == 0 {
		return topup.Invoice{}, ErrNoCodeAvailable
	}

	invoice := topup.Invoice{
		ID:           newID(),
		WalletUserID: walletUserID,
		Price:        price,
		Code:         code,
		Total:        topup.Total(price, code),
		Status:       "pending",
		ExpiresAt:    time.Now().Add(ttl),
	}
	if _, err := tx.ExecContext(ctx, `
		insert into topup_invoices (id, wallet_user_id, price, code, total, status, source, expires_at, created_at)
		values (?, ?, ?, ?, ?, 'pending', '', ?, ?)`,
		invoice.ID, invoice.WalletUserID, invoice.Price, invoice.Code, invoice.Total,
		invoice.ExpiresAt.Unix(), nowUnix()); err != nil {
		return topup.Invoice{}, err
	}
	if err := tx.Commit(); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan invoice: %w", err)
	}
	return invoice, nil
}

// MarkTopupPaid menandai lunas dan mengkredit saldo tepat satu kali.
func (store *Store) MarkTopupPaid(ctx context.Context, invoiceID, source string) (topup.Invoice, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("mulai pelunasan: %w", err)
	}
	defer tx.Rollback()

	var invoice topup.Invoice
	var walletID string
	var expiresAt int64
	err = tx.QueryRowContext(ctx, `
		select i.id, i.wallet_user_id, i.price, i.code, i.total, i.status, i.expires_at, w.id
		from topup_invoices i join wallets w on w.wallet_user_id = i.wallet_user_id
		where i.id = ?`, invoiceID,
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code,
		&invoice.Total, &invoice.Status, &expiresAt, &walletID)
	if errors.Is(err, sql.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca invoice: %w", err)
	}
	invoice.ExpiresAt = time.Unix(expiresAt, 0)

	if invoice.Status == "paid" {
		return invoice, tx.Commit()
	}
	if !topup.CanTransition(invoice.Status, "paid") {
		return topup.Invoice{}, fmt.Errorf("invoice berstatus %s tidak bisa dilunasi", invoice.Status)
	}

	var balance int64
	if err := tx.QueryRowContext(ctx,
		`select balance from wallets where id = ?`, walletID).Scan(&balance); err != nil {
		return topup.Invoice{}, fmt.Errorf("baca saldo: %w", err)
	}
	balance += invoice.Total
	if _, err := tx.ExecContext(ctx,
		`update wallets set balance = ?, updated_at = ? where id = ?`, balance, nowUnix(), walletID); err != nil {
		return topup.Invoice{}, fmt.Errorf("kredit saldo: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		insert into wallet_ledger (id, wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key, created_at)
		values (?, ?, ?, ?, 'topup', 'topup-qris', ?, ?, ?)`,
		newID(), walletID, invoice.Total, balance, invoice.ID, "topup:"+invoice.ID, nowUnix()); err != nil {
		return topup.Invoice{}, fmt.Errorf("catat ledger: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`update topup_invoices set status = 'paid', paid_at = ?, source = ? where id = ?`,
		nowUnix(), topup.NormalizeSource(source), invoice.ID); err != nil {
		return topup.Invoice{}, fmt.Errorf("tandai lunas: %w", err)
	}
	invoice.Status = "paid"
	if err := tx.Commit(); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan pelunasan: %w", err)
	}
	return invoice, nil
}

// TopupInvoice mengambil satu invoice.
func (store *Store) TopupInvoice(ctx context.Context, invoiceID string) (topup.Invoice, error) {
	var invoice topup.Invoice
	var expiresAt int64
	err := store.db.QueryRowContext(ctx, `
		select id, wallet_user_id, price, code, total, status, source, expires_at
		from topup_invoices where id = ?`, invoiceID,
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code,
		&invoice.Total, &invoice.Status, &invoice.Source, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca invoice: %w", err)
	}
	invoice.ExpiresAt = time.Unix(expiresAt, 0)
	return invoice, nil
}

// RecentTopupInvoices mengambil invoice terbaru untuk panel admin.
func (store *Store) RecentTopupInvoices(ctx context.Context, limit int) ([]topup.Invoice, error) {
	rows, err := store.db.QueryContext(ctx, `
		select id, wallet_user_id, price, code, total, status, source, expires_at
		from topup_invoices order by created_at desc limit ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("daftar invoice: %w", err)
	}
	defer rows.Close()

	var invoices []topup.Invoice
	for rows.Next() {
		var invoice topup.Invoice
		var expiresAt int64
		if err := rows.Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code,
			&invoice.Total, &invoice.Status, &invoice.Source, &expiresAt); err != nil {
			return nil, fmt.Errorf("baca invoice: %w", err)
		}
		invoice.ExpiresAt = time.Unix(expiresAt, 0)
		invoices = append(invoices, invoice)
	}
	return invoices, rows.Err()
}

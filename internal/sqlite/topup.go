package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/paypan"
	"github.com/jhopan/Pan-Wallet/internal/topup"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

// ErrInvoiceNotFound dipakai saat invoice tidak ada.
var ErrInvoiceNotFound = errors.New("invoice tidak ditemukan")

// CreateTopupInvoice meminta invoice ke PayPan lalu menyimpan catatan lokal.
// Pan-Wallet tidak menghitung kode/total sendiri: PayPan yang menentukan.
func (store *Store) CreateTopupInvoice(ctx context.Context, client *paypan.Client, walletUserID string, price int64) (topup.Invoice, error) {
	if price < topup.MinPrice {
		return topup.Invoice{}, fmt.Errorf("nominal top-up minimal Rp25.000")
	}
	if price > topup.MaxPrice {
		return topup.Invoice{}, fmt.Errorf("nominal top-up maksimal Rp%d", topup.MaxPrice)
	}

	var status string
	err := store.db.QueryRowContext(ctx, `
		select u.status from wallets w join wallet_users u on u.id = w.wallet_user_id
		where w.wallet_user_id = ?`, walletUserID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("cek wallet: %w", err)
	}
	if status != "active" {
		return topup.Invoice{}, wallet.ErrWalletBlocked
	}

	remote, err := client.CreateInvoice(ctx, price, "topup "+walletUserID)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("PayPan: %w", err)
	}

	invoice := topup.Invoice{
		ID:            newID(),
		WalletUserID:  walletUserID,
		PaypanInvoice: remote.ID,
		Price:         remote.Price,
		Code:          remote.Code,
		Total:         remote.Total,
		Status:        topup.PaypanToLocalStatus(remote.Status),
		ExpiresAt:     time.Unix(remote.ExpiresAt, 0),
		PayURL:        remote.PayURL,
	}
	if invoice.Price == 0 {
		invoice.Price = price
	}
	if invoice.Total == 0 {
		invoice.Total = price
	}

	if _, err := store.db.ExecContext(ctx, `
		insert into topup_invoices (id, wallet_user_id, paypan_invoice_id, price, code, total, status, source, expires_at, created_at)
		values (?, ?, ?, ?, ?, ?, 'pending', '', ?, ?)`,
		invoice.ID, invoice.WalletUserID, invoice.PaypanInvoice, invoice.Price,
		invoice.Code, invoice.Total, invoice.ExpiresAt.Unix(), nowUnix()); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan invoice: %w", err)
	}
	return invoice, nil
}

// MarkTopupPaidByPaypan menandai lunas berdasarkan id invoice PayPan dan
// mengkredit saldo tepat satu kali. Dipakai jalur webhook maupun rekonsiliasi.
func (store *Store) MarkTopupPaidByPaypan(ctx context.Context, paypanInvoiceID, source string) (topup.Invoice, error) {
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
		where i.paypan_invoice_id = ?`, paypanInvoiceID,
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
		// Sudah pernah dilunasi: webhook ganda / retry PayPan.
		// Jangan kredit ulang, dan jangan kirim notifikasi lagi.
		invoice.BalanceAfter = balanceOf(ctx, tx, walletID)
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
	// Total yang masuk saldo harus persis price + kode unik.
	// Kalau tidak cocok, jangan kredit sama sekali — lebih baik gagal
	// daripada pelanggan kurang 1 rupiah.
	if invoice.Total != invoice.Price+int64(invoice.Code) {
		return topup.Invoice{}, fmt.Errorf(
			"total invoice tidak konsisten: total %d, price %d, kode %d",
			invoice.Total, invoice.Price, invoice.Code)
	}

	balance += invoice.Total
	if _, err := tx.ExecContext(ctx,
		`update wallets set balance = ?, updated_at = ? where id = ?`, balance, nowUnix(), walletID); err != nil {
		return topup.Invoice{}, fmt.Errorf("kredit saldo: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		insert into wallet_ledger (id, wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key, created_at)
		values (?, ?, ?, ?, 'topup', 'topup-qris', ?, ?, ?)`,
		newID(), walletID, invoice.Total, balance, paypanInvoiceID,
		"topup:"+paypanInvoiceID, nowUnix()); err != nil {
		return topup.Invoice{}, fmt.Errorf("catat ledger: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`update topup_invoices set status = 'paid', paid_at = ?, source = ? where id = ?`,
		nowUnix(), topup.NormalizeSource(source), invoice.ID); err != nil {
		return topup.Invoice{}, fmt.Errorf("tandai lunas: %w", err)
	}
	invoice.Status = "paid"
	invoice.BalanceAfter = balance
	invoice.FreshlyPaid = true
	if err := tx.Commit(); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan pelunasan: %w", err)
	}
	return invoice, nil
}

// MarkTopupPaidByLocalID dipakai panel admin saat menandai lunas manual.
func (store *Store) MarkTopupPaidByLocalID(ctx context.Context, localID, source string) (topup.Invoice, error) {
	var paypanID string
	err := store.db.QueryRowContext(ctx,
		`select coalesce(paypan_invoice_id,'') from topup_invoices where id = ?`, localID).Scan(&paypanID)
	if errors.Is(err, sql.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("cari invoice: %w", err)
	}
	if paypanID == "" {
		return topup.Invoice{}, fmt.Errorf("invoice tanpa referensi PayPan")
	}
	return store.MarkTopupPaidByPaypan(ctx, paypanID, source)
}

// TopupInvoice mengambil satu invoice berdasarkan id lokal.
func (store *Store) TopupInvoice(ctx context.Context, invoiceID string) (topup.Invoice, error) {
	return store.scanInvoice(ctx,
		`select id, wallet_user_id, coalesce(paypan_invoice_id,''), price, code, total, status, source, expires_at
		 from topup_invoices where id = ?`, invoiceID)
}

// TopupInvoiceByPaypan mengambil invoice dari id PayPan (untuk webhook).
func (store *Store) TopupInvoiceByPaypan(ctx context.Context, paypanInvoiceID string) (topup.Invoice, error) {
	return store.scanInvoice(ctx,
		`select id, wallet_user_id, coalesce(paypan_invoice_id,''), price, code, total, status, source, expires_at
		 from topup_invoices where paypan_invoice_id = ?`, paypanInvoiceID)
}

func (store *Store) scanInvoice(ctx context.Context, query string, args ...any) (topup.Invoice, error) {
	var invoice topup.Invoice
	var expiresAt int64
	err := store.db.QueryRowContext(ctx, query, args...).Scan(
		&invoice.ID, &invoice.WalletUserID, &invoice.PaypanInvoice, &invoice.Price,
		&invoice.Code, &invoice.Total, &invoice.Status, &invoice.Source, &expiresAt)
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
		select id, wallet_user_id, coalesce(paypan_invoice_id,''), price, code, total, status, source, expires_at
		from topup_invoices order by created_at desc limit ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("daftar invoice: %w", err)
	}
	defer rows.Close()

	var invoices []topup.Invoice
	for rows.Next() {
		var invoice topup.Invoice
		var expiresAt int64
		if err := rows.Scan(&invoice.ID, &invoice.WalletUserID, &invoice.PaypanInvoice, &invoice.Price,
			&invoice.Code, &invoice.Total, &invoice.Status, &invoice.Source, &expiresAt); err != nil {
			return nil, fmt.Errorf("baca invoice: %w", err)
		}
		invoice.ExpiresAt = time.Unix(expiresAt, 0)
		invoices = append(invoices, invoice)
	}
	return invoices, rows.Err()
}

// ExpireTopupInvoice menandai invoice pending sebagai kedaluwarsa.
func (store *Store) ExpireTopupInvoice(ctx context.Context, localID string) error {
	_, err := store.db.ExecContext(ctx,
		`update topup_invoices set status = 'expired' where id = ? and status = 'pending'`, localID)
	return err
}

// PendingTopupInvoices mengambil invoice pending yang lebih tua dari batas,
// untuk dicek ulang ke PayPan oleh rekonsiliasi.
func (store *Store) PendingTopupInvoices(ctx context.Context, olderThan time.Time) ([]topup.Invoice, error) {
	rows, err := store.db.QueryContext(ctx, `
		select id, wallet_user_id, coalesce(paypan_invoice_id,''), price, code, total, status, source, expires_at
		from topup_invoices
		where status = 'pending' and created_at < ?
		order by created_at limit 50`, olderThan.Unix())
	if err != nil {
		return nil, fmt.Errorf("daftar invoice pending: %w", err)
	}
	defer rows.Close()

	var invoices []topup.Invoice
	for rows.Next() {
		var invoice topup.Invoice
		var expiresAt int64
		if err := rows.Scan(&invoice.ID, &invoice.WalletUserID, &invoice.PaypanInvoice, &invoice.Price,
			&invoice.Code, &invoice.Total, &invoice.Status, &invoice.Source, &expiresAt); err != nil {
			return nil, fmt.Errorf("baca invoice: %w", err)
		}
		invoice.ExpiresAt = time.Unix(expiresAt, 0)
		invoices = append(invoices, invoice)
	}
	return invoices, rows.Err()
}

// balanceOf membaca saldo wallet di dalam transaksi.
func balanceOf(ctx context.Context, tx *sql.Tx, walletID string) int64 {
	var balance int64
	_ = tx.QueryRowContext(ctx, `select balance from wallets where id = ?`, walletID).Scan(&balance)
	return balance
}

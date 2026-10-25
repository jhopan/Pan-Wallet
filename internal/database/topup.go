package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/PanWallet/internal/paypan"
	"github.com/jhopan/PanWallet/internal/topup"
)

// ErrInvoiceNotFound dipakai saat invoice tidak ada.
var ErrInvoiceNotFound = errors.New("invoice tidak ditemukan")

// CreateTopupInvoice meminta invoice ke PayPan lalu menyimpan catatan lokal.
func (store *Store) CreateTopupInvoice(ctx context.Context, client *paypan.Client, walletUserID string, price int64) (topup.Invoice, error) {
	if price < topup.MinPrice {
		return topup.Invoice{}, fmt.Errorf("nominal top-up minimal Rp25.000")
	}
	if price > topup.MaxPrice {
		return topup.Invoice{}, fmt.Errorf("nominal top-up maksimal Rp%d", topup.MaxPrice)
	}

	var walletID string
	err := store.pool.QueryRow(ctx,
		`select id::text from wallets where wallet_user_id = $1::uuid`, walletUserID).Scan(&walletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("cari wallet: %w", err)
	}

	remote, err := client.CreateInvoice(ctx, price, "topup "+walletUserID)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("PayPan: %w", err)
	}

	var invoice topup.Invoice
	err = store.pool.QueryRow(ctx, `
		insert into topup_invoices (wallet_user_id, paypan_invoice_id, price, code, total, status, expires_at)
		values ($1::uuid, $2, $3, $4, $5, 'pending', $6)
		returning id::text, wallet_user_id::text, paypan_invoice_id, price, code, total, status, expires_at`,
		walletUserID, remote.ID, remote.Price, remote.Code, remote.Total, time.Unix(remote.ExpiresAt, 0),
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.PaypanInvoice, &invoice.Price,
		&invoice.Code, &invoice.Total, &invoice.Status, &invoice.ExpiresAt)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan invoice: %w", err)
	}
	invoice.PayURL = remote.PayURL
	return invoice, nil
}

// MarkTopupPaidByPaypan menandai lunas dan mengkredit saldo sekali saja.
func (store *Store) MarkTopupPaidByPaypan(ctx context.Context, paypanInvoiceID, source string) (topup.Invoice, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("mulai pelunasan: %w", err)
	}
	defer transaction.Rollback(ctx)

	var invoice topup.Invoice
	var walletID string
	err = transaction.QueryRow(ctx, `
		select i.id::text, i.wallet_user_id::text, i.price, i.code, i.total, i.status, i.expires_at, w.id::text
		from topup_invoices i join wallets w on w.wallet_user_id = i.wallet_user_id
		where i.paypan_invoice_id = $1 for update of i`, paypanInvoiceID,
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code,
		&invoice.Total, &invoice.Status, &invoice.ExpiresAt, &walletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("kunci invoice: %w", err)
	}
	if invoice.Status == "paid" {
		return invoice, transaction.Commit(ctx)
	}
	if !topup.CanTransition(invoice.Status, "paid") {
		return topup.Invoice{}, fmt.Errorf("invoice berstatus %s tidak bisa dilunasi", invoice.Status)
	}

	var balanceAfter int64
	if err := transaction.QueryRow(ctx, `
		update wallets set balance = balance + $1, updated_at = now()
		where id = $2::uuid returning balance`, invoice.Total, walletID,
	).Scan(&balanceAfter); err != nil {
		return topup.Invoice{}, fmt.Errorf("kredit saldo: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		insert into wallet_ledger (wallet_id, amount, balance_after, entry_type, product, reference_id, idempotency_key)
		values ($1::uuid, $2, $3, 'topup', 'topup-qris', $4, $5)`,
		walletID, invoice.Total, balanceAfter, paypanInvoiceID, "topup:"+paypanInvoiceID,
	); err != nil {
		return topup.Invoice{}, fmt.Errorf("catat ledger: %w", err)
	}
	if _, err := transaction.Exec(ctx,
		`update topup_invoices set status = 'paid', paid_at = now(), source = $2 where id = $1::uuid`,
		invoice.ID, topup.NormalizeSource(source)); err != nil {
		return topup.Invoice{}, fmt.Errorf("tandai lunas: %w", err)
	}
	invoice.Status = "paid"
	if err := transaction.Commit(ctx); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan pelunasan: %w", err)
	}
	return invoice, nil
}

// MarkTopupPaidByLocalID dipakai panel admin saat menandai lunas manual.
func (store *Store) MarkTopupPaidByLocalID(ctx context.Context, localID, source string) (topup.Invoice, error) {
	var paypanID string
	err := store.pool.QueryRow(ctx,
		`select coalesce(paypan_invoice_id,'') from topup_invoices where id = $1::uuid`, localID).Scan(&paypanID)
	if errors.Is(err, pgx.ErrNoRows) {
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

// ExpireTopupInvoice menandai invoice pending sebagai kedaluwarsa.
func (store *Store) ExpireTopupInvoice(ctx context.Context, localID string) error {
	_, err := store.pool.Exec(ctx,
		`update topup_invoices set status = 'expired' where id = $1::uuid and status = 'pending'`, localID)
	return err
}

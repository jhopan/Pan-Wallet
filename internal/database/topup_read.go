package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/Pan-Wallet/internal/topup"
)

// TopupInvoice mengambil satu invoice berdasarkan id.
func (store *Store) TopupInvoice(ctx context.Context, invoiceID string) (topup.Invoice, error) {
	var invoice topup.Invoice
	err := store.pool.QueryRow(ctx, `
		select id::text, wallet_user_id::text, price, code, total, status, source, expires_at
		from topup_invoices where id = $1::uuid`, invoiceID,
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code, &invoice.Total,
		&invoice.Status, &invoice.Source, &invoice.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	}
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca invoice: %w", err)
	}
	return invoice, nil
}

// RecentTopupInvoices mengambil invoice terbaru untuk panel admin.
func (store *Store) RecentTopupInvoices(ctx context.Context, limit int) ([]topup.Invoice, error) {
	rows, err := store.pool.Query(ctx, `
		select id::text, wallet_user_id::text, price, code, total, status, source, expires_at
		from topup_invoices order by created_at desc limit $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("daftar invoice: %w", err)
	}
	defer rows.Close()

	var invoices []topup.Invoice
	for rows.Next() {
		var invoice topup.Invoice
		if err := rows.Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code,
			&invoice.Total, &invoice.Status, &invoice.Source, &invoice.ExpiresAt); err != nil {
			return nil, fmt.Errorf("baca invoice: %w", err)
		}
		invoices = append(invoices, invoice)
	}
	return invoices, rows.Err()
}

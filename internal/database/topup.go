package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jhopan/Pan-Wallet/internal/topup"
)

// ErrNoCodeAvailable muncul saat semua kode 1-999 untuk harga itu sudah dipakai.
var ErrNoCodeAvailable = errors.New("kode invoice habis untuk nominal ini")

// ErrInvoiceNotFound dipakai saat invoice tidak ada atau bukan milik wallet.
var ErrInvoiceNotFound = errors.New("invoice tidak ditemukan")

// CreateTopupInvoice mengklaim satu total unik untuk wallet lalu menyimpan
// invoice. Klaim atomik: kalau ada balapan UNIQUE pada total, coba kode lain.
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Balapan klaim total: coba kode berikutnya.
			continue
		}
		return topup.Invoice{}, err
	}
	return topup.Invoice{}, ErrNoCodeAvailable
}

func (store *Store) tryCreateInvoice(ctx context.Context, walletUserID string, price int64, ttl time.Duration) (topup.Invoice, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("mulai invoice: %w", err)
	}
	defer transaction.Rollback(ctx)

	var walletID string
	if err := transaction.QueryRow(ctx,
		`select id::text from wallets where wallet_user_id = $1::uuid`, walletUserID,
	).Scan(&walletID); errors.Is(err, pgx.ErrNoRows) {
		return topup.Invoice{}, ErrInvoiceNotFound
	} else if err != nil {
		return topup.Invoice{}, fmt.Errorf("cari wallet: %w", err)
	}

	// Kode yang sudah dipakai untuk harga sama (invoice pending).
	usedRows, err := transaction.Query(ctx,
		`select code from topup_invoices where price = $1 and status = 'pending'`, price)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca kode terpakai: %w", err)
	}
	var used []int
	for usedRows.Next() {
		var code int
		if err := usedRows.Scan(&code); err != nil {
			usedRows.Close()
			return topup.Invoice{}, err
		}
		used = append(used, code)
	}
	usedRows.Close()

	// Semua total yang masih diklaim invoice pending (lintas level harga).
	totalRows, err := transaction.Query(ctx,
		`select total from topup_invoices where status = 'pending'`)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("baca total terklaim: %w", err)
	}
	var claimed []int64
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
	total := topup.Total(price, code)

	var invoice topup.Invoice
	err = transaction.QueryRow(ctx, `
		insert into topup_invoices (wallet_user_id, price, code, total, status, expires_at)
		values ($1::uuid, $2, $3, $4, 'pending', $5)
		returning id::text, wallet_user_id::text, price, code, total, status, expires_at`,
		walletUserID, price, code, total, time.Now().Add(ttl),
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code, &invoice.Total, &invoice.Status, &invoice.ExpiresAt)
	if err != nil {
		return topup.Invoice{}, err
	}

	if err := transaction.Commit(ctx); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan invoice: %w", err)
	}
	return invoice, nil
}

// MarkTopupPaid menandai invoice lunas dan mengkredit saldo sekali saja.
// Idempotent: invoice yang sudah paid mengembalikan hasil tanpa kredit ulang.
func (store *Store) MarkTopupPaid(ctx context.Context, invoiceID, source string) (topup.Invoice, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return topup.Invoice{}, fmt.Errorf("mulai lunas: %w", err)
	}
	defer transaction.Rollback(ctx)

	var invoice topup.Invoice
	var walletID string
	err = transaction.QueryRow(ctx, `
		select i.id::text, i.wallet_user_id::text, i.price, i.code, i.total, i.status, i.expires_at, w.id::text
		from topup_invoices i join wallets w on w.wallet_user_id = i.wallet_user_id
		where i.id = $1::uuid for update of i`, invoiceID,
	).Scan(&invoice.ID, &invoice.WalletUserID, &invoice.Price, &invoice.Code, &invoice.Total, &invoice.Status, &invoice.ExpiresAt, &walletID)
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
		walletID, invoice.Total, balanceAfter, invoice.ID, "topup:"+invoice.ID,
	); err != nil {
		return topup.Invoice{}, fmt.Errorf("catat ledger topup: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
		update topup_invoices set status = 'paid', paid_at = now(), source = $2 where id = $1::uuid`,
		invoice.ID, topup.NormalizeSource(source),
	); err != nil {
		return topup.Invoice{}, fmt.Errorf("tandai lunas: %w", err)
	}
	invoice.Status = "paid"
	if err := transaction.Commit(ctx); err != nil {
		return topup.Invoice{}, fmt.Errorf("simpan pelunasan: %w", err)
	}
	return invoice, nil
}

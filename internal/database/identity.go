package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jhopan/Pan-Wallet/internal/identity"
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
)

// ErrIdentityTaken dipakai saat channel sudah tertaut ke wallet lain.
var ErrIdentityTaken = errors.New("identitas sudah tertaut ke akun lain")

// ErrCodeInvalid dipakai saat kode ikat salah atau kedaluwarsa.
var ErrCodeInvalid = errors.New("kode tidak valid atau sudah kedaluwarsa")

// ResolveIdentity mencari wallet lewat channel; Telegram boleh membuat baru.
func (store *Store) ResolveIdentity(ctx context.Context, provider, externalID string) (sqlite.WalletRow, bool, error) {
	normalized, err := identity.Normalize(provider, externalID)
	if err != nil {
		return sqlite.WalletRow{}, false, err
	}

	if row, err := store.identityWallet(ctx, provider, normalized); err == nil {
		return row, false, nil
	} else if !errors.Is(err, ErrInvoiceNotFound) {
		return sqlite.WalletRow{}, false, err
	}

	if !identity.IsPrimary(provider) {
		return sqlite.WalletRow{}, false, ErrInvoiceNotFound
	}

	userID, err := store.CreateWalletUser(ctx, 0)
	if err != nil {
		return sqlite.WalletRow{}, false, err
	}
	if _, err := store.pool.Exec(ctx,
		`insert into identity_links (wallet_user_id, provider, external_id) values ($1::uuid, $2, $3)`,
		userID, provider, normalized); err != nil {
		return sqlite.WalletRow{}, false, fmt.Errorf("tautkan identitas: %w", err)
	}
	row, err := store.WalletBalanceByUserID(ctx, userID)
	if err != nil {
		return sqlite.WalletRow{}, false, err
	}
	return row, true, nil
}

func (store *Store) identityWallet(ctx context.Context, provider, externalID string) (sqlite.WalletRow, error) {
	var userID string
	err := store.pool.QueryRow(ctx,
		`select wallet_user_id::text from identity_links where provider = $1 and external_id = $2`,
		provider, externalID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlite.WalletRow{}, ErrInvoiceNotFound
	}
	if err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("cari identitas: %w", err)
	}
	return store.WalletBalanceByUserID(ctx, userID)
}

// WalletBalanceByUserID membaca saldo lewat wallet_user_id internal.
func (store *Store) WalletBalanceByUserID(ctx context.Context, userID string) (sqlite.WalletRow, error) {
	var row sqlite.WalletRow
	err := store.pool.QueryRow(ctx, `
		select u.id::text, u.public_id, u.status, w.balance, w.currency, u.created_at,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallet_users u join wallets w on w.wallet_user_id = u.id
		where u.id = $1::uuid`, userID,
	).Scan(&row.WalletUserID, &row.PublicID, &row.Status, &row.Balance, &row.Currency, &row.CreatedAt, &row.HeldActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlite.WalletRow{}, ErrInvoiceNotFound
	}
	if err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("baca wallet: %w", err)
	}
	row.Available = row.Balance - row.HeldActive
	return row, nil
}

// StartIdentityLink menerbitkan kode untuk mengikat channel baru.
func (store *Store) StartIdentityLink(ctx context.Context, provider, externalID string) (string, error) {
	normalized, err := identity.Normalize(provider, externalID)
	if err != nil {
		return "", err
	}
	if identity.IsPrimary(provider) {
		return "", fmt.Errorf("telegram tidak perlu diikat")
	}
	if _, err := store.identityWallet(ctx, provider, normalized); err == nil {
		return "", ErrIdentityTaken
	} else if !errors.Is(err, ErrInvoiceNotFound) {
		return "", err
	}

	for attempt := 0; attempt < 5; attempt++ {
		code, err := identity.GenerateCode()
		if err != nil {
			return "", err
		}
		_, err = store.pool.Exec(ctx, `
			insert into identity_link_codes (code, provider, external_id, expires_at)
			values ($1, $2, $3, now() + interval '10 minutes')`, code, provider, normalized)
		if err == nil {
			return code, nil
		}
	}
	return "", fmt.Errorf("gagal membuat kode unik")
}

// ConfirmIdentityLink menautkan channel ke wallet milik telegramID.
func (store *Store) ConfirmIdentityLink(ctx context.Context, code, telegramID string) (sqlite.WalletRow, error) {
	normalizedTelegram, err := identity.Normalize(identity.ProviderTelegram, telegramID)
	if err != nil {
		return sqlite.WalletRow{}, err
	}
	owner, err := store.identityWallet(ctx, identity.ProviderTelegram, normalizedTelegram)
	if err != nil {
		if errors.Is(err, ErrInvoiceNotFound) {
			return sqlite.WalletRow{}, fmt.Errorf("telegram belum terdaftar")
		}
		return sqlite.WalletRow{}, err
	}

	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("mulai ikat: %w", err)
	}
	defer transaction.Rollback(ctx)

	var provider, externalID string
	var expiresAt time.Time
	var usedAt *time.Time
	err = transaction.QueryRow(ctx, `
		select provider, external_id, expires_at, used_at from identity_link_codes
		where code = $1 for update`, code).Scan(&provider, &externalID, &expiresAt, &usedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlite.WalletRow{}, ErrCodeInvalid
	}
	if err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("baca kode: %w", err)
	}
	if usedAt != nil || time.Now().After(expiresAt) {
		return sqlite.WalletRow{}, ErrCodeInvalid
	}

	var existingOwner string
	err = transaction.QueryRow(ctx,
		`select wallet_user_id::text from identity_links where provider = $1 and external_id = $2`,
		provider, externalID).Scan(&existingOwner)
	if err == nil && existingOwner != owner.WalletUserID {
		return sqlite.WalletRow{}, ErrIdentityTaken
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return sqlite.WalletRow{}, fmt.Errorf("cek pemilik: %w", err)
	}

	if err != nil {
		if _, err := transaction.Exec(ctx,
			`insert into identity_links (wallet_user_id, provider, external_id) values ($1::uuid, $2, $3)`,
			owner.WalletUserID, provider, externalID); err != nil {
			return sqlite.WalletRow{}, fmt.Errorf("tautkan identitas: %w", err)
		}
	}
	if _, err := transaction.Exec(ctx,
		`update identity_link_codes set used_at = now() where code = $1`, code); err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("tandai kode: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return sqlite.WalletRow{}, fmt.Errorf("simpan tautan: %w", err)
	}
	return owner, nil
}

// ListIdentities menampilkan channel yang tertaut ke satu wallet.
func (store *Store) ListIdentities(ctx context.Context, walletUserID string) ([]string, error) {
	rows, err := store.pool.Query(ctx,
		`select provider || ':' || external_id from identity_links
		 where wallet_user_id = $1::uuid order by provider`, walletUserID)
	if err != nil {
		return nil, fmt.Errorf("daftar identitas: %w", err)
	}
	defer rows.Close()

	var links []string
	for rows.Next() {
		var link string
		if err := rows.Scan(&link); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

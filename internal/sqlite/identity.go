package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/identity"
)

// ErrIdentityTaken dipakai saat channel sudah tertaut ke wallet lain.
var ErrIdentityTaken = errors.New("identitas sudah tertaut ke akun lain")

// ErrCodeInvalid dipakai saat kode ikat salah, kedaluwarsa, atau sudah dipakai.
var ErrCodeInvalid = errors.New("kode tidak valid atau sudah kedaluwarsa")

const linkCodeTTL = 10 * time.Minute

// ResolveIdentity mencari wallet lewat channel + external_id.
// Kalau belum ada DAN provider adalah Telegram (identitas primer), wallet baru
// dibuat otomatis. Channel lain harus ditautkan lewat kode.
func (store *Store) ResolveIdentity(ctx context.Context, provider, externalID string) (WalletRow, bool, error) {
	normalized, err := identity.Normalize(provider, externalID)
	if err != nil {
		return WalletRow{}, false, err
	}

	if row, err := store.identityWallet(ctx, provider, normalized); err == nil {
		return row, false, nil
	} else if !errors.Is(err, ErrInvoiceNotFound) {
		return WalletRow{}, false, err
	}

	// Belum tertaut. Hanya Telegram yang boleh membuat wallet otomatis.
	if !identity.IsPrimary(provider) {
		return WalletRow{}, false, ErrInvoiceNotFound
	}

	userID, err := store.CreateWalletUser(ctx, 0)
	if err != nil {
		return WalletRow{}, false, err
	}
	if _, err := store.db.ExecContext(ctx,
		`insert into identity_links (id, wallet_user_id, provider, external_id, verified_at, created_at)
		 values (?, ?, ?, ?, ?, ?)`,
		newID(), userID, provider, normalized, nowUnix(), nowUnix()); err != nil {
		return WalletRow{}, false, fmt.Errorf("tautkan identitas: %w", err)
	}

	row, err := store.WalletBalanceByUserID(ctx, userID)
	if err != nil {
		return WalletRow{}, false, err
	}
	return row, true, nil
}

// identityWallet mencari wallet yang sudah tertaut ke channel tertentu.
func (store *Store) identityWallet(ctx context.Context, provider, externalID string) (WalletRow, error) {
	var userID string
	err := store.db.QueryRowContext(ctx,
		`select wallet_user_id from identity_links where provider = ? and external_id = ?`,
		provider, externalID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return WalletRow{}, ErrInvoiceNotFound
	}
	if err != nil {
		return WalletRow{}, fmt.Errorf("cari identitas: %w", err)
	}
	return store.WalletBalanceByUserID(ctx, userID)
}

// WalletBalanceByUserID membaca saldo lewat wallet_user_id internal.
func (store *Store) WalletBalanceByUserID(ctx context.Context, userID string) (WalletRow, error) {
	var row WalletRow
	var createdAt int64
	err := store.db.QueryRowContext(ctx, `
		select u.id, u.public_id, u.status, w.balance, w.currency, u.created_at,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallet_users u join wallets w on w.wallet_user_id = u.id
		where u.id = ?`, userID,
	).Scan(&row.WalletUserID, &row.PublicID, &row.Status, &row.Balance, &row.Currency, &createdAt, &row.HeldActive)
	if errors.Is(err, sql.ErrNoRows) {
		return WalletRow{}, ErrInvoiceNotFound
	}
	if err != nil {
		return WalletRow{}, fmt.Errorf("baca wallet: %w", err)
	}
	row.CreatedAt = time.Unix(createdAt, 0)
	row.Available = row.Balance - row.HeldActive
	return row, nil
}

// StartIdentityLink menerbitkan kode 6 angka untuk mengikat channel baru.
// Kode dikonfirmasi lewat Telegram, jadi email/WA tidak perlu kirim pesan.
func (store *Store) StartIdentityLink(ctx context.Context, provider, externalID string) (string, error) {
	normalized, err := identity.Normalize(provider, externalID)
	if err != nil {
		return "", err
	}
	if identity.IsPrimary(provider) {
		return "", fmt.Errorf("telegram tidak perlu diikat, dia identitas utama")
	}

	// Kalau sudah tertaut, tidak perlu kode baru.
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
		_, err = store.db.ExecContext(ctx, `
			insert into identity_link_codes (code, provider, external_id, expires_at, created_at)
			values (?, ?, ?, ?, ?)`,
			code, provider, normalized, time.Now().Add(linkCodeTTL).Unix(), nowUnix())
		if err == nil {
			return code, nil
		}
		if !strings.Contains(err.Error(), "UNIQUE") {
			return "", fmt.Errorf("simpan kode: %w", err)
		}
	}
	return "", fmt.Errorf("gagal membuat kode unik")
}

// ConfirmIdentityLink menautkan channel ke wallet milik telegramID.
// Kode hanya bisa dipakai sekali dan tidak bisa merebut identitas wallet lain.
func (store *Store) ConfirmIdentityLink(ctx context.Context, code, telegramID string) (WalletRow, error) {
	normalizedTelegram, err := identity.Normalize(identity.ProviderTelegram, telegramID)
	if err != nil {
		return WalletRow{}, err
	}

	owner, err := store.identityWallet(ctx, identity.ProviderTelegram, normalizedTelegram)
	if err != nil {
		if errors.Is(err, ErrInvoiceNotFound) {
			return WalletRow{}, fmt.Errorf("telegram belum terdaftar")
		}
		return WalletRow{}, err
	}

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return WalletRow{}, fmt.Errorf("mulai ikat: %w", err)
	}
	defer tx.Rollback()

	var provider, externalID string
	var expiresAt int64
	var usedAt sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		select provider, external_id, expires_at, used_at from identity_link_codes where code = ?`, code,
	).Scan(&provider, &externalID, &expiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return WalletRow{}, ErrCodeInvalid
	}
	if err != nil {
		return WalletRow{}, fmt.Errorf("baca kode: %w", err)
	}
	if usedAt.Valid {
		return WalletRow{}, ErrCodeInvalid
	}
	if time.Now().Unix() > expiresAt {
		return WalletRow{}, ErrCodeInvalid
	}

	// Cegah pengambilalihan: identitas yang sudah milik wallet lain ditolak.
	var existingOwner string
	err = tx.QueryRowContext(ctx,
		`select wallet_user_id from identity_links where provider = ? and external_id = ?`,
		provider, externalID).Scan(&existingOwner)
	if err == nil && existingOwner != owner.WalletUserID {
		return WalletRow{}, ErrIdentityTaken
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return WalletRow{}, fmt.Errorf("cek pemilik: %w", err)
	}

	if err == nil && existingOwner == owner.WalletUserID {
		// Sudah tertaut ke wallet yang sama: cukup tandai kode terpakai.
		if _, err := tx.ExecContext(ctx, `update identity_link_codes set used_at = ? where code = ?`, nowUnix(), code); err != nil {
			return WalletRow{}, fmt.Errorf("tandai kode: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return WalletRow{}, err
		}
		return owner, nil
	}

	if _, err := tx.ExecContext(ctx, `
		insert into identity_links (id, wallet_user_id, provider, external_id, verified_at, created_at)
		values (?, ?, ?, ?, ?, ?)`,
		newID(), owner.WalletUserID, provider, externalID, nowUnix(), nowUnix()); err != nil {
		return WalletRow{}, fmt.Errorf("tautkan identitas: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `update identity_link_codes set used_at = ? where code = ?`, nowUnix(), code); err != nil {
		return WalletRow{}, fmt.Errorf("tandai kode: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return WalletRow{}, fmt.Errorf("simpan tautan: %w", err)
	}
	return owner, nil
}

// ListIdentities menampilkan channel yang tertaut ke satu wallet.
func (store *Store) ListIdentities(ctx context.Context, walletUserID string) ([]string, error) {
	rows, err := store.db.QueryContext(ctx,
		`select provider || ':' || external_id from identity_links where wallet_user_id = ? order by provider`,
		walletUserID)
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

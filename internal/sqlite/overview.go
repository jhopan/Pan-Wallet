package sqlite

import (
	"context"
	"errors"
	"fmt"

	"github.com/jhopan/PanWallet/internal/report"
)

// Overview menghitung ringkasan dashboard dari data SQLite.
func (store *Store) Overview(ctx context.Context) (report.Overview, error) {
	var overview report.Overview
	if err := store.db.QueryRowContext(ctx, `
		select (select count(*) from wallets),
		       (select count(*) from wallet_holds),
		       (select count(*) from wallet_ledger)`).
		Scan(&overview.WalletCount, &overview.HoldCount, &overview.LedgerCount); err != nil {
		return report.Overview{}, fmt.Errorf("ringkasan wallet: %w", err)
	}

	rows, err := store.db.QueryContext(ctx, `
		select w.wallet_user_id, u.public_id, w.balance, w.currency,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0)
		from wallets w join wallet_users u on u.id = w.wallet_user_id
		order by w.updated_at desc`)
	if err != nil {
		return report.Overview{}, fmt.Errorf("daftar wallet: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item report.OverviewWallet
		if err := rows.Scan(&item.WalletUserID, &item.PublicID, &item.Balance, &item.Currency, &item.HeldActive); err != nil {
			return report.Overview{}, fmt.Errorf("baca wallet: %w", err)
		}
		item.Available = item.Balance - item.HeldActive
		overview.Wallets = append(overview.Wallets, item)
	}
	if err := rows.Err(); err != nil {
		return report.Overview{}, fmt.Errorf("iterasi wallet: %w", err)
	}
	return overview, nil
}

// Setting membaca konfigurasi; kosong bila belum ada.
func (store *Store) Setting(ctx context.Context, key string) (string, error) {
	var value string
	err := store.db.QueryRowContext(ctx, `select value from app_settings where key = ?`, key).Scan(&value)
	if errors.Is(err, sqlNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("baca setting %s: %w", key, err)
	}
	return value, nil
}

// SetSetting menyimpan konfigurasi (upsert).
func (store *Store) SetSetting(ctx context.Context, key, value string) error {
	if _, err := store.db.ExecContext(ctx, `
		insert into app_settings (key, value, updated_at) values (?, ?, ?)
		on conflict (key) do update set value = excluded.value, updated_at = excluded.updated_at`,
		key, value, nowUnix()); err != nil {
		return fmt.Errorf("simpan setting %s: %w", key, err)
	}
	return nil
}

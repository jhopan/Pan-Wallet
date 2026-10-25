package database

import (
	"context"
	"fmt"

	"github.com/jhopan/PanWallet/internal/report"
)

func (store *Store) Overview(ctx context.Context) (report.Overview, error) {
	var overview report.Overview

	if err := store.pool.QueryRow(ctx,
		`select (select count(*) from wallets),
		        (select count(*) from wallet_holds),
		        (select count(*) from wallet_ledger)`,
	).Scan(&overview.WalletCount, &overview.HoldCount, &overview.LedgerCount); err != nil {
		return report.Overview{}, fmt.Errorf("ringkasan wallet: %w", err)
	}

	rows, err := store.pool.Query(ctx, `
		select w.wallet_user_id::text, u.public_id, w.balance, w.currency,
		       coalesce((select sum(h.amount) from wallet_holds h where h.wallet_id = w.id and h.status = 'active'), 0) as held
		from wallets w join wallet_users u on u.id = w.wallet_user_id
		order by w.updated_at desc`)
	if err != nil {
		return report.Overview{}, fmt.Errorf("daftar wallet: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var wallet report.OverviewWallet
		if err := rows.Scan(&wallet.WalletUserID, &wallet.PublicID, &wallet.Balance, &wallet.Currency, &wallet.HeldActive); err != nil {
			return report.Overview{}, fmt.Errorf("baca wallet: %w", err)
		}
		wallet.Available = wallet.Balance - wallet.HeldActive
		overview.Wallets = append(overview.Wallets, wallet)
	}
	if err := rows.Err(); err != nil {
		return report.Overview{}, fmt.Errorf("iterasi wallet: %w", err)
	}
	return overview, nil
}

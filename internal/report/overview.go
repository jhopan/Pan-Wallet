package report

// Overview is the read-only summary shown on the wallet dashboard.
type Overview struct {
	WalletCount int
	HoldCount   int
	LedgerCount int
	Wallets     []OverviewWallet
}

type OverviewWallet struct {
	WalletUserID string
	Balance      int64
	HeldActive   int64
	Available    int64
	Currency     string
}

package database

import _ "embed"

type Migration struct {
	Name string
	SQL  string
}

//go:embed migrations/000001_wallet_core.sql
var walletCoreSQL string

func Migrations() []Migration {
	return []Migration{{
		Name: "000001_wallet_core.sql",
		SQL:  walletCoreSQL,
	}}
}

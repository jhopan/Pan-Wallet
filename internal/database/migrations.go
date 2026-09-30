package database

import _ "embed"

type Migration struct {
	Name string
	SQL  string
}

//go:embed migrations/000001_wallet_core.sql
var walletCoreSQL string

//go:embed migrations/000002_api_security.sql
var apiSecuritySQL string

//go:embed migrations/000003_hold_transitions.sql
var holdTransitionsSQL string

func Migrations() []Migration {
	return []Migration{
		{Name: "000001_wallet_core.sql", SQL: walletCoreSQL},
		{Name: "000002_api_security.sql", SQL: apiSecuritySQL},
		{Name: "000003_hold_transitions.sql", SQL: holdTransitionsSQL},
	}
}

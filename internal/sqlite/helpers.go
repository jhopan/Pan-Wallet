package sqlite

import (
	"database/sql"
	"time"
)

// sqlNoRows adalah alias agar import database/sql tidak perlu diulang.
var sqlNoRows = sql.ErrNoRows

func nowUnix() int64 { return time.Now().Unix() }

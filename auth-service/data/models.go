package data

import (
	"database/sql"
	"time"
)

var dbTimeout = 3 * time.Second

var db *sql.DB

type Models struct {
	User User
}

func New(dbPool *sql.DB) Models {
	db = dbPool

	return Models{
		User: User{},
	}
}

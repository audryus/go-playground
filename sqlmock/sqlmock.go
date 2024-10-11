package main

import (
	"context"
	"database/sql"

	pgx "github.com/jackc/pgx/v5"
)

type PgxIface interface {
	Begin(context.Context) (pgx.Tx, error)
	Close()
}

func recordStats(db *sql.DB, userID, productID int) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	defer func() {
		switch err {
		case nil:
			err = tx.Commit()
		default:
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec("UPDATE products SET views = views + 1"); err != nil {
		return
	}
	if _, err = tx.Exec("INSERT INTO product_viewers (user_id, product_id) VALUES (?, ?)", userID, productID); err != nil {
		return
	}
	return
}

func main() {
}

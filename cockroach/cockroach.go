package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	dsn := "postgresql://root@localhost:26257/system"
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	defer func(conn *pgx.Conn) {
		err := conn.Close(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}(conn)

	if err != nil {
		log.Fatal("failed to connect database", err)
	}

	var now time.Time
	err = conn.QueryRow(ctx, "SELECT NOW()").Scan(&now)
	if err != nil {
		slog.Error("failed to execute query", err)
	}

	fmt.Println(now)
}

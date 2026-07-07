// Command migrate runs goose SQL migrations from ./migrations.
// Usage: migrate [up|down|status|redo]  (default: up). Reads YAVER_DATABASE_URL.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

func main() {
	url := os.Getenv("YAVER_DATABASE_URL")
	if url == "" {
		log.Fatal("YAVER_DATABASE_URL is required")
	}
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("dialect: %v", err)
	}
	if err := goose.RunContext(context.Background(), command, db, "migrations"); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}

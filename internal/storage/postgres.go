package storage

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrations are idempotent SQL statements executed on startup.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id         BIGINT PRIMARY KEY,
		username   TEXT NOT NULL DEFAULT '',
		city       TEXT NOT NULL DEFAULT '',
		timezone   TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS reminders (
		id        BIGSERIAL PRIMARY KEY,
		user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content   TEXT NOT NULL DEFAULT '',
		remind_at TIMESTAMPTZ,
		done      BOOLEAN NOT NULL DEFAULT false
	)`,
	`CREATE TABLE IF NOT EXISTS plans (
		id         BIGSERIAL PRIMARY KEY,
		user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content    TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
}

// Start creates a connection pool, pings the database, and runs migrations.
func Start(ctx context.Context, databaseURL string) *pgxpool.Pool {
	conn, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Run migrations
	for i, stmt := range migrations {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			log.Fatalf("Migration %d failed: %v", i, err)
		}
	}

	log.Println("Database connected and migrations applied")
	return conn
}

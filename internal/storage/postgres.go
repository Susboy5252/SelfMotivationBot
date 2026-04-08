package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrations is an ordered list of SQL statements executed on startup.
// Each statement is idempotent so running them multiple times is safe.
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
		id         BIGSERIAL PRIMARY KEY,
		creator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content    TEXT NOT NULL DEFAULT '',
		remind_at  TIMESTAMPTZ,
		done       BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS reminder_recipients (
		id          BIGSERIAL PRIMARY KEY,
		reminder_id BIGINT NOT NULL REFERENCES reminders(id) ON DELETE CASCADE,
		user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		sender_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status      TEXT NOT NULL DEFAULT 'pending',
		done        BOOLEAN NOT NULL DEFAULT false,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		UNIQUE(reminder_id, user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS plans (
		id         BIGSERIAL PRIMARY KEY,
		user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content    TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
}

// Connect creates a connection pool and verifies it with a ping.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: failed to ping database: %w", err)
	}

	return pool, nil
}

// Migrate runs all migration statements sequentially.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	for i, stmt := range migrations {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("storage: migration %d failed: %w", i, err)
		}
	}
	return nil
}

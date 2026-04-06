package handler

import (
	"context"
	"errors"
	"fmt"
	"motivator/models"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type user struct {
	db *pgxpool.Pool
}

func newUser(db *pgxpool.Pool) *user {
	return &user{
		db: db,
	}
}

// CreateUser registers a new user. Uses ON CONFLICT to avoid duplicates.
func (u *user) CreateUser(ctx context.Context, userID int64, username string) error {
	_, err := u.db.Exec(ctx,
		"INSERT INTO users (id, username) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING",
		userID, username,
	)
	if err != nil {
		return fmt.Errorf("user.CreateUser: %w", err)
	}
	return nil
}

// GetUser retrieves a user by ID. Returns nil if not found.
func (u *user) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	usr := models.User{}

	row := u.db.QueryRow(ctx,
		"SELECT id, username, city, timezone, created_at, updated_at FROM users WHERE id = $1", userID,
	)
	err := row.Scan(&usr.ID, &usr.Username, &usr.City, &usr.TimeZone, &usr.CreatedAt, &usr.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("user.GetUser: %w", err)
	}
	return &usr, nil
}

// ConfirmUser checks that all mentioned users exist in the bot.
// Returns the ID of the last confirmed user, or the problematic mention and ErrUserNotFound.
func (u *user) ConfirmUser(ctx context.Context, mentions []string) (int64, string, error) {
	var id int64

	for _, m := range mentions {
		username := strings.TrimPrefix(m, "@")

		err := u.db.QueryRow(ctx,
			"SELECT id FROM users WHERE username = $1", username,
		).Scan(&id)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, m, ErrUserNotFound
			}
			return 0, "", fmt.Errorf("user.ConfirmUser: %w", err)
		}
	}
	return id, "", nil
}

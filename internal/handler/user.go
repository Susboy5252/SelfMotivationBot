package handler

import (
	"context"
	"errors"
	"fmt"
	"motivator/models"

	"github.com/jackc/pgx/v5"
)

type user struct {
	db *pgx.Conn
}

func newUser (db *pgx.Conn) *user {
	return &user {
		db: db,
	}
}

	func (u *user) CreateUser(ctx context.Context, userID int64, username string) error {
		_, err := u.db.Exec(ctx, "insert into users (id, username) values ($1, $2)", userID, username)

		if err != nil {
			return fmt.Errorf("Error Exec id or username %w", err)
		}

return nil
	}

	func (u *user) GetUser(ctx context.Context, userID int64) (*models.User, error) {
		user := models.User{}

		row := u.db.QueryRow(ctx, "select id, username, city, timezone, created_at, updated_at from users where id=$1", userID)
		err := row.Scan(&user.ID, &user.Username, &user.City, &user.TimeZone, &user.Created_At, &user.Updated_At)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("Error row.Scan GetUser %w", err)
		}
		return &user, nil
	}

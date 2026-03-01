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

var(
	ErrUserNotFound = errors.New("user not found")
)
type user struct {
	db *pgxpool.Pool
}

func newUser (db *pgxpool.Pool) *user {
	return &user {
		db: db,
	}
}

	func (u *user) CreateUser(ctx context.Context, userID int64, username string) error {
		_, err := u.db.Exec(ctx, "insert into users (id, username) values ($1, $2)", userID, username)

		if err != nil {
			return fmt.Errorf("Error Exec id or username %w", err)
		}
		_, err = u.db.Exec(ctx, "insert into reminders (id) values ($1)", userID)

		if err != nil {
			return fmt.Errorf("Error Exec id %w", err)
		}


return nil
	}

	func (u *user) GetUser(ctx context.Context, userID int64) (*models.User, error) {
		user := models.User{}

		row := u.db.QueryRow(ctx, "select id, username, city, timezone, created_at, updated_at from users where id=$1", userID)
		err := row.Scan(&user.ID, &user.Username, &user.City, &user.TimeZone, &user.Created_At, &user.Updated_At)
		if err != nil {
			if user.ID == 0{
				return nil, nil
			}
			return nil, fmt.Errorf("Error row.Scan GetUser %w", err)
		}
		return &user, nil
	}

	func (u *user) ConfirmUser(ctx context.Context, mentions []string) (int64, string, error) {
		var usernames []string
 		var id int64

		for _, m := range mentions {
    	username := strings.TrimPrefix(m, "@")
   	  usernames = append(usernames, username)

    err := u.db.QueryRow(context.Background(),
        "SELECT id FROM users WHERE username = $1",
        username,
    ).Scan(&id)

    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return 0, m, ErrUserNotFound
        }
        return 0, "", fmt.Errorf("Error u.db.QueryRow %w", err)
    }
	}
		return id, "", nil
	}

package handler

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type city struct {
	db *pgxpool.Pool
}

func newCity(db *pgxpool.Pool) *city {
	return &city{
		db: db,
	}
}

// GetUserCity retrieves the stored city for a user.
func (c *city) GetUserCity(ctx context.Context, userID int64) (string, error) {
	var cityName string
	row := c.db.QueryRow(ctx, "SELECT city FROM users WHERE id = $1", userID)
	if err := row.Scan(&cityName); err != nil {
		return "", fmt.Errorf("city.GetUserCity: %w", err)
	}
	return cityName, nil
}

// updateUserCity updates the user's city and the updated_at timestamp.
func (c *city) updateUserCity(ctx context.Context, userID int64, cityName string) error {
	_, err := c.db.Exec(ctx,
		"UPDATE users SET city = $1, updated_at = now() WHERE id = $2",
		cityName, userID,
	)
	if err != nil {
		return fmt.Errorf("city.updateUserCity: %w", err)
	}
	return nil
}

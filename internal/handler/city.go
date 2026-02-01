package handler

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type city struct {
	db *pgx.Conn
}

func newCity (db *pgx.Conn) *city  {
	return &city  {
		db: db,
	}
}

func (c *city) GetUserCity(ctx context.Context, userID int64) (string, error) {
	var city string
	row :=c.db.QueryRow(ctx, "select city from users where id=$1", userID)
	err := row.Scan(&city)
	if err != nil{
		return "", fmt.Errorf("Error scan %w", err)
	}
	return city, nil
}

func (c *city) updateUserCity(ctx context.Context, userID int64, city string) error {
	_, err := c.db.Exec(ctx, "update users set city=$1 where id=$2", city, userID)
	if err != nil {
		return fmt.Errorf("Error city Exec %w", err)
	}
	return nil
}


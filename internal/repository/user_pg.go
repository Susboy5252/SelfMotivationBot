package repository

import (
	"context"
	"errors"
	"fmt"
	"motivator/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// userRepo is a PostgreSQL implementation of UserRepository.
type userRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new PostgreSQL-backed UserRepository.
func NewUserRepo(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, id int64, username string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO users (id, username) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING",
		id, username,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, username, city, timezone, created_at, updated_at FROM users WHERE id = $1", id,
	)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.City, &u.TimeZone, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	return &u, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, username, city, timezone, created_at, updated_at FROM users WHERE username = $1", username,
	)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.City, &u.TimeZone, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.GetByUsername: %w", err)
	}
	return &u, nil
}

func (r *userRepo) UpdateCity(ctx context.Context, id int64, city string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET city = $1, updated_at = now() WHERE id = $2",
		city, id,
	)
	if err != nil {
		return fmt.Errorf("userRepo.UpdateCity: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateTimezone(ctx context.Context, id int64, timezone string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET timezone = $1, updated_at = now() WHERE id = $2",
		timezone, id,
	)
	if err != nil {
		return fmt.Errorf("userRepo.UpdateTimezone: %w", err)
	}
	return nil
}

func (r *userRepo) UpdateLanguage(ctx context.Context, id int64, language string) error {

	_, err := r.pool.Exec(ctx, "UPDATE users set language = $1, updated_at = now() Where id = $2",
	language, id,
)
	if err != nil {
		return fmt.Errorf("userRepo.UpdateLanguage: %w", err)
	}
	return nil
}

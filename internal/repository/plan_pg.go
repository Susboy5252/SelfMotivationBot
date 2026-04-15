package repository

import (
	"context"
	"fmt"
	"motivator/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// planRepo is a PostgreSQL implementation of PlanRepository.
type planRepo struct {
	pool *pgxpool.Pool
}

// NewPlanRepo creates a new PostgreSQL-backed PlanRepository.
func NewPlanRepo(pool *pgxpool.Pool) PlanRepository {
	return &planRepo{pool: pool}
}

func (r *planRepo) Create(ctx context.Context, userID int64, content string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		"INSERT INTO plans (user_id, content) VALUES ($1, $2) RETURNING id",
		userID, content,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("planRepo.Create: %w", err)
	}
	return id, nil
}

func (r *planRepo) GetByUserID(ctx context.Context, userID int64) ([]models.Plan, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, user_id, content, created_at FROM plans WHERE user_id = $1 ORDER BY id ASC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("planRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	var result []models.Plan
	for rows.Next() {
		var p models.Plan
		if err := rows.Scan(&p.ID, &p.UserID, &p.Content, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("planRepo.GetByUserID scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

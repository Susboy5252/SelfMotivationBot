package handler

import (
	"context"
	"fmt"
	"motivator/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type plan struct {
	db *pgxpool.Pool
}

func newPlan(db *pgxpool.Pool) *plan {
	return &plan{db: db}
}

// savePlan persists a plan entry in the database.
func (p *plan) savePlan(ctx context.Context, userID int64, content string) (int64, error) {
	var id int64
	err := p.db.QueryRow(ctx,
		"INSERT INTO plans (user_id, content) VALUES ($1, $2) RETURNING id",
		userID, content,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("plan.savePlan: %w", err)
	}
	return id, nil
}

// getPlans retrieves all plans for a user.
func (p *plan) getPlans(ctx context.Context, userID int64) ([]models.Plan, error) {
	rows, err := p.db.Query(ctx,
		"SELECT id, user_id, content, created_at FROM plans WHERE user_id = $1 ORDER BY id ASC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("plan.getPlans: %w", err)
	}
	defer rows.Close()

	var result []models.Plan
	for rows.Next() {
		var pl models.Plan
		if err := rows.Scan(&pl.ID, &pl.UserID, &pl.Content, &pl.CreatedAt); err != nil {
			return nil, fmt.Errorf("plan.getPlans scan: %w", err)
		}
		result = append(result, pl)
	}
	return result, rows.Err()
}

// deletePlan removes a plan by ID and user ID.
func (p *plan) deletePlan(ctx context.Context, planID int64, userID int64) error {
	_, err := p.db.Exec(ctx,
		"DELETE FROM plans WHERE id = $1 AND user_id = $2",
		planID, userID,
	)
	if err != nil {
		return fmt.Errorf("plan.deletePlan: %w", err)
	}
	return nil
}

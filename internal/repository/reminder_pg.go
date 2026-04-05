package repository

import (
	"context"
	"errors"
	"fmt"
	"motivator/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// reminderRepo is a PostgreSQL implementation of ReminderRepository.
type reminderRepo struct {
	pool *pgxpool.Pool
}

// NewReminderRepo creates a new PostgreSQL-backed ReminderRepository.
func NewReminderRepo(pool *pgxpool.Pool) ReminderRepository {
	return &reminderRepo{pool: pool}
}

func (r *reminderRepo) Create(ctx context.Context, rem *models.Reminder) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		"INSERT INTO reminders (user_id, content, remind_at) VALUES ($1, $2, $3) RETURNING id",
		rem.UserID, rem.Content, rem.RemindAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("reminderRepo.Create: %w", err)
	}
	return id, nil
}

func (r *reminderRepo) GetByUserID(ctx context.Context, userID int64) ([]models.Reminder, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, user_id, content, remind_at, done FROM reminders WHERE user_id = $1 ORDER BY id ASC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("reminderRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	var result []models.Reminder
	for rows.Next() {
		var rem models.Reminder
		if err := rows.Scan(&rem.ID, &rem.UserID, &rem.Content, &rem.RemindAt, &rem.Done); err != nil {
			return nil, fmt.Errorf("reminderRepo.GetByUserID scan: %w", err)
		}
		result = append(result, rem)
	}
	return result, rows.Err()
}

func (r *reminderRepo) GetByID(ctx context.Context, id int64) (*models.Reminder, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, user_id, content, remind_at, done FROM reminders WHERE id = $1", id,
	)

	var rem models.Reminder
	err := row.Scan(&rem.ID, &rem.UserID, &rem.Content, &rem.RemindAt, &rem.Done)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("reminderRepo.GetByID: %w", err)
	}
	return &rem, nil
}

func (r *reminderRepo) UpdateRemindAt(ctx context.Context, id int64, remindAt interface{}) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE reminders SET remind_at = $1 WHERE id = $2",
		remindAt, id,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.UpdateRemindAt: %w", err)
	}
	return nil
}

func (r *reminderRepo) ToggleDone(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE reminders SET done = NOT done WHERE id = $1", id,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.ToggleDone: %w", err)
	}
	return nil
}

func (r *reminderRepo) Delete(ctx context.Context, id int64, userID int64) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM reminders WHERE id = $1 AND user_id = $2", id, userID,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.Delete: %w", err)
	}
	return nil
}

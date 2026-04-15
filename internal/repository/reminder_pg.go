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
		"INSERT INTO reminders (creator_id, content, remind_at) VALUES ($1, $2, $3) RETURNING id",
		rem.CreatorID, rem.Content, rem.RemindAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("reminderRepo.Create: %w", err)
	}
	return id, nil
}

// GetByCreatorID returns reminders created by a specific user.
func (r *reminderRepo) GetByCreatorID(ctx context.Context, creatorID int64) ([]models.Reminder, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, creator_id, content, remind_at, done FROM reminders WHERE creator_id = $1 ORDER BY id ASC",
		creatorID,
	)
	if err != nil {
		return nil, fmt.Errorf("reminderRepo.GetByCreatorID: %w", err)
	}
	defer rows.Close()

	var result []models.Reminder
	for rows.Next() {
		var rem models.Reminder
		if err := rows.Scan(&rem.ID, &rem.CreatorID, &rem.Content, &rem.RemindAt, &rem.Done); err != nil {
			return nil, fmt.Errorf("reminderRepo.GetByCreatorID scan: %w", err)
		}
		result = append(result, rem)
	}
	return result, rows.Err()
}

// GetAllForUser returns reminders the user created + accepted reminders from friends.
// For friend reminders, it fills SenderUsername from the sender's username.
// Excludes reminders where the creator only sent them to friends (has recipients but
// creator is not among them).
func (r *reminderRepo) GetAllForUser(ctx context.Context, userID int64, timeZone string) ([]models.Reminder, error) {
	query := `
			SELECT r.id, r.creator_id, r.content, 
       r.remind_at AT TIME ZONE $2 AS remind_at, 
       r.done, '' AS sender_username
FROM reminders r
WHERE r.creator_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM reminder_recipients rr WHERE rr.reminder_id = r.id AND rr.user_id = $1
  )
  AND NOT EXISTS (
    SELECT 1 FROM reminder_recipients rr2 WHERE rr2.reminder_id = r.id
  )
UNION ALL
SELECT r.id, r.creator_id, r.content, 
       r.remind_at AT TIME ZONE $2 AS remind_at, 
       rr.done, u.username AS sender_username
FROM reminders r
JOIN reminder_recipients rr ON rr.reminder_id = r.id
JOIN users u ON u.id = rr.sender_id
WHERE rr.user_id = $1 AND rr.status = 'accepted'
ORDER BY id ASC;
	`
	rows, err := r.pool.Query(ctx, query, userID, timeZone)
	if err != nil {
		return nil, fmt.Errorf("reminderRepo.GetAllForUser: %w", err)
	}
	defer rows.Close()

	var result []models.Reminder
	for rows.Next() {
		var rem models.Reminder
		if err := rows.Scan(&rem.ID, &rem.CreatorID, &rem.Content, &rem.RemindAt, &rem.Done, &rem.SenderUsername); err != nil {
			return nil, fmt.Errorf("reminderRepo.GetAllForUser scan: %w", err)
		}
		result = append(result, rem)
	}
	return result, rows.Err()
}

func (r *reminderRepo) GetByID(ctx context.Context, id int64) (*models.Reminder, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT id, creator_id, content, remind_at, done FROM reminders WHERE id = $1", id,
	)

	var rem models.Reminder
	err := row.Scan(&rem.ID, &rem.CreatorID, &rem.Content, &rem.RemindAt, &rem.Done)
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
		"UPDATE reminders SET remind_at = $1, updated_at = now() WHERE id = $2",
		remindAt, id,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.UpdateRemindAt: %w", err)
	}
	return nil
}

func (r *reminderRepo) ToggleDone(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE reminders SET done = NOT done, updated_at = now() WHERE id = $1", id,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.ToggleDone: %w", err)
	}
	return nil
}

func (r *reminderRepo) Delete(ctx context.Context, id int64, userID int64) error {
	// Delete if user is the creator OR if they are a recipient
	_, err := r.pool.Exec(ctx, `
		DELETE FROM reminders WHERE id = $1 AND (
			creator_id = $2
			OR EXISTS (SELECT 1 FROM reminder_recipients WHERE reminder_id = $1 AND user_id = $2)
		)
	`, id, userID)
	if err != nil {
		return fmt.Errorf("reminderRepo.Delete: %w", err)
	}
	return nil
}

// AddRecipient creates a new reminder_recipients row.
func (r *reminderRepo) AddRecipient(ctx context.Context, rec *models.ReminderRecipient) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO reminder_recipients (reminder_id, user_id, sender_id, status) VALUES ($1, $2, $3, $4) ON CONFLICT (reminder_id, user_id) DO NOTHING",
		rec.ReminderID, rec.UserID, rec.SenderID, rec.Status,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.AddRecipient: %w", err)
	}
	return nil
}

// UpdateRecipientStatus updates the status of a recipient (pending -> accepted/declined).
func (r *reminderRepo) UpdateRecipientStatus(ctx context.Context, reminderID, userID int64, status string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE reminder_recipients SET status = $1 WHERE reminder_id = $2 AND user_id = $3",
		status, reminderID, userID,
	)
	if err != nil {
		return fmt.Errorf("reminderRepo.UpdateRecipientStatus: %w", err)
	}
	return nil
}

// GetPendingForUser returns all pending recipient records for a user.
func (r *reminderRepo) GetPendingForUser(ctx context.Context, userID int64) ([]models.ReminderRecipient, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, reminder_id, user_id, sender_id, status, done, created_at FROM reminder_recipients WHERE user_id = $1 AND status = 'pending'",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("reminderRepo.GetPendingForUser: %w", err)
	}
	defer rows.Close()

	var result []models.ReminderRecipient
	for rows.Next() {
		var rec models.ReminderRecipient
		if err := rows.Scan(&rec.ID, &rec.ReminderID, &rec.UserID, &rec.SenderID, &rec.Status, &rec.Done, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("reminderRepo.GetPendingForUser scan: %w", err)
		}
		result = append(result, rec)
	}
	return result, rows.Err()
}

package repository

import (
	"context"
	"motivator/internal/models"
)

// UserRepository defines all data-access operations for users.
type UserRepository interface {
	Create(ctx context.Context, id int64, username string) error
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	UpdateCity(ctx context.Context, id int64, city string) error
	UpdateLanguage(ctx context.Context, id int64, language string) error
	UpdateTimezone(ctx context.Context, id int64, timezone string) error
	UpdateUsername(ctx context.Context, id int64, username string) error
}

// ReminderRepository defines all data-access operations for reminders.
type ReminderRepository interface {
	Create(ctx context.Context, r *models.Reminder) (int64, error)
	GetByCreatorID(ctx context.Context, creatorID int64) ([]models.Reminder, error)
	GetAllForUser(ctx context.Context, userID int64, timeZone string) ([]models.Reminder, error)
	GetByID(ctx context.Context, id int64) (*models.Reminder, error)
	UpdateRemindAt(ctx context.Context, id int64, remindAt interface{}) error
	ToggleDone(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64, userID int64) error

	// Recipient operations
	AddRecipient(ctx context.Context, rec *models.ReminderRecipient) error
	UpdateRecipientStatus(ctx context.Context, reminderID, userID int64, status string) error
	GetPendingForUser(ctx context.Context, userID int64) ([]models.ReminderRecipient, error)
}

// PlanRepository defines all data-access operations for plans.
type PlanRepository interface {
	Create(ctx context.Context, userID int64, content string) (int64, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.Plan, error)
}

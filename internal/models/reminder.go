package models

import "time"

// Reminder represents a single reminder or note created by a user.
type Reminder struct {
	ID        int64
	CreatorID int64      // who created this reminder (was UserID)
	Content   string
	RemindAt  *time.Time // nil means it's a note, not a timed reminder
	Done      bool
	CreatedAt time.Time
	UpdatedAt time.Time

	// Display-only fields (not stored in reminders table directly)
	SenderUsername string // filled when this reminder came from a friend
}

// IsNote returns true when the reminder has no scheduled time.
func (r *Reminder) IsNote() bool {
	return r.RemindAt == nil || r.RemindAt.IsZero()
}

// ReminderRecipient represents a shared reminder sent to another user.
type ReminderRecipient struct {
	ID         int64
	ReminderID int64
	UserID     int64  // recipient
	SenderID   int64  // who sent it
	Status     string // "pending", "accepted", "declined"
	Done       bool
	CreatedAt  time.Time
}

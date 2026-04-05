package models

import "time"

// Reminder represents a single reminder or note belonging to a user.
type Reminder struct {
	ID       int64
	UserID   int64
	Content  string
	RemindAt *time.Time // nil means it's a note, not a timed reminder
	Done     bool
}

// IsNote returns true when the reminder has no scheduled time.
func (r *Reminder) IsNote() bool {
	return r.RemindAt == nil || r.RemindAt.IsZero()
}

package models

import (
	"time"
)

// User represents a registered bot user.
type User struct {
	ID        int64
	Username  string
	City      string
	TimeZone  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Reminders represents a single reminder or note.
type Reminders struct {
	SerialID int64
	LocalID  int
	Content  string
	Time     *time.Time
	Done     bool
}

// Plan represents a user's plan entry stored in the database.
type Plan struct {
	ID        int64
	UserID    int64
	Content   string
	CreatedAt time.Time
}

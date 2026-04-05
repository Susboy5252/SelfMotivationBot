package models

import "time"

// Plan represents a user's plan entry.
type Plan struct {
	ID        int64
	UserID    int64
	Content   string
	CreatedAt time.Time
}

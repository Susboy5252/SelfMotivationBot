package models

import "time"

// User represents a registered bot user.
type User struct {
	ID        int64
	Username  string
	City      string
	TimeZone  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Language string
}

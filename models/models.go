package models

import (
	"time"
)

type User struct {
	ID         int64
	Username   string
	City       string
	TimeZone   string
	Created_At time.Time
	Updated_At time.Time
}

type Reminders struct {
	Serial_id int64
	Local_id  int
	Content   string
	Time *time.Time
	Done bool
}

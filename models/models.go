package models

import "time"

type User struct {
	ID         int64
	Username   string
	City       string
	TimeZone   string
	Created_At time.Time
	Updated_At time.Time
}

var (
	Map_id = make(map[int]int64)
)

type Reminders struct {
	Serial_id int64
	Local_id  int
	Content   string
}

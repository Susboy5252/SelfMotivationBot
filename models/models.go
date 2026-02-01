package models

import "time"

type User struct {
	ID int64
	Username string
	City string
	TimeZone string
	Created_At time.Time
	Updated_At time.Time
}

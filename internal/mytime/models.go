package mytime

import (
	"time"
)

type GeonamesResponse struct {
	PostalCodes []struct {
		Lng       float64 `json:"lng"`
		Lat       float64 `json:"lat"`
		PlaceName string  `json:"placeName"`
	} `json:"postalCodes"`
}

type Coordinates struct {
	Lng float64
	Lat float64
}

type TimeRequest struct {
	TimezoneID string `json:"timezoneId"`
	TimeNow    string `json:"time"`
}

/*type Time struct {
		TimezoneID  string  `json:"timezoneId"`
		TimeNow     string  `json:"time"`

}*/

var Weekdays = map[time.Weekday]string{
	time.Sunday:    "Воскресенье",
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
}

var Months = map[time.Month]string{
	time.January:   "января",
	time.February:  "февраля",
	time.March:     "марта",
	time.April:     "апреля",
	time.May:       "мая",
	time.June:      "июня",
	time.July:      "июля",
	time.August:    "августа",
	time.September: "сентября",
	time.October:   "октября",
	time.November:  "ноября",
	time.December:  "декабря",
}

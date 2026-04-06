package mytime

import (
	"time"
)

// GeonamesResponse is the API response from GeoNames postalCodeLookup.
type GeonamesResponse struct {
	PostalCodes []struct {
		Lng       float64 `json:"lng"`
		Lat       float64 `json:"lat"`
		PlaceName string  `json:"placeName"`
	} `json:"postalCodes"`
}

// Coordinates holds latitude and longitude.
type Coordinates struct {
	Lng float64
	Lat float64
}

// TimeRequest is the API response from GeoNames timezone endpoint.
type TimeRequest struct {
	TimezoneID string `json:"timezoneId"`
	TimeNow    string `json:"time"`
}

// Weekdays maps Go's time.Weekday to Russian day names.
var Weekdays = map[time.Weekday]string{
	time.Sunday:    "Воскресенье",
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
}

// Months maps Go's time.Month to Russian month names (genitive case).
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

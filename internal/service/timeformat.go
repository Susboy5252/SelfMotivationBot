package service

import (
	"fmt"
	"time"
)

// Russian weekday names.
var weekdays = map[time.Weekday]string{
	time.Sunday:    "Воскресенье",
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
}

// Russian month names (genitive case).
var months = map[time.Month]string{
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

// FormatDateTimeRu formats a time.Time into a human-readable Russian string.
// Example: "Понедельник, 5 мая 2025 14:30"
func FormatDateTimeRu(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d %02d:%02d",
		weekdays[t.Weekday()],
		t.Day(),
		months[t.Month()],
		t.Year(),
		t.Hour(),
		t.Minute(),
	)
}

func FormatDateTimeEn(t time.Time) string {
	return time.Now().Format("Monday 02 January 2006 15:04")
}

// GetLocalTime returns the current local time string for a timezone.
func GetLocalTime(timezone string) (string, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("timeformat: invalid timezone %q: %w", timezone, err)
	}
	return time.Now().In(loc).Format("15:04"), nil
}

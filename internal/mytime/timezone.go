package mytime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TimeZone provides timezone detection and storage.
type TimeZone struct {
	db          *pgxpool.Pool
	geoUsername string
}

// New creates a TimeZone service with the given GeoNames API username.
func New(db *pgxpool.Pool, geoUsername string) *TimeZone {
	return &TimeZone{
		db:          db,
		geoUsername: geoUsername,
	}
}

// GetCoordinates resolves a city name to geographic coordinates using GeoNames API.
func (t *TimeZone) GetCoordinates(city string) (Coordinates, error) {
	baseURL := "http://api.geonames.org/postalCodeLookupJSON?placename=%s&username=%s"
	cityEscaped := url.QueryEscape(city)

	resp, err := http.Get(fmt.Sprintf(baseURL, cityEscaped, url.QueryEscape(t.geoUsername)))
	if err != nil {
		return Coordinates{}, fmt.Errorf("mytime.GetCoordinates: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Coordinates{}, fmt.Errorf("mytime.GetCoordinates: API returned status %d", resp.StatusCode)
	}

	var coordinatesResponse GeonamesResponse
	if err := json.NewDecoder(resp.Body).Decode(&coordinatesResponse); err != nil {
		return Coordinates{}, fmt.Errorf("mytime.GetCoordinates: failed to decode response: %w", err)
	}

	if len(coordinatesResponse.PostalCodes) == 0 {
		return Coordinates{}, fmt.Errorf("mytime.GetCoordinates: no results for city %q", city)
	}

	result := coordinatesResponse.PostalCodes[0]
	return Coordinates{
		Lng: result.Lng,
		Lat: result.Lat,
	}, nil
}

// GetTimeZone resolves coordinates to a timezone and saves it for the user.
// Returns (timezone string, error) following Go conventions.
func (t *TimeZone) GetTimeZone(ctx context.Context, userID int64, lng float64, lat float64) (string, error) {
	apiURL := "http://api.geonames.org/timezoneJSON?lat=%f&lng=%f&username=%s"

	resp, err := http.Get(fmt.Sprintf(apiURL, lat, lng, url.QueryEscape(t.geoUsername)))
	if err != nil {
		return "", fmt.Errorf("mytime.GetTimeZone: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mytime.GetTimeZone: API returned status %d", resp.StatusCode)
	}

	var timeResponse TimeRequest
	if err := json.NewDecoder(resp.Body).Decode(&timeResponse); err != nil {
		return "", fmt.Errorf("mytime.GetTimeZone: failed to decode response: %w", err)
	}

	if timeResponse.TimezoneID == "" {
		return "", fmt.Errorf("mytime.GetTimeZone: empty timezone for coordinates (%f, %f)", lat, lng)
	}

	_, err = t.db.Exec(ctx, "UPDATE users SET timezone = $1, updated_at = now() WHERE id = $2", timeResponse.TimezoneID, userID)
	if err != nil {
		return "", fmt.Errorf("mytime.GetTimeZone: failed to update user timezone: %w", err)
	}

	return timeResponse.TimezoneID, nil
}

// GetUserTimeZone retrieves the stored timezone for a user.
func (t *TimeZone) GetUserTimeZone(ctx context.Context, userID int64) (string, error) {
	var timeZone string
	row := t.db.QueryRow(ctx, "SELECT timezone FROM users WHERE id = $1", userID)

	if err := row.Scan(&timeZone); err != nil {
		return "", fmt.Errorf("mytime.GetUserTimeZone: %w", err)
	}
	return timeZone, nil
}

// GetUserLocalTimeZone returns the current local time string for a timezone.
func GetUserLocalTimeZone(timeZone string) (string, error) {
	location, err := time.LoadLocation(timeZone)
	if err != nil {
		return "", fmt.Errorf("mytime.GetUserLocalTimeZone: invalid timezone %q: %w", timeZone, err)
	}
	localTime := time.Now().In(location)
	return localTime.Format("15:04"), nil
}

// FormatDateTime formats a time.Time into a human-readable Russian string.
func FormatDateTime(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d %02d:%02d",
		Weekdays[t.Weekday()],
		t.Day(),
		Months[t.Month()],
		t.Year(),
		t.Hour(),
		t.Minute(),
	)
}

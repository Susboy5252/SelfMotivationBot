package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Coordinates holds latitude and longitude.
type Coordinates struct {
	Lat float64
	Lng float64
}

// geonamesPostalResponse is the API response from GeoNames postalCodeLookup.
type geonamesPostalResponse struct {
	PostalCodes []struct {
		Lng float64 `json:"lng"`
		Lat float64 `json:"lat"`
	} `json:"postalCodes"`
}

// geonamesTimezoneResponse is the API response from GeoNames timezone.
type geonamesTimezoneResponse struct {
	TimezoneID string `json:"timezoneId"`
}

// GeoService provides geocoding and timezone lookup.
type GeoService struct {
	username string
}

// NewGeoService creates a GeoService with the given GeoNames API username.
func NewGeoService(username string) *GeoService {
	return &GeoService{username: username}
}

// GetCoordinates resolves a city name to geographic coordinates.
func (g *GeoService) GetCoordinates(city string) (Coordinates, error) {
	apiURL := fmt.Sprintf(
		"http://api.geonames.org/postalCodeLookupJSON?placename=%s&username=%s",
		url.QueryEscape(city), url.QueryEscape(g.username),
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return Coordinates{}, fmt.Errorf("geo: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Coordinates{}, fmt.Errorf("geo: API returned status %d", resp.StatusCode)
	}

	var data geonamesPostalResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return Coordinates{}, fmt.Errorf("geo: failed to decode response: %w", err)
	}

	if len(data.PostalCodes) == 0 {
		return Coordinates{}, fmt.Errorf("geo: no results for city %q", city)
	}

	return Coordinates{
		Lat: data.PostalCodes[0].Lat,
		Lng: data.PostalCodes[0].Lng,
	}, nil
}

// GetTimezone resolves coordinates to an IANA timezone identifier.
func (g *GeoService) GetTimezone(lat, lng float64) (string, error) {
	apiURL := fmt.Sprintf(
		"http://api.geonames.org/timezoneJSON?lat=%f&lng=%f&username=%s",
		lat, lng, url.QueryEscape(g.username),
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("geo: timezone request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("geo: timezone API returned status %d", resp.StatusCode)
	}

	var data geonamesTimezoneResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("geo: failed to decode timezone response: %w", err)
	}

	if data.TimezoneID == "" {
		return "", fmt.Errorf("geo: empty timezone for coordinates (%f, %f)", lat, lng)
	}

	return data.TimezoneID, nil
}

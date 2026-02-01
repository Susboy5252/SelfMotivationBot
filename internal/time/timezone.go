package time

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

type TimeZone struct {
	db *pgx.Conn
}

func New(db *pgx.Conn) *TimeZone {
	return &TimeZone{
		db :db,
	}
}
func GetCoordinates(city string) (Coordinates, error) {
	baseURL := "http://api.geonames.org/postalCodeLookupJSON?placename=%s&username=%s"
	cityEscaped := url.QueryEscape(city)
	resp, err := http.Get(fmt.Sprintf(baseURL, cityEscaped, "kunt"))
	if err != nil {
	  return Coordinates{}, fmt.Errorf("Error get cooridtantes %w", err)
	}

	if resp.StatusCode != 200 {
		return Coordinates{}, fmt.Errorf("Error connect to API, %d", resp.StatusCode)
	}
		var coordinatesResponse GeonamesResponse
		err = json.NewDecoder(resp.Body).Decode(&coordinatesResponse)
		if err != nil {
			return Coordinates{}, fmt.Errorf("Error unmarshal response, %w", err)
		}
			if len(coordinatesResponse.PostalCodes) == 0 {
				return Coordinates{}, fmt.Errorf("Error empty coorditanes")
	}

	 result := coordinatesResponse.PostalCodes[0]
	return Coordinates {
		Lng: result.Lng,
		Lat: result.Lat,
	}, nil
}

func (t *TimeZone) GetTimeZone(ctx context.Context, userID int64, lng float64, lat float64) (error, string) {
	
	url := "http://api.geonames.org/timezoneJSON?lat=%f&lng=%f&username=%s"
	resp, err := http.Get(fmt.Sprintf(url, lat, lng, "kunt"))

  if err != nil {
		return fmt.Errorf("Error get time"), "Error"
	}

		if resp.StatusCode != 200 {
		return fmt.Errorf("Error connect to API, %d", resp.StatusCode), ""
	}
	var timeResponse TimeRequest
	err = json.NewDecoder(resp.Body).Decode(&timeResponse)

			if err != nil {
			return fmt.Errorf("Error unmarshal response, %w", err), ""
		}
			if len(timeResponse.TimezoneID) == 0 {
				return fmt.Errorf("Error empty TimezoneID"), (fmt.Sprintf(url, lng, lat, "kunt"))
	}
	
	_, err = t.db.Exec(ctx, "update users set timezone=$1 where id=$2",timeResponse.TimezoneID, userID)
	if err != nil {
		return fmt.Errorf("Error Exec TimeZone %w", err), ""
	}

	return nil, timeResponse.TimezoneID

}

func (t *TimeZone) GetUserTimeZone(ctx context.Context, userID int64) (string, error) {
	var timeZone string
	row := t.db.QueryRow(ctx, "select timezone from users where id=$1", userID)

	err := row.Scan(&timeZone)

if err != nil {
	return "", fmt.Errorf("Error QueryRow GetUserTimeZone")
	}
	return timeZone, nil
}

func GetUserLocalTimeZone(timeZone string) (string, error) {

	location, err := time.LoadLocation(timeZone)
	if err != nil {
	 return "",	fmt.Errorf("Error time.LoadLocation GetUserLocalTimeZone %w", err)
	}
  localTime := time.Now().In(location)

  return	localTime.Format("15:04"), nil
}

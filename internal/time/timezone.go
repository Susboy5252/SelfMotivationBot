package time

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

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

func GetTimeZone(lng float64, lat float64) (error, string) {
	
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
	return nil, timeResponse.TimezoneID

}

package time

type GeonamesResponse struct {
    PostalCodes []struct {
        Lng float64 `json:"lng"`
        Lat float64 `json:"lat"`
        PlaceName string `json:"placeName"`
    } `json:"postalCodes"`
}

type Coordinates struct {
		Lng         float64
		Lat         float64

}

type TimeRequest struct {
		TimezoneID  string  `json:"timezoneId"`
		TimeNow     string  `json:"time"`
}

/*type Time struct {
		TimezoneID  string  `json:"timezoneId"`
		TimeNow     string  `json:"time"`

}*/

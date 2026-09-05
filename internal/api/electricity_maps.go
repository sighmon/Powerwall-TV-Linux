package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ElectricityMapsData struct {
	CarbonIntensity      int     `json:"carbonIntensity"`
	FossilFuelPercentage float64 `json:"fossilFuelPercentage"`
}

type electricityMapsResponse struct {
	Data ElectricityMapsData `json:"data"`
}

func FetchElectricityMaps(apiKey, zone string) (*ElectricityMapsData, error) {
	endpoint := "https://api.electricitymaps.com/v3/home-assistant?zone=" + url.QueryEscape(zone)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("auth-token", apiKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Electricity Maps failed: %s", resp.Status)
	}
	var result electricityMapsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

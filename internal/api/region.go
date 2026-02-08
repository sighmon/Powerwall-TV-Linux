package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type regionResp struct {
	Response struct {
		FleetBaseURL string `json:"fleet_api_base_url"`
	} `json:"response"`
}

func ResolveFleetBaseURL(token string) (string, error) {
	candidates := []string{
		"https://fleet-api.prd.na.vn.cloud.tesla.com",
		"https://fleet-api.prd.eu.vn.cloud.tesla.com",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	for _, base := range candidates {
		url := fmt.Sprintf("%s/api/1/users/region", base)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}
		var r regionResp
		err = json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		if err != nil {
			continue
		}
		if r.Response.FleetBaseURL != "" {
			return r.Response.FleetBaseURL, nil
		}
	}
	return "", fmt.Errorf("failed to resolve fleet region")
}

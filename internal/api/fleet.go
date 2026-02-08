package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type FleetClient struct {
	BaseURL    string
	Token      string
	client     *http.Client
}

func NewFleetClient(baseURL, token string) *FleetClient {
	return &FleetClient{
		BaseURL: baseURL,
		Token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type ProductsResponse struct {
	Response []Product `json:"response"`
}

type Product struct {
	EnergySiteID int    `json:"energy_site_id"`
	SiteName     string `json:"site_name"`
}

type FleetLiveStatusResp struct {
	Response FleetLiveStatus `json:"response"`
}

type FleetLiveStatus struct {
	BatteryPower     float64 `json:"battery_power"`
	BatteryPercent   float64 `json:"percentage_charged"`
	SolarPower       float64 `json:"solar_power"`
	LoadPower        float64 `json:"load_power"`
	GridPower        float64 `json:"grid_power"`
	GridStatus       string  `json:"grid_status"`
}

type FleetSnapshot struct {
	SiteName string
	Status   FleetLiveStatus
}

func (c *FleetClient) FetchSnapshot(siteIdx int) (*FleetSnapshot, int, error) {
	products, err := c.products()
	if err != nil {
		return nil, siteIdx, err
	}
	if len(products) == 0 {
		return nil, siteIdx, fmt.Errorf("no energy sites found")
	}
	if siteIdx < 0 || siteIdx >= len(products) {
		siteIdx = 0
	}
	site := products[siteIdx]

	live, err := c.liveStatus(site.EnergySiteID)
	if err != nil {
		return nil, siteIdx, err
	}

	return &FleetSnapshot{SiteName: site.SiteName, Status: live}, siteIdx, nil
}

func (c *FleetClient) products() ([]Product, error) {
	url := fmt.Sprintf("%s/api/1/products", c.BaseURL)
	var resp ProductsResponse
	if err := c.getJSON(url, &resp); err != nil {
		return nil, err
	}
	return resp.Response, nil
}

func (c *FleetClient) liveStatus(siteID int) (FleetLiveStatus, error) {
	url := fmt.Sprintf("%s/api/1/energy_sites/%d/live_status", c.BaseURL, siteID)
	var resp FleetLiveStatusResp
	if err := c.getJSON(url, &resp); err != nil {
		return FleetLiveStatus{}, err
	}
	return resp.Response, nil
}

func (c *FleetClient) getJSON(url string, v any) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s failed: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

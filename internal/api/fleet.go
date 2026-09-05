package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type FleetClient struct {
	BaseURL string
	Token   string
	client  *http.Client
}

type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPStatusError) Error() string { return "fleet request failed: " + e.Status }

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
	DeviceType      string `json:"device_type"`
	EnergySiteID    int    `json:"energy_site_id"`
	SiteName        string `json:"site_name"`
	Name            string `json:"name"`
	Title           string `json:"title"`
	DisplayName     string `json:"display_name"`
	SiteDisplayName string `json:"site_display_name"`
	VIN             string `json:"vin"`
	State           string `json:"state"`
}

type FleetLiveStatusResp struct {
	Response FleetLiveStatus `json:"response"`
}

type FleetLiveStatus struct {
	BatteryPower    float64         `json:"battery_power"`
	BatteryPercent  float64         `json:"percentage_charged"`
	SolarPower      float64         `json:"solar_power"`
	LoadPower       float64         `json:"load_power"`
	GridPower       float64         `json:"grid_power"`
	GridStatus      string          `json:"grid_status"`
	IslandStatus    string          `json:"island_status"`
	StormModeActive bool            `json:"storm_mode_active"`
	WallConnectors  []WallConnector `json:"wall_connectors"`
}

type WallConnector struct {
	VIN                string   `json:"vin"`
	DIN                string   `json:"din"`
	WallConnectorState *float64 `json:"wall_connector_state"`
	WallConnectorPower *float64 `json:"wall_connector_power"`
}

type FleetSnapshot struct {
	SiteID    int
	SiteName  string
	SiteCount int
	Status    FleetLiveStatus
	Sites     []Product
	Vehicles  []FleetVehicle
}

type SiteInfo struct {
	Response struct {
		SiteName             string  `json:"site_name"`
		DisplayName          string  `json:"display_name"`
		SiteDisplayName      string  `json:"site_display_name"`
		Version              string  `json:"version"`
		BatteryCount         float64 `json:"battery_count"`
		BackupReservePercent float64 `json:"backup_reserve_percent"`
		InstallationDate     string  `json:"installation_date"`
	} `json:"response"`
}

func (c *FleetClient) FetchSiteInfo(siteID int) (*SiteInfo, error) {
	endpoint := fmt.Sprintf("%s/api/1/energy_sites/%d/site_info", c.BaseURL, siteID)
	var info SiteInfo
	if err := c.getJSON(endpoint, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *FleetClient) FetchSnapshot(siteIdx int) (*FleetSnapshot, int, error) {
	products, err := c.products()
	if err != nil {
		return nil, siteIdx, err
	}
	energySites := make([]Product, 0, len(products))
	vehicles := make([]FleetVehicle, 0)
	for _, product := range products {
		if product.EnergySiteID != 0 {
			energySites = append(energySites, product)
		}
		if product.DeviceType == "vehicle" && product.VIN != "" {
			vehicles = append(vehicles, FleetVehicle{VIN: product.VIN, DisplayName: product.DisplayName, State: product.State})
		}
	}
	if len(energySites) == 0 {
		return nil, siteIdx, fmt.Errorf("no energy sites found")
	}
	if siteIdx < 0 {
		siteIdx = 0
	}
	if siteIdx >= len(energySites) {
		siteIdx = len(energySites) - 1
	}
	site := energySites[siteIdx]

	live, err := c.liveStatus(site.EnergySiteID)
	if err != nil {
		return nil, siteIdx, err
	}

	return &FleetSnapshot{SiteID: site.EnergySiteID, SiteName: site.DisplayLabel(), SiteCount: len(energySites), Status: live, Sites: energySites, Vehicles: vehicles}, siteIdx, nil
}

func (p Product) DisplayLabel() string {
	for _, name := range []string{p.DisplayName, p.SiteDisplayName, p.Name, p.Title, p.SiteName} {
		if name != "" {
			return name
		}
	}
	return ""
}

type FleetVehicle struct {
	VIN         string `json:"vin"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
}

type vehiclesResponse struct {
	Response []FleetVehicle `json:"response"`
}

type VehicleChargeState struct {
	BatteryLevel        *float64 `json:"battery_level"`
	ChargingState       string   `json:"charging_state"`
	MinutesToFullCharge *float64 `json:"minutes_to_full_charge"`
}

type vehicleDataResponse struct {
	Response struct {
		ChargeState *VehicleChargeState `json:"charge_state"`
	} `json:"response"`
}

func (c *FleetClient) FetchVehicles() ([]FleetVehicle, error) {
	var response vehiclesResponse
	if err := c.getJSON(c.BaseURL+"/api/1/vehicles", &response); err != nil {
		return nil, err
	}
	return response.Response, nil
}

func (c *FleetClient) FetchVehicleData(vin string) (*VehicleChargeState, error) {
	var response vehicleDataResponse
	endpoint := fmt.Sprintf("%s/api/1/vehicles/%s/vehicle_data", c.BaseURL, url.PathEscape(vin))
	if err := c.getJSON(endpoint, &response); err != nil {
		return nil, err
	}
	if response.Response.ChargeState == nil {
		return nil, fmt.Errorf("vehicle charge state unavailable")
	}
	return response.Response.ChargeState, nil
}

func (c *FleetClient) SetOperationMode(siteID int, mode string) error {
	body, err := json.Marshal(map[string]string{"default_real_mode": mode})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/1/energy_sites/%d/operation", c.BaseURL, siteID)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status, Body: responseBody}
	}
	return nil
}

type CalendarHistory struct {
	Battery []HistorySample
	Solar   []HistorySample
	Home    []HistorySample
	Grid    []HistorySample
	SOE     []HistorySample
	Totals  HistoryTotals
}

type HistorySample struct {
	Time  time.Time
	Value float64
	Flow  string
}

type HistoryTotals struct {
	BatteryWh float64
	SolarWh   float64
	HomeWh    float64
	GridWh    float64
}

type energyHistoryResponse struct {
	Response struct {
		TimeSeries []energyHistoryPoint `json:"time_series"`
	} `json:"response"`
}

type energyHistoryPoint struct {
	Timestamp                   string  `json:"timestamp"`
	BatteryExported             float64 `json:"battery_energy_exported"`
	BatteryImportedFromSolar    float64 `json:"battery_energy_imported_from_solar"`
	BatteryImportedFromGrid     float64 `json:"battery_energy_imported_from_grid"`
	GridImported                float64 `json:"grid_energy_imported"`
	GridExported                float64 `json:"grid_energy_exported"`
	GridExportedFromSolar       float64 `json:"grid_energy_exported_from_solar"`
	GridExportedFromBattery     float64 `json:"grid_energy_exported_from_battery"`
	ConsumerImported            float64 `json:"consumer_energy_imported"`
	ConsumerImportedFromGrid    float64 `json:"consumer_energy_imported_from_grid"`
	ConsumerImportedFromSolar   float64 `json:"consumer_energy_imported_from_solar"`
	ConsumerImportedFromBattery float64 `json:"consumer_energy_imported_from_battery"`
	SolarExported               float64 `json:"solar_energy_exported"`
}

type soeHistoryResponse struct {
	Response struct {
		TimeSeries []struct {
			Timestamp string  `json:"timestamp"`
			SOE       float64 `json:"soe"`
		} `json:"time_series"`
	} `json:"response"`
}

func (c *FleetClient) FetchCalendarHistory(siteID int, end time.Time, timeZone string) (*CalendarHistory, error) {
	if timeZone == "" {
		timeZone = "Etc/UTC"
	}
	now := time.Now()
	effectiveEnd := end
	if !sameLocalDay(end, now) {
		y, m, d := end.Date()
		effectiveEnd = time.Date(y, m, d, 23, 59, 59, 0, end.Location())
	}
	values := url.Values{}
	values.Set("period", "day")
	values.Set("start_date", effectiveEnd.Add(-24*time.Hour).Format(time.RFC3339))
	values.Set("end_date", effectiveEnd.Format(time.RFC3339))
	values.Set("time_zone", timeZone)

	var energy energyHistoryResponse
	values.Set("kind", "energy")
	endpoint := fmt.Sprintf("%s/api/1/energy_sites/%d/calendar_history?%s", c.BaseURL, siteID, values.Encode())
	if err := c.getJSON(endpoint, &energy); err != nil {
		return nil, err
	}
	var soe soeHistoryResponse
	values.Set("kind", "soe")
	endpoint = fmt.Sprintf("%s/api/1/energy_sites/%d/calendar_history?%s", c.BaseURL, siteID, values.Encode())
	if err := c.getJSON(endpoint, &soe); err != nil {
		return nil, err
	}

	history := &CalendarHistory{}
	// Fleet calendar values are energy buckets. The source chart converts each
	// bucket to kW with /84; store equivalent watts for the GTK chart.
	const bucketWhToWatts = 1000.0 / 84.0
	for _, sample := range energy.Response.TimeSeries {
		at, err := time.Parse(time.RFC3339, sample.Timestamp)
		if err != nil {
			continue
		}
		batteryWh := sample.BatteryExported - sample.BatteryImportedFromSolar - sample.BatteryImportedFromGrid
		homeWh := sample.ConsumerImportedFromGrid + sample.ConsumerImportedFromSolar + sample.ConsumerImportedFromBattery
		if homeWh == 0 {
			homeWh = sample.ConsumerImported
		}
		exportedWh := sample.GridExportedFromSolar + sample.GridExportedFromBattery
		if exportedWh == 0 {
			exportedWh = sample.GridExported
		}
		gridWh := sample.GridImported - exportedWh
		batteryFlow := "blue"
		if batteryWh >= 0 && sample.GridExportedFromBattery > 40 {
			batteryFlow = "gray"
		} else if batteryWh < 0 {
			batteryFlow = "gray"
			if sample.BatteryImportedFromGrid+40 <= sample.BatteryImportedFromSolar {
				batteryFlow = "yellow"
			}
		}
		homeFlow := "gray"
		if sample.ConsumerImportedFromSolar >= sample.ConsumerImportedFromGrid && sample.ConsumerImportedFromSolar >= sample.ConsumerImportedFromBattery {
			homeFlow = "yellow"
		} else if sample.ConsumerImportedFromBattery >= sample.ConsumerImportedFromGrid {
			homeFlow = "green"
		}
		gridFlow := "gray"
		if gridWh < 0 {
			if sample.GridExportedFromSolar >= sample.GridExportedFromBattery {
				gridFlow = "yellow"
			} else {
				gridFlow = "green"
			}
		}
		history.Battery = append(history.Battery, HistorySample{Time: at, Value: batteryWh * bucketWhToWatts, Flow: batteryFlow})
		history.Solar = append(history.Solar, HistorySample{Time: at, Value: sample.SolarExported * bucketWhToWatts, Flow: "yellow"})
		history.Home = append(history.Home, HistorySample{Time: at, Value: homeWh * bucketWhToWatts, Flow: homeFlow})
		history.Grid = append(history.Grid, HistorySample{Time: at, Value: gridWh * bucketWhToWatts, Flow: gridFlow})
		history.Totals.BatteryWh += batteryWh
		history.Totals.SolarWh += sample.SolarExported
		history.Totals.HomeWh += homeWh
		history.Totals.GridWh += gridWh
	}
	for _, sample := range soe.Response.TimeSeries {
		if at, err := time.Parse(time.RFC3339, sample.Timestamp); err == nil {
			history.SOE = append(history.SOE, HistorySample{Time: at, Value: sample.SOE, Flow: "green"})
		}
	}
	return history, nil
}

func (c *FleetClient) FetchSolarEnergyToday(siteID int, now time.Time, timeZone string) (float64, error) {
	if timeZone == "" {
		timeZone = "Etc/UTC"
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	values := url.Values{}
	values.Set("kind", "energy")
	values.Set("period", "day")
	values.Set("start_date", start.Format(time.RFC3339))
	values.Set("end_date", now.Format(time.RFC3339))
	values.Set("time_zone", timeZone)
	endpoint := fmt.Sprintf("%s/api/1/energy_sites/%d/calendar_history?%s", c.BaseURL, siteID, values.Encode())
	var response energyHistoryResponse
	if err := c.getJSON(endpoint, &response); err != nil {
		return 0, err
	}
	total := 0.0
	for _, point := range response.Response.TimeSeries {
		total += point.SolarExported
	}
	return total, nil
}

func sameLocalDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status, Body: body}
	}
	return json.Unmarshal(body, v)
}

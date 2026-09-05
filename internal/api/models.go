package api

type BatteryPercentage struct {
	Percentage float64 `json:"percentage"`
}

type GridStatus struct {
	Status string `json:"status"`
}

type PowerwallData struct {
	Battery struct {
		InstantPower float64 `json:"instant_power"`
		Count        float64 `json:"num_meters_aggregated"`
	} `json:"battery"`
	Load struct {
		InstantPower float64 `json:"instant_power"`
	} `json:"load"`
	Solar struct {
		InstantPower   float64 `json:"instant_power"`
		EnergyExported float64 `json:"energy_exported"`
	} `json:"solar"`
	Site struct {
		InstantPower float64 `json:"instant_power"`
	} `json:"site"`
}

type LocalSnapshot struct {
	Data           PowerwallData
	BatteryPercent BatteryPercentage
	GridStatus     GridStatus
	SiteName       string
	WallConnector  *WallConnectorVitals
	Warning        string
}

type LocalSiteInfo struct {
	SiteName string `json:"site_name"`
}

type WallConnectorVitals struct {
	ContactorClosed    bool    `json:"contactor_closed"`
	VehicleConnected   bool    `json:"vehicle_connected"`
	GridVolts          float64 `json:"grid_v"`
	VehicleCurrentAmps float64 `json:"vehicle_current_a"`
}

func (v WallConnectorVitals) Power() float64 {
	return v.GridVolts * v.VehicleCurrentAmps
}

func (v WallConnectorVitals) Status() string {
	if v.ContactorClosed && v.VehicleCurrentAmps > 0 {
		return "Charging"
	}
	if v.VehicleConnected {
		return "Plugged in"
	}
	return "Idle"
}

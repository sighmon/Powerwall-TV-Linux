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
		InstantPower float64 `json:"instant_power"`
		EnergyExported float64 `json:"energy_exported"`
	} `json:"solar"`
	Site struct {
		InstantPower float64 `json:"instant_power"`
	} `json:"site"`
}

type LocalSnapshot struct {
	Data             PowerwallData
	BatteryPercent   BatteryPercentage
	GridStatus       GridStatus
}

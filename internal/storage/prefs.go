package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type LoginMode string

const (
	LoginModeLocal    LoginMode = "local"
	LoginModeFleetAPI LoginMode = "fleetAPI"
)

type Prefs struct {
	LoginMode                 LoginMode `json:"loginMode"`
	GatewayIP                 string    `json:"gatewayIP"`
	WallConnectorIP           string    `json:"wallConnectorIP"`
	Username                  string    `json:"username"`
	ShowLessPrecision         bool      `json:"showLessPrecision"`
	CurrentEnergySiteIdx      int       `json:"currentEnergySiteIndex"`
	FleetBaseURL              string    `json:"fleetBaseURL"`
	SceneScale                float64   `json:"sceneScale"`
	SceneHorizontalOffset     float64   `json:"sceneHorizontalOffset"`
	SceneVerticalOffset       float64   `json:"sceneVerticalOffset"`
	ElectricityMapsZone       string    `json:"electricityMapsZone"`
	AlwaysShowRuntimeEstimate bool      `json:"alwaysShowPowerwallRuntimeEstimate"`
	PreventScreenSaver        bool      `json:"preventScreenSaver"`
	KeepWindowInFront         bool      `json:"keepWindowInFront"`
	ShowInMenuBar             bool      `json:"showInMenuBar"`
	MenuBarLabelMetrics       string    `json:"menuBarLabelMetric"`
	ShowSchedulerButton       bool      `json:"showSchedulerButton"`
	LastChargingVIN           string    `json:"lastChargingWallConnectorVIN"`
	AutoHideSummaryOnOverlap  bool      `json:"autoHideSummaryOnOverlap"`
	AutoHideButtonsOnOverlap  bool      `json:"autoHideButtonsOnOverlap"`
}

func DefaultPrefs() Prefs {
	return Prefs{
		LoginMode:           LoginModeFleetAPI,
		Username:            "customer",
		FleetBaseURL:        "https://fleet-api.prd.na.vn.cloud.tesla.com",
		ShowLessPrecision:   false,
		SceneScale:          1,
		MenuBarLabelMetrics: "solar",
	}
}

func prefsPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "powerwall-tv", "prefs.json"), nil
}

func LoadPrefs() (Prefs, error) {
	path, err := prefsPath()
	if err != nil {
		return DefaultPrefs(), err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return DefaultPrefs(), err
	}
	var p Prefs
	if err := json.Unmarshal(b, &p); err != nil {
		return DefaultPrefs(), err
	}
	// fill defaults if missing
	if p.LoginMode == "" {
		p.LoginMode = DefaultPrefs().LoginMode
	}
	if p.Username == "" {
		p.Username = DefaultPrefs().Username
	}
	if p.FleetBaseURL == "" {
		p.FleetBaseURL = DefaultPrefs().FleetBaseURL
	}
	// Zero is the JSON default for settings written by older versions, but it
	// is not a valid persisted scene scale.
	if p.SceneScale == 0 {
		p.SceneScale = 1
	}
	if p.MenuBarLabelMetrics == "" {
		p.MenuBarLabelMetrics = "solar"
	}
	return p, nil
}

func SavePrefs(p Prefs) error {
	path, err := prefsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

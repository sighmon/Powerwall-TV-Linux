package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type VehicleChargeCacheEntry struct {
	BatteryLevel        *float64  `json:"batteryLevel,omitempty"`
	ChargingState       string    `json:"chargingState,omitempty"`
	MinutesToFullCharge *float64  `json:"minutesToFullCharge,omitempty"`
	FetchedAt           time.Time `json:"fetchedAt"`
}

func vehicleChargeCachePath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "powerwall-tv", "vehicle-charge-cache.json"), nil
}

func LoadVehicleChargeCache(now time.Time) map[string]VehicleChargeCacheEntry {
	path, err := vehicleChargeCachePath()
	if err != nil {
		return map[string]VehicleChargeCacheEntry{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]VehicleChargeCacheEntry{}
	}
	cache := map[string]VehicleChargeCacheEntry{}
	if json.Unmarshal(data, &cache) != nil {
		return map[string]VehicleChargeCacheEntry{}
	}
	cutoff := now.Add(-time.Hour)
	for vin, entry := range cache {
		if entry.FetchedAt.Before(cutoff) {
			delete(cache, vin)
		}
	}
	return cache
}

func SaveVehicleChargeCache(cache map[string]VehicleChargeCacheEntry) error {
	path, err := vehicleChargeCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func ClearVehicleChargeCache() error {
	path, err := vehicleChargeCachePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

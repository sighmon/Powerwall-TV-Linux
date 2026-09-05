package storage

import (
	"testing"
	"time"
)

func TestVehicleChargeCacheRoundTripAndExpiry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	level := 72.0
	if err := SaveVehicleChargeCache(map[string]VehicleChargeCacheEntry{
		"current": {BatteryLevel: &level, ChargingState: "Charging", FetchedAt: now.Add(-time.Minute)},
		"expired": {BatteryLevel: &level, FetchedAt: now.Add(-2 * time.Hour)},
	}); err != nil {
		t.Fatal(err)
	}
	cache := LoadVehicleChargeCache(now)
	if len(cache) != 1 || cache["current"].BatteryLevel == nil || *cache["current"].BatteryLevel != 72 {
		t.Fatalf("cache = %#v", cache)
	}
	if err := ClearVehicleChargeCache(); err != nil {
		t.Fatal(err)
	}
	if got := LoadVehicleChargeCache(now); len(got) != 0 {
		t.Fatalf("cache after clear = %#v", got)
	}
}

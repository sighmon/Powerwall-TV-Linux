package api

import "testing"

func TestWallConnectorVitals(t *testing.T) {
	tests := []struct {
		name   string
		vitals WallConnectorVitals
		power  float64
		status string
	}{
		{name: "idle", vitals: WallConnectorVitals{GridVolts: 230}, status: "Idle"},
		{name: "connected", vitals: WallConnectorVitals{VehicleConnected: true, GridVolts: 230}, status: "Plugged in"},
		{name: "charging", vitals: WallConnectorVitals{ContactorClosed: true, VehicleConnected: true, GridVolts: 230, VehicleCurrentAmps: 16}, power: 3680, status: "Charging"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vitals.Power(); got != tt.power {
				t.Fatalf("Power() = %v, want %v", got, tt.power)
			}
			if got := tt.vitals.Status(); got != tt.status {
				t.Fatalf("Status() = %q, want %q", got, tt.status)
			}
		})
	}
}

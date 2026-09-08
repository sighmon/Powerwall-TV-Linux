package ui

import (
	"testing"
	"time"

	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/schedule"
)

func TestCalculateSceneRect(t *testing.T) {
	tests := []struct {
		name                       string
		width, height, scale, x, y float64
		want                       sceneRect
	}{
		{
			name:  "natural size is biased left but clamped",
			width: 1280, height: 720, scale: 1,
			want: sceneRect{x: 0, y: 0, width: 1280, height: 720},
		},
		{
			name:  "large window scales uniformly",
			width: 1920, height: 1080, scale: 1,
			want: sceneRect{x: 0, y: 0, width: 1920, height: 1080},
		},
		{
			name:  "wide window applies source left bias",
			width: 1600, height: 720, scale: 1,
			want: sceneRect{x: 0, y: 0, width: 1280, height: 720},
		},
		{
			name:  "small window crops natural scene",
			width: 800, height: 600, scale: 1,
			want: sceneRect{x: -320, y: -60, width: 1280, height: 720},
		},
		{
			name:  "user scale and offsets are applied",
			width: 1600, height: 900, scale: 0.8, x: 0.2, y: -0.2,
			want: sceneRect{x: 320, y: 0, width: 1280, height: 720},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSceneRect(tt.width, tt.height, tt.scale, tt.x, tt.y)
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestHomeControlsPositionKeepsMinimumEdgePadding(t *testing.T) {
	x, y := homeControlsPosition(720, 36)
	if x != 24 || y != 660 {
		t.Fatalf("normal controls position = (%d, %d), want (24, 660)", x, y)
	}
	x, y = homeControlsPosition(60, 36)
	if x != 24 || y != 24 {
		t.Fatalf("short-window controls position = (%d, %d), want (24, 24)", x, y)
	}
}

func float64Pointer(value float64) *float64 { return &value }

func TestFleetWallConnectorSummary(t *testing.T) {
	connectors := []api.WallConnector{
		{WallConnectorState: float64Pointer(4), WallConnectorPower: float64Pointer(0)},
		{WallConnectorState: float64Pointer(1), WallConnectorPower: float64Pointer(7200)},
	}
	power, status := fleetWallConnectorSummary(connectors)
	if power != 7200 || status != "Charging" {
		t.Fatalf("got (%v, %q), want (7200, Charging)", power, status)
	}
}

func TestVehicleUnavailableRecognizesTeslaOfflineResponse(t *testing.T) {
	err := &api.HTTPStatusError{StatusCode: 408, Body: []byte(`{"error":"vehicle unavailable: asleep"}`)}
	if !vehicleUnavailable(err) {
		t.Fatal("expected asleep response to be unavailable")
	}
	err.StatusCode = 403
	if vehicleUnavailable(err) {
		t.Fatal("permission response must not be cached as offline")
	}
}

func TestFleetGridLabelText(t *testing.T) {
	tests := []struct {
		grid, island string
		want         string
	}{
		{want: "GRID"},
		{island: "off_grid_intentional", want: "OFF-GRID"},
		{island: "off_grid_unintentional", want: "GRID DOWN"},
		{grid: "SystemIslandedActive", want: "OFF-GRID"},
	}
	for _, tt := range tests {
		if got := fleetGridLabelText(tt.grid, tt.island); got != tt.want {
			t.Errorf("fleetGridLabelText(%q, %q) = %q, want %q", tt.grid, tt.island, got, tt.want)
		}
	}
}

func TestBatteryStatusLabelIncludesSchedulerAndStormWatch(t *testing.T) {
	store := schedule.Store{Enabled: true, Schedules: []schedule.Schedule{{Enabled: true}, {Enabled: true}}}
	if got, want := batteryStatusLabel(2, store, true), "POWERWALL · 2x · ⏰ 2 · ⚡"; got != want {
		t.Fatalf("batteryStatusLabel = %q, want %q", got, want)
	}
}

func TestScheduleActiveNowHandlesDaytimeAndOvernightWindows(t *testing.T) {
	store := schedule.Store{Enabled: true, Schedules: []schedule.Schedule{{Enabled: true, StartMinutes: 15 * 60, EndMinutes: 21 * 60}}}
	if !scheduleActiveNow(store, time.Date(2026, 9, 5, 16, 0, 0, 0, time.Local)) {
		t.Fatal("daytime schedule should be active")
	}
	if scheduleActiveNow(store, time.Date(2026, 9, 5, 21, 0, 0, 0, time.Local)) {
		t.Fatal("schedule should be inactive at its end boundary")
	}
	store.Schedules[0].StartMinutes = 21 * 60
	store.Schedules[0].EndMinutes = 6 * 60
	if !scheduleActiveNow(store, time.Date(2026, 9, 5, 23, 0, 0, 0, time.Local)) ||
		!scheduleActiveNow(store, time.Date(2026, 9, 6, 5, 0, 0, 0, time.Local)) {
		t.Fatal("overnight schedule should be active on both sides of midnight")
	}
}

func TestCalculateSceneRectClampsPreferences(t *testing.T) {
	got := calculateSceneRect(1600, 900, 9, -9, 9)
	want := calculateSceneRect(1600, 900, maxSceneScale, minSceneOffset, maxSceneOffset)
	if got != want {
		t.Fatalf("unclamped preferences: got %#v, want %#v", got, want)
	}
}

func TestCalculateSceneRectKeepsExpandedRightContentVisible(t *testing.T) {
	withoutCarbon := calculateSceneRectForContent(800, 720, 1, 0, 0, 0.34)
	withCarbon := calculateSceneRectForContent(800, 720, 1, 0, 0, 0.40)
	if withCarbon.x >= withoutCarbon.x {
		t.Fatalf("carbon scene x = %.1f, want left of %.1f", withCarbon.x, withoutCarbon.x)
	}
	rightContent := withCarbon.x + withCarbon.width*(0.5+0.40)
	if rightContent > 800.001 {
		t.Fatalf("right content = %.1f, outside window", rightContent)
	}
}

func TestSceneIntersection(t *testing.T) {
	scene := sceneRect{x: 100, y: 50, width: 400, height: 300}
	if !scene.intersects(20, 300, 120, 40) {
		t.Fatal("expected partial intersection")
	}
	if scene.intersects(20, 400, 120, 40) {
		t.Fatal("unexpected intersection")
	}
}

func TestFormatKWMatchesSourcePrecisionBehavior(t *testing.T) {
	if got := formatKW(-1234, false); got != "-1.234 kW" {
		t.Fatalf("full precision = %q", got)
	}
	if got := formatKW(-1234, true); got != "1.2 kW" {
		t.Fatalf("reduced precision = %q", got)
	}
}

func TestIntroEaseOutHonorsDelayAndDuration(t *testing.T) {
	start := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if got := introEaseOut(start, 1200*time.Millisecond, 700*time.Millisecond, start.Add(time.Second)); got != 0 {
		t.Fatalf("progress before delay = %v", got)
	}
	if got := introEaseOut(start, 1200*time.Millisecond, 700*time.Millisecond, start.Add(1900*time.Millisecond)); got != 1 {
		t.Fatalf("progress after duration = %v", got)
	}
	if got := introEaseOut(start, 0, time.Second, start.Add(500*time.Millisecond)); got <= 0.5 || got >= 1 {
		t.Fatalf("ease-out midpoint = %v", got)
	}
}

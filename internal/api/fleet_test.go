package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body))}
}

func TestFetchCalendarHistory(t *testing.T) {
	requests := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Query().Get("time_zone") != "Australia/Adelaide" {
			t.Errorf("time_zone = %q", r.URL.Query().Get("time_zone"))
		}
		switch r.URL.Query().Get("kind") {
		case "energy":
			return jsonResponse(`{"response":{"time_series":[{"timestamp":"2026-09-05T00:00:00Z","battery_energy_exported":200,"battery_energy_imported_from_solar":50,"battery_energy_imported_from_grid":25,"grid_energy_imported":100,"grid_energy_exported_from_solar":20,"grid_energy_exported_from_battery":10,"consumer_energy_imported_from_grid":40,"consumer_energy_imported_from_solar":60,"consumer_energy_imported_from_battery":10,"solar_energy_exported":150}]}}`), nil
		case "soe":
			return jsonResponse(`{"response":{"time_series":[{"timestamp":"2026-09-05T00:00:00Z","soe":72.5}]}}`), nil
		default:
			return nil, fmt.Errorf("unexpected kind")
		}
	})

	client := NewFleetClient("https://fleet.example", "token")
	client.client.Transport = transport
	end := time.Date(2026, 9, 5, 12, 0, 0, 0, time.FixedZone("ACST", 9*60*60+30*60))
	history, err := client.FetchCalendarHistory(42, end, "Australia/Adelaide")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if history.Totals.BatteryWh != 125 || history.Totals.SolarWh != 150 || history.Totals.HomeWh != 110 || history.Totals.GridWh != 70 {
		t.Fatalf("unexpected totals: %#v", history.Totals)
	}
	if len(history.SOE) != 1 || history.SOE[0].Value != 72.5 {
		t.Fatalf("unexpected SOE: %#v", history.SOE)
	}
	if got, want := history.Battery[0].Value, 125*1000.0/84.0; got != want {
		t.Fatalf("battery chart value = %v, want %v", got, want)
	}
	if history.Battery[0].Flow != "blue" || history.Home[0].Flow != "yellow" || history.Grid[0].Flow != "gray" {
		t.Fatalf("unexpected flow colors: battery=%q home=%q grid=%q", history.Battery[0].Flow, history.Home[0].Flow, history.Grid[0].Flow)
	}
}

func TestFetchSnapshotClampsSiteIndex(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/1/products" {
			return jsonResponse(`{"response":[{"energy_site_id":1,"site_name":"One"},{"energy_site_id":2,"site_name":"Two"}]}`), nil
		}
		return jsonResponse(`{"response":{"percentage_charged":50}}`), nil
	})

	client := NewFleetClient("https://fleet.example", "token")
	client.client.Transport = transport
	snapshot, idx, err := client.FetchSnapshot(-1)
	if err != nil {
		t.Fatal(err)
	}
	if idx != 0 || snapshot.SiteID != 1 || snapshot.SiteName != "One" {
		t.Fatalf("got index %d and snapshot %#v", idx, snapshot)
	}
}

func TestProductDisplayLabelUsesFleetFallbackOrder(t *testing.T) {
	product := Product{SiteName: "site", Title: "title", Name: "name", SiteDisplayName: "site display", DisplayName: "display"}
	if got := product.DisplayLabel(); got != "display" {
		t.Fatalf("DisplayLabel = %q, want display", got)
	}
	product.DisplayName = ""
	if got := product.DisplayLabel(); got != "site display" {
		t.Fatalf("DisplayLabel fallback = %q, want site display", got)
	}
}

func TestSetOperationModeReturnsHTTPStatusError(t *testing.T) {
	client := NewFleetClient("https://fleet.example", "token")
	client.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden", Body: io.NopCloser(strings.NewReader(`{"error":"missing scope"}`))}, nil
	})
	err := client.SetOperationMode(42, "autonomous")
	statusErr, ok := err.(*HTTPStatusError)
	if !ok || statusErr.StatusCode != http.StatusForbidden || string(statusErr.Body) != `{"error":"missing scope"}` {
		t.Fatalf("SetOperationMode error = %#v", err)
	}
}

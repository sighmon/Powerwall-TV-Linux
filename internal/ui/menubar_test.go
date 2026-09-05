package ui

import "testing"

func TestMenuBarSelectionReadsLegacyAndMultipleValuesInDisplayOrder(t *testing.T) {
	assertMenuMetrics(t, selectedMenuBarMetrics("battery"), []menuBarMetric{menuBattery})
	assertMenuMetrics(t, selectedMenuBarMetrics("battery,solar,site"), []menuBarMetric{menuSolar, menuSite, menuBattery})
	assertMenuMetrics(t, selectedMenuBarMetrics("unknown"), []menuBarMetric{menuSolar})
}

func TestMenuBarSelectionTogglesAndKeepsOne(t *testing.T) {
	if got := toggleMenuBarMetric(menuLoad, "solar"); got != "solar,load" {
		t.Fatalf("add = %q", got)
	}
	if got := toggleMenuBarMetric(menuSolar, "solar,load"); got != "load" {
		t.Fatalf("remove = %q", got)
	}
	if got := toggleMenuBarMetric(menuSolar, "solar"); got != "solar" {
		t.Fatalf("last removal = %q", got)
	}
}

func TestAutomaticMenuBarMetricUsesGreatestAbsoluteFlow(t *testing.T) {
	if got := automaticMenuBarMetric(1200, 3400, 500, -5900); got != menuBattery {
		t.Fatalf("automatic = %q", got)
	}
	if got := automaticMenuBarMetric(6100, 3400, -5900, 200); got != menuSolar {
		t.Fatalf("automatic = %q", got)
	}
}

func assertMenuMetrics(t *testing.T, got, want []menuBarMetric) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("metrics = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("metrics = %v, want %v", got, want)
		}
	}
}

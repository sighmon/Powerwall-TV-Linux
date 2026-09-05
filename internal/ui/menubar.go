package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

type menuBarMetric string

const (
	menuAuto    menuBarMetric = "auto"
	menuSolar   menuBarMetric = "solar"
	menuLoad    menuBarMetric = "load"
	menuSite    menuBarMetric = "site"
	menuBattery menuBarMetric = "battery"
)

var menuBarMetrics = []menuBarMetric{menuAuto, menuSolar, menuLoad, menuSite, menuBattery}

func selectedMenuBarMetrics(raw string) []menuBarMetric {
	selected := make(map[menuBarMetric]bool)
	for _, value := range strings.Split(raw, ",") {
		selected[menuBarMetric(value)] = true
	}
	var result []menuBarMetric
	for _, metric := range menuBarMetrics {
		if selected[metric] {
			result = append(result, metric)
		}
	}
	if len(result) == 0 {
		return []menuBarMetric{menuSolar}
	}
	return result
}

func menuBarMetricsRaw(metrics []menuBarMetric) string {
	selected := make(map[menuBarMetric]bool)
	for _, metric := range metrics {
		selected[metric] = true
	}
	var ordered []string
	for _, metric := range menuBarMetrics {
		if selected[metric] {
			ordered = append(ordered, string(metric))
		}
	}
	if len(ordered) == 0 {
		return string(menuSolar)
	}
	return strings.Join(ordered, ",")
}

func toggleMenuBarMetric(metric menuBarMetric, raw string) string {
	metrics := selectedMenuBarMetrics(raw)
	for i, existing := range metrics {
		if existing == metric {
			if len(metrics) == 1 {
				return menuBarMetricsRaw(metrics)
			}
			return menuBarMetricsRaw(append(metrics[:i], metrics[i+1:]...))
		}
	}
	return menuBarMetricsRaw(append(metrics, metric))
}

func automaticMenuBarMetric(solar, load, site, battery float64) menuBarMetric {
	values := []struct {
		metric menuBarMetric
		watts  float64
	}{{menuSolar, solar}, {menuLoad, load}, {menuSite, site}, {menuBattery, battery}}
	best := values[0]
	for _, value := range values[1:] {
		if math.Abs(value.watts) > math.Abs(best.watts) {
			best = value
		}
	}
	return best.metric
}

func buildStatusIcon(state *statepkg.State, dashboard *dashboardWidgets, parent gtk.IWindow, app *gtk.Application) *statusIcon {
	icon := newStatusIcon("com.sighmon.PowerwallTV")
	if icon == nil {
		return nil
	}
	icon.setVisible(state.Prefs.ShowInMenuBar)
	icon.Connect("activate", func() { parent.ToWindow().Present() })

	menu, _ := gtk.MenuNew()
	show, _ := gtk.MenuItemNewWithLabel("Show Powerwall TV")
	show.Connect("activate", func() { parent.ToWindow().Present() })
	menu.Append(show)
	separator, _ := gtk.SeparatorMenuItemNew()
	menu.Append(separator)
	for _, metric := range menuBarMetrics {
		metric := metric
		item, _ := gtk.CheckMenuItemNewWithLabel(menuBarMetricTitle(metric))
		item.SetActive(containsMenuBarMetric(selectedMenuBarMetrics(state.Prefs.MenuBarLabelMetrics), metric))
		item.Connect("toggled", func() {
			state.Prefs.MenuBarLabelMetrics = toggleMenuBarMetric(metric, state.Prefs.MenuBarLabelMetrics)
			_ = storage.SavePrefs(state.Prefs)
			shouldBeActive := containsMenuBarMetric(selectedMenuBarMetrics(state.Prefs.MenuBarLabelMetrics), metric)
			if item.GetActive() != shouldBeActive {
				item.SetActive(shouldBeActive)
			}
		})
		menu.Append(item)
	}
	separator, _ = gtk.SeparatorMenuItemNew()
	menu.Append(separator)
	quit, _ := gtk.MenuItemNewWithLabel("Quit")
	quit.Connect("activate", func() { app.Quit() })
	menu.Append(quit)
	menu.ShowAll()
	icon.Connect("popup-menu", func() { icon.popup(menu) })

	update := func() {
		text := menuBarStatusText(state.Prefs, dashboard)
		icon.setTitle(text)
		icon.setTooltip(text)
	}
	update()
	glib.TimeoutAdd(1000, func() bool { update(); return true })
	return icon
}

func menuBarStatusText(prefs storage.Prefs, dashboard *dashboardWidgets) string {
	automatic := automaticMenuBarMetric(dashboard.solarP, dashboard.homeP, dashboard.gridP, dashboard.batteryP)
	rendered := make(map[menuBarMetric]bool)
	var values []string
	for _, metric := range selectedMenuBarMetrics(prefs.MenuBarLabelMetrics) {
		if metric == menuAuto {
			metric = automatic
		}
		if rendered[metric] {
			continue
		}
		rendered[metric] = true
		prefix, watts := menuBarMetricValue(metric, dashboard)
		values = append(values, prefix+" "+formatKW(watts, prefs.ShowLessPrecision))
	}
	trend := ""
	if dashboard.batteryP > 40 {
		trend = " ▼"
	} else if dashboard.batteryP < -40 {
		trend = " ▲"
	}
	values = append(values, fmt.Sprintf("%.0f%%%s", dashboard.batteryPercent, trend))
	text := strings.Join(values, " · ")
	if dashboard.stormWatch {
		text += " ☂"
	}
	if dashboard.gridBaseLabel == "GRID DOWN" {
		text += " ⛔"
	}
	return text
}

func menuBarMetricValue(metric menuBarMetric, dashboard *dashboardWidgets) (string, float64) {
	switch metric {
	case menuLoad:
		return "⌂", dashboard.homeP
	case menuSite:
		return "⇄", dashboard.gridP
	case menuBattery:
		return "⚡", dashboard.batteryP
	default:
		return "☀", dashboard.solarP
	}
}

func menuBarMetricTitle(metric menuBarMetric) string {
	switch metric {
	case menuAuto:
		return "Auto"
	case menuLoad:
		return "Home"
	case menuSite:
		return "Grid"
	case menuBattery:
		return "Battery"
	default:
		return "Solar"
	}
}

func containsMenuBarMetric(metrics []menuBarMetric, wanted menuBarMetric) bool {
	for _, metric := range metrics {
		if metric == wanted {
			return true
		}
	}
	return false
}

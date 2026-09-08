package ui

import (
	"os"
	"strconv"
	"testing"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

// TestDashboardRender is an opt-in screenshot smoke test. It is skipped by the
// normal suite; set POWERWALL_VISUAL_OUTPUT to capture a real GTK render under
// X11/Wayland. Width and height default to the source scene's natural size.
func TestDashboardRender(t *testing.T) {
	output := os.Getenv("POWERWALL_VISUAL_OUTPUT")
	if output == "" {
		t.Skip("set POWERWALL_VISUAL_OUTPUT to run the GTK render smoke test")
	}
	if err := gtk.InitCheck(nil); err != nil {
		t.Fatal(err)
	}
	width := visualDimension("POWERWALL_VISUAL_WIDTH", 1280)
	height := visualDimension("POWERWALL_VISUAL_HEIGHT", 720)
	initialWidth := visualDimension("POWERWALL_VISUAL_INITIAL_WIDTH", width)
	initialHeight := visualDimension("POWERWALL_VISUAL_INITIAL_HEIGHT", height)
	window, err := gtk.OffscreenWindowNew()
	if err != nil {
		t.Fatal(err)
	}
	window.SetDefaultSize(initialWidth, initialHeight)
	state := &statepkg.State{Prefs: storage.DefaultPrefs()}
	state.Prefs.LoginMode = storage.LoginModeLocal
	state.Prefs.GatewayIP = "demo"
	app, err := gtk.ApplicationNew("com.sighmon.PowerwallTV.VisualTest", glib.APPLICATION_NON_UNIQUE)
	if err != nil {
		t.Fatal(err)
	}
	mainView, err := buildMainView(state, window, app)
	if err != nil {
		t.Fatal(err)
	}
	window.Add(mainView)
	window.ShowAll()
	if initialWidth != width || initialHeight != height {
		glib.TimeoutAdd(250, func() bool {
			window.SetDefaultSize(width, height)
			window.Resize(width, height)
			return false
		})
	}

	var captureErr error
	glib.TimeoutAdd(2300, func() bool {
		pixbuf, err := window.GetPixbuf()
		if err == nil {
			captureErr = pixbuf.SavePNG(output, 9)
		} else {
			captureErr = err
		}
		window.Destroy()
		gtk.MainQuit()
		return false
	})
	gtk.Main()
	if captureErr != nil {
		t.Fatal(captureErr)
	}
}

func visualDimension(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// Exercise real GTK allocations after delayed data and in both resize directions.
func TestDashboardMetricSpacing(t *testing.T) {
	if os.Getenv("POWERWALL_VISUAL_OUTPUT") == "" {
		t.Skip("set POWERWALL_VISUAL_OUTPUT to run the GTK layout regression test")
	}
	if err := gtk.InitCheck(nil); err != nil {
		t.Fatal(err)
	}
	window, err := gtk.OffscreenWindowNew()
	if err != nil {
		t.Fatal(err)
	}
	window.SetDefaultSize(1280, 720)
	state := &statepkg.State{Prefs: storage.DefaultPrefs()}
	state.Prefs.LoginMode = storage.LoginModeLocal
	state.Prefs.GatewayIP = "demo"
	root, w, err := buildDashboard(state, NewChartsData(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	window.Add(root)
	window.ShowAll()
	check := func(stage string, width, height int) {
		if w.width != width || w.height != height {
			t.Errorf("%s: dashboard size = %dx%d, want %dx%d", stage, w.width, w.height, width, height)
		}
		for value, box := range w.metricBoxes {
			children := box.GetChildren()
			label := children.NthData(1).(*gtk.Widget)
			children.Free()
			va, la := value.GetAllocation(), label.GetAllocation()
			if gap := la.GetY() - va.GetY() - va.GetHeight(); gap != metricLineSpacing {
				name, _ := value.GetText()
				t.Errorf("%s: %s gap = %d, want %d", stage, name, gap, metricLineSpacing)
			}
		}
		if w.solarVal.GetLineWrap() {
			t.Errorf("%s: solar value unexpectedly wraps", stage)
		}
	}
	glib.TimeoutAdd(100, func() bool {
		for value := range w.metricBoxes {
			setValue(value, "")
		}
		return false
	})
	glib.TimeoutAdd(250, func() bool {
		for value := range w.metricBoxes {
			setValue(value, "12.345 kW")
		}
		return false
	})
	glib.TimeoutAdd(450, func() bool {
		check("late data", 1280, 720)
		window.SetDefaultSize(800, 600)
		window.Resize(800, 600)
		return false
	})
	glib.TimeoutAdd(650, func() bool {
		check("shrink", 800, 600)
		window.SetDefaultSize(1600, 900)
		window.Resize(1600, 900)
		return false
	})
	glib.TimeoutAdd(900, func() bool {
		check("grow", 1600, 900)
		window.Destroy()
		gtk.MainQuit()
		return false
	})
	gtk.Main()
}

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

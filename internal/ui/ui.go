package ui

import (
	"github.com/gotk3/gotk3/gtk"
	statepkg "powerwall-tv-gtk/internal/state"
)

func Run(state statepkg.State) error {
	gtk.Init(nil)

	app, err := gtk.ApplicationNew("com.sighmon.PowerwallTV", 0)
	if err != nil {
		return err
	}

	app.Connect("activate", func() {
		win, _ := gtk.ApplicationWindowNew(app)
		win.SetTitle("Powerwall TV")
		win.SetDefaultSize(1280, 720)
		win.SetIconName("com.sighmon.PowerwallTV")

		content, _ := buildMainView(&state, win, app)
		win.Add(content)
		win.ShowAll()
	})

	app.Run(nil)
	return nil
}

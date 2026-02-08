package state

import "powerwall-tv-gtk/internal/storage"

type State struct {
	Prefs storage.Prefs
}

func Load() State {
	prefs, _ := storage.LoadPrefs()
	return State{Prefs: prefs}
}

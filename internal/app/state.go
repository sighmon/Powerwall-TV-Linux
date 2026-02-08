package app

import "powerwall-tv-gtk/internal/storage"

type State struct {
	Prefs storage.Prefs
}

func LoadState() State {
	prefs, _ := storage.LoadPrefs()
	return State{Prefs: prefs}
}

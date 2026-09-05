package state

import "powerwall-tv-gtk/internal/storage"

type State struct {
	Prefs                  storage.Prefs
	FirmwareVersion        string
	InstallationDate       string
	VehicleCacheGeneration uint64
}

func Load() State {
	prefs, _ := storage.LoadPrefs()
	return State{Prefs: prefs}
}

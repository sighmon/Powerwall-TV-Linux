package app

import (
	"powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/ui"
)

func Run() error {
	st := state.Load()
	return ui.Run(st)
}

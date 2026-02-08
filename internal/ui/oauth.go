package ui

import (
	"context"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"powerwall-tv-gtk/internal/auth"
	statepkg "powerwall-tv-gtk/internal/state"
)

func startFleetLogin(parent gtk.IWindow, state *statepkg.State, status *gtk.Label) {
	secrets := auth.LoadSecrets()
	cfg := auth.Config{
		ClientID:     secrets.ClientID,
		ClientSecret: secrets.ClientSecret,
		Scopes:       "openid energy_device_data offline_access",
		FleetBaseURL: state.Prefs.FleetBaseURL,
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		status.SetText("Missing Tesla client ID/secret")
		return
	}

	status.SetText("Opening Tesla login…")

	authURL, callbackURL, ln, expectedState, err := auth.StartOAuth(cfg)
	if err != nil {
		status.SetText("OAuth start failed: " + err.Error())
		return
	}
	_ = auth.OpenBrowser(authURL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	code, err := auth.AwaitCallback(ctx, ln, expectedState)
	if err != nil {
		status.SetText("Login failed: " + err.Error())
		return
	}

	status.SetText("Exchanging token…")
	tok, err := auth.ExchangeCode(ctx, cfg, code, callbackURL)
	if err != nil {
		status.SetText("Token exchange failed: " + err.Error())
		return
	}

	auth.SaveStoredToken(*tok)

	glib.IdleAdd(func() {
		status.SetText("Logged in")
	})
}

package ui

import (
	"context"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/auth"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

func startFleetLogin(parent gtk.IWindow, state *statepkg.State, status *gtk.Label) {
	secrets := auth.LoadSecrets()
	cfg := auth.Config{
		ClientID:            secrets.ClientID,
		ClientSecret:        secrets.ClientSecret,
		Scopes:              "openid energy_device_data vehicle_device_data energy_cmds offline_access",
		FleetBaseURL:        state.Prefs.FleetBaseURL,
		PromptMissingScopes: true,
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		setOAuthStatus(status, "Missing Tesla client ID/secret")
		return
	}

	setOAuthStatus(status, "Opening Tesla login…")

	authURL, callbackURL, ln, expectedState, err := auth.StartOAuth(cfg)
	if err != nil {
		setOAuthStatus(status, "OAuth start failed: "+err.Error())
		return
	}
	_ = auth.OpenBrowser(authURL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	code, err := auth.AwaitCallback(ctx, ln, expectedState)
	if err != nil {
		setOAuthStatus(status, "Login failed: "+err.Error())
		return
	}

	setOAuthStatus(status, "Exchanging token…")
	tok, err := auth.ExchangeCode(ctx, cfg, code, callbackURL)
	if err != nil {
		setOAuthStatus(status, "Token exchange failed: "+err.Error())
		return
	}

	if err := auth.SaveStoredToken(*tok); err != nil {
		setOAuthStatus(status, "Could not save login: "+err.Error())
		return
	}
	if base, err := api.ResolveFleetBaseURL(tok.AccessToken); err == nil && base != "" && base != state.Prefs.FleetBaseURL {
		state.Prefs.FleetBaseURL = base
		_ = storage.SavePrefs(state.Prefs)
		storage.ClearFleetTokens()
		setOAuthStatus(status, "Tesla region found; restarting login…")
		startFleetLogin(parent, state, status)
		return
	}

	setOAuthStatus(status, "Logged in")
}

func setOAuthStatus(status *gtk.Label, text string) {
	glib.IdleAdd(func() { status.SetText(text) })
}

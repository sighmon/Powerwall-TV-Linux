package auth

import (
	"errors"
	"time"

	"powerwall-tv-gtk/internal/storage"
)

type StoredToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

func LoadStoredToken() (StoredToken, error) {
	access, _ := storage.GetFleetAccessToken()
	refresh, _ := storage.GetFleetRefreshToken()
	exp, _ := storage.GetFleetExpiry()
	if access == "" {
		return StoredToken{}, errors.New("missing access token")
	}
	var expiry time.Time
	if exp != "" {
		expiry, _ = time.Parse(time.RFC3339, exp)
	}
	return StoredToken{AccessToken: access, RefreshToken: refresh, Expiry: expiry}, nil
}

func SaveStoredToken(tok TokenResponse) {
	_ = storage.SetFleetAccessToken(tok.AccessToken)
	if tok.RefreshToken != "" {
		_ = storage.SetFleetRefreshToken(tok.RefreshToken)
	}
	if tok.ExpiresIn > 0 {
		_ = storage.SetFleetExpiry(time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339))
	}
}

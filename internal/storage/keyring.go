package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	serviceName          = "com.sighmon.PowerwallTV"
	legacyServiceName    = "org.sighmon.PowerwallTV"
	keyGatewayPassword   = "gatewayPassword"
	keyFleetAccessToken  = "fleet_access_token"
	keyFleetRefreshToken = "fleet_refresh_token"
	keyFleetExpiry       = "fleet_expiry"
)

type fallbackSecrets map[string]string

func fallbackSecretsPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "powerwall-tv", "secrets.json"), nil
}

func loadFallbackSecrets() fallbackSecrets {
	path, err := fallbackSecretsPath()
	if err != nil {
		return fallbackSecrets{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fallbackSecrets{}
	}
	out := fallbackSecrets{}
	if err := json.Unmarshal(b, &out); err != nil {
		return fallbackSecrets{}
	}
	return out
}

func saveFallbackSecrets(s fallbackSecrets) error {
	path, err := fallbackSecretsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func SetSecret(key, value string) error {
	if err := keyring.Set(serviceName, key, value); err == nil {
		return nil
	}
	// Flatpak/sandbox fallback when Secret Service is unavailable.
	s := loadFallbackSecrets()
	s[key] = value
	return saveFallbackSecrets(s)
}

func GetSecret(key string) (string, error) {
	if v, err := keyring.Get(serviceName, key); err == nil {
		return v, nil
	}
	// Migration support: read old org.* service if present.
	if v, err := keyring.Get(legacyServiceName, key); err == nil {
		_ = keyring.Set(serviceName, key, v)
		return v, nil
	}
	s := loadFallbackSecrets()
	if v, ok := s[key]; ok {
		return v, nil
	}
	return "", keyring.ErrNotFound
}

func DeleteSecret(key string) error {
	_ = keyring.Delete(serviceName, key)
	s := loadFallbackSecrets()
	delete(s, key)
	return saveFallbackSecrets(s)
}

func SetGatewayPassword(pw string) error { return SetSecret(keyGatewayPassword, pw) }
func GetGatewayPassword() (string, error) { return GetSecret(keyGatewayPassword) }

func SetFleetAccessToken(t string) error { return SetSecret(keyFleetAccessToken, t) }
func GetFleetAccessToken() (string, error) { return GetSecret(keyFleetAccessToken) }

func SetFleetRefreshToken(t string) error { return SetSecret(keyFleetRefreshToken, t) }
func GetFleetRefreshToken() (string, error) { return GetSecret(keyFleetRefreshToken) }

func SetFleetExpiry(ts string) error { return SetSecret(keyFleetExpiry, ts) }
func GetFleetExpiry() (string, error) { return GetSecret(keyFleetExpiry) }

func ClearFleetTokens() {
	_ = DeleteSecret(keyFleetAccessToken)
	_ = DeleteSecret(keyFleetRefreshToken)
	_ = DeleteSecret(keyFleetExpiry)
}

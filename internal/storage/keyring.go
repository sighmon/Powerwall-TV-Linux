package storage

import (
	"github.com/zalando/go-keyring"
)

const (
	serviceName         = "org.sighmon.PowerwallTV"
	keyGatewayPassword  = "gatewayPassword"
	keyFleetAccessToken = "fleet_access_token"
	keyFleetRefreshToken = "fleet_refresh_token"
	keyFleetExpiry      = "fleet_expiry"
)

func SetSecret(key, value string) error {
	return keyring.Set(serviceName, key, value)
}

func GetSecret(key string) (string, error) {
	return keyring.Get(serviceName, key)
}

func DeleteSecret(key string) error {
	return keyring.Delete(serviceName, key)
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

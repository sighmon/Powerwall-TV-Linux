//go:build ignore
// +build ignore

package auth

// Copy to secrets.go and fill in values, or use environment variables.
// This file is safe to commit.

type Secrets struct {
	ClientID     string
	ClientSecret string
}

func LoadSecrets() Secrets {
	return Secrets{
		ClientID:     "",
		ClientSecret: "",
	}
}

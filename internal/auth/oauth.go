package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	ClientID            string
	ClientSecret        string
	Scopes              string
	FleetBaseURL        string
	PromptMissingScopes bool
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func randomState() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func StartOAuth(cfg Config) (authURL string, callbackURL string, listener net.Listener, state string, err error) {
	state = randomState()
	path := oauthSocketPath()
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return "", "", nil, "", err
	}
	callbackURL = "powerwalltv://app/callback"

	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("scope", cfg.Scopes)
	q.Set("state", state)
	if cfg.PromptMissingScopes {
		q.Set("prompt_missing_scopes", "true")
	}

	authURL = "https://auth.tesla.com/oauth2/v3/authorize?" + q.Encode()
	return authURL, callbackURL, ln, state, nil
}

func AwaitCallback(ctx context.Context, ln net.Listener, expectedState string) (code string, err error) {
	defer os.Remove(oauthSocketPath())
	type result struct {
		code string
		err  error
	}
	results := make(chan result, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			results <- result{err: err}
			return
		}
		defer conn.Close()
		data, err := io.ReadAll(io.LimitReader(conn, 64*1024))
		if err != nil {
			results <- result{err: err}
			return
		}
		callback, err := parseOAuthCallback(string(data), expectedState)
		results <- result{code: callback, err: err}
	}()

	select {
	case <-ctx.Done():
		_ = ln.Close()
		return "", ctx.Err()
	case result := <-results:
		_ = ln.Close()
		return result.code, result.err
	}
}

func parseOAuthCallback(raw, expectedState string) (string, error) {
	callback, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if callback.Scheme != "powerwalltv" || callback.Host != "app" || callback.Path != "/callback" {
		return "", fmt.Errorf("invalid OAuth callback")
	}
	if callback.Query().Get("state") != expectedState {
		return "", fmt.Errorf("invalid OAuth state")
	}
	code := callback.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("missing OAuth code")
	}
	return code, nil
}

func ForwardOAuthCallback(args []string) (bool, error) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "powerwalltv://") {
			continue
		}
		conn, err := net.DialTimeout("unix", oauthSocketPath(), 5*time.Second)
		if err != nil {
			return true, err
		}
		_, writeErr := io.WriteString(conn, arg)
		closeErr := conn.Close()
		if writeErr != nil {
			return true, writeErr
		}
		return true, closeErr
	}
	return false, nil
}

func oauthSocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("powerwall-tv-oauth-%d.sock", os.Getuid()))
}

func ExchangeCode(ctx context.Context, cfg Config, code string, callbackURL string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", callbackURL)
	form.Set("audience", cfg.FleetBaseURL)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://fleet-auth.prd.vn.cloud.tesla.com/oauth2/v3/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	hc := &http.Client{Timeout: 10 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token exchange failed: %s", resp.Status)
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

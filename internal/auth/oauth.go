package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Scopes       string
	FleetBaseURL string
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
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", nil, "", err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	callbackURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("scope", cfg.Scopes)
	q.Set("state", state)

	authURL = "https://auth.tesla.com/oauth2/v3/authorize?" + q.Encode()
	return authURL, callbackURL, ln, state, nil
}

func AwaitCallback(ctx context.Context, ln net.Listener, expectedState string) (code string, err error) {
	mux := http.NewServeMux()
	result := make(chan string, 1)

	srv := &http.Server{Handler: mux}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("state") != expectedState {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid state"))
			return
		}
		code = r.FormValue("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing code"))
			return
		}
		_, _ = w.Write([]byte("Login complete. You can close this window."))
		result <- code
	})

	go func() {
		_ = srv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		_ = srv.Close()
		return "", ctx.Err()
	case code := <-result:
		_ = srv.Close()
		return code, nil
	}
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

package auth

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOAuthCustomCallbackRoundTrip(t *testing.T) {
	authURL, callbackURL, listener, state, err := StartOAuth(Config{
		ClientID: "client", Scopes: "openid offline_access", FleetBaseURL: "https://fleet.example", PromptMissingScopes: true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("sandbox does not allow Unix sockets")
		}
		t.Fatal(err)
	}
	defer listener.Close()
	if callbackURL != "powerwalltv://app/callback" {
		t.Fatalf("callback URL = %q", callbackURL)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("redirect_uri") != callbackURL || parsed.Query().Get("prompt_missing_scopes") != "true" {
		t.Fatalf("authorization query = %v", parsed.Query())
	}
	callback := callbackURL + "?code=abc123&state=" + url.QueryEscape(state)
	forwarded := make(chan error, 1)
	go func() {
		_, err := ForwardOAuthCallback([]string{callback})
		forwarded <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	code, err := AwaitCallback(ctx, listener, state)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-forwarded; err != nil {
		t.Fatal(err)
	}
	if code != "abc123" {
		t.Fatalf("code = %q", code)
	}
}

func TestParseOAuthCallbackRejectsWrongState(t *testing.T) {
	if _, err := parseOAuthCallback("powerwalltv://app/callback?code=abc&state=wrong", "right"); err == nil {
		t.Fatal("expected state validation error")
	}
}

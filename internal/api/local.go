package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"powerwall-tv-gtk/internal/util"
)

type LocalClient struct {
	IP       string
	Username string
	Password string
	client   *http.Client
}

func NewLocalClient(ip, username, password string) *LocalClient {
	return &LocalClient{
		IP:       ip,
		Username: username,
		Password: password,
		client:   util.InsecureClient(),
	}
}

func (c *LocalClient) login() error {
	url := fmt.Sprintf("https://%s/api/login/Basic", c.IP)
	payload := map[string]any{
		"username": "customer",
		"password": c.Password,
		"email":    c.Username,
		"force_sm_off": false,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("login failed: %s", resp.Status)
	}
	return nil
}

func (c *LocalClient) FetchSnapshot() (*LocalSnapshot, error) {
	if c.IP == "" || c.Password == "" {
		return nil, fmt.Errorf("missing gateway IP or password")
	}
	if err := c.login(); err != nil {
		return nil, err
	}

	snap := &LocalSnapshot{}
	if err := c.getJSON("/api/meters/aggregates", &snap.Data); err != nil {
		return nil, err
	}
	if err := c.getJSON("/api/system_status/soe", &snap.BatteryPercent); err != nil {
		return nil, err
	}
	if err := c.getJSON("/api/system_status/grid_status", &snap.GridStatus); err != nil {
		return nil, err
	}
	return snap, nil
}

func (c *LocalClient) getJSON(path string, v any) error {
	url := fmt.Sprintf("https://%s%s", c.IP, path)
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s failed: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

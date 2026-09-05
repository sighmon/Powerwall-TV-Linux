package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
		"username":     "customer",
		"password":     c.Password,
		"email":        c.Username,
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
	var site LocalSiteInfo
	if err := c.getJSON("/api/site_info", &site); err == nil {
		snap.SiteName = site.SiteName
	}
	return snap, nil
}

func (c *LocalClient) SetIslandMode(mode string) error {
	if c.IP == "" || c.Password == "" {
		return fmt.Errorf("off-grid scheduling requires local Gateway IP and password")
	}
	if err := c.login(); err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"island_mode": mode})
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("https://%s/api/v2/islanding/mode", c.IP)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("island mode update failed: %s", resp.Status)
	}
	return nil
}

func FetchWallConnectorVitals(ip string) (*WallConnectorVitals, error) {
	if ip == "" {
		return nil, nil
	}
	client := &http.Client{Timeout: 9 * time.Second}
	url := fmt.Sprintf("http://%s/api/1/vitals", ip)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Wall Connector vitals failed: %s", resp.Status)
	}
	var vitals WallConnectorVitals
	if err := json.NewDecoder(resp.Body).Decode(&vitals); err != nil {
		return nil, err
	}
	return &vitals, nil
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

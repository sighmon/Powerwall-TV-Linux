package schedule

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/auth"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

type Mode string

const (
	SelfPowered      Mode = "selfPowered"
	TimeBasedControl Mode = "timeBasedControl"
	OffGrid          Mode = "offGrid"
	OnGrid           Mode = "onGrid"
)

var Modes = []Mode{SelfPowered, TimeBasedControl, OffGrid, OnGrid}

func (m Mode) Title() string {
	switch m {
	case SelfPowered:
		return "Self-Powered"
	case TimeBasedControl:
		return "Time-Based Control"
	case OffGrid:
		return "Off-Grid"
	case OnGrid:
		return "On-Grid"
	default:
		return string(m)
	}
}

type Schedule struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Enabled          bool   `json:"isEnabled"`
	EnergySiteID     int    `json:"energySiteId"`
	EnergySiteName   string `json:"energySiteName"`
	StartMinutes     int    `json:"startMinutes"`
	EndMinutes       int    `json:"endMinutes"`
	StartMode        Mode   `json:"startMode"`
	EndMode          Mode   `json:"endMode"`
	LastAppliedStart string `json:"lastAppliedStart,omitempty"`
	LastAppliedEnd   string `json:"lastAppliedEnd,omitempty"`
}

func New() Schedule {
	return Schedule{ID: randomID(), Name: "Peak export", StartMinutes: 15 * 60, EndMinutes: 21 * 60, StartMode: TimeBasedControl, EndMode: SelfPowered}
}

type Store struct {
	Enabled       bool       `json:"enabled"`
	LastRunStatus string     `json:"lastRunStatus"`
	Schedules     []Schedule `json:"schedules"`
}

type Manager struct {
	mu      sync.Mutex
	store   Store
	path    string
	running bool
}

func Load() *Manager {
	manager := &Manager{store: Store{LastRunStatus: "No schedule has run yet"}}
	if dir, err := os.UserConfigDir(); err == nil {
		manager.path = filepath.Join(dir, "powerwall-tv", "schedules.json")
		if data, err := os.ReadFile(manager.path); err == nil {
			_ = json.Unmarshal(data, &manager.store)
		}
	}
	return manager
}

func (m *Manager) Snapshot() Store {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := m.store
	copy.Schedules = append([]Schedule(nil), m.store.Schedules...)
	return copy
}

func (m *Manager) Replace(store Store, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := make(map[string]Schedule, len(m.store.Schedules))
	for _, item := range m.store.Schedules {
		previous[item.ID] = item
	}
	for i := range store.Schedules {
		item := &store.Schedules[i]
		if item.ID == "" {
			item.ID = randomID()
		}
		old, existed := previous[item.ID]
		changedWhileEnabled := existed && item.Enabled && (item.StartMinutes != old.StartMinutes ||
			item.EndMinutes != old.EndMinutes ||
			item.StartMode != old.StartMode ||
			item.EndMode != old.EndMode ||
			item.EnergySiteID != old.EnergySiteID)
		if item.Enabled && item.EnergySiteID != 0 && (!existed || !old.Enabled || changedWhileEnabled || (store.Enabled && !m.store.Enabled)) {
			item.LastAppliedStart = boundaryKey(item.StartMinutes, now)
			item.LastAppliedEnd = boundaryKey(item.EndMinutes, now)
		}
		if item.EnergySiteID == 0 {
			item.Enabled = false
		}
	}
	if store.LastRunStatus == "" {
		store.LastRunStatus = m.store.LastRunStatus
	}
	m.store = store
	return m.saveLocked()
}

type dueBoundary struct {
	index    int
	start    bool
	due      time.Time
	mode     Mode
	applyKey string
}

func dueBoundaries(store Store, now time.Time) []dueBoundary {
	if !store.Enabled {
		return nil
	}
	var due []dueBoundary
	for index, item := range store.Schedules {
		if !item.Enabled || item.EnergySiteID == 0 {
			continue
		}
		for _, boundary := range []struct {
			start   bool
			minutes int
			mode    Mode
			last    string
		}{{true, item.StartMinutes, item.StartMode, item.LastAppliedStart}, {false, item.EndMinutes, item.EndMode, item.LastAppliedEnd}} {
			dueAt := mostRecentTime(boundary.minutes, now)
			key := dueAt.Format("2006-01-02")
			if boundary.last != key {
				due = append(due, dueBoundary{index: index, start: boundary.start, due: dueAt, mode: boundary.mode, applyKey: key})
			}
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i].due.Before(due[j].due) })
	return due
}

func (m *Manager) ApplyDue(state *statepkg.State, currentSiteID int, now time.Time) string {
	m.mu.Lock()
	if m.running {
		status := m.store.LastRunStatus
		m.mu.Unlock()
		return status
	}
	m.running = true
	due := dueBoundaries(m.store, now)
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()
	for _, boundary := range due {
		m.mu.Lock()
		item := m.store.Schedules[boundary.index]
		m.mu.Unlock()
		err := executeWithRetry(func() error {
			return execute(state, currentSiteID, item, boundary.mode)
		}, time.Sleep, 3)
		m.mu.Lock()
		if err != nil {
			m.store.LastRunStatus = fmt.Sprintf("Failed %s: %v", item.Name, err)
			_ = m.saveLocked()
			status := m.store.LastRunStatus
			m.mu.Unlock()
			return status
		}
		if boundary.start {
			m.store.Schedules[boundary.index].LastAppliedStart = boundary.applyKey
		} else {
			m.store.Schedules[boundary.index].LastAppliedEnd = boundary.applyKey
		}
		kind := "end"
		if boundary.start {
			kind = "start"
		}
		m.store.LastRunStatus = fmt.Sprintf("%s at %s (%d) %s: %s", item.Name, item.EnergySiteName, item.EnergySiteID, kind, boundary.mode.Title())
		_ = m.saveLocked()
		m.mu.Unlock()
	}
	return m.Snapshot().LastRunStatus
}

func executeWithRetry(attempt func() error, wait func(time.Duration), retries int) error {
	err := attempt()
	for retry := 0; err != nil && retry < retries; retry++ {
		var statusErr *api.HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == 401 || statusErr.StatusCode == 403) {
			return fmt.Errorf("re-login to grant Powerwall command access for scheduled mode changes: %w", err)
		}
		// Match the source's capped exponential command retry: 1, 2, 4s.
		wait(time.Second << retry)
		err = attempt()
	}
	return err
}

func execute(state *statepkg.State, currentSiteID int, item Schedule, mode Mode) error {
	if mode == OffGrid || mode == OnGrid {
		if currentSiteID != item.EnergySiteID {
			return fmt.Errorf("select %s before running an off-grid or on-grid schedule", item.EnergySiteName)
		}
		password, _ := storage.GetGatewayPassword()
		value := "backup"
		if mode == OffGrid {
			value = "intentional_reconnect_failsafe"
		}
		return api.NewLocalClient(state.Prefs.GatewayIP, state.Prefs.Username, password).SetIslandMode(value)
	}
	token, err := auth.LoadStoredToken()
	if err != nil {
		return err
	}
	value := "self_consumption"
	if mode == TimeBasedControl {
		value = "autonomous"
	}
	return api.NewFleetClient(state.Prefs.FleetBaseURL, token.AccessToken).SetOperationMode(item.EnergySiteID, value)
}

func (m *Manager) saveLocked() error {
	if m.path == "" {
		return fmt.Errorf("configuration directory unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0o600)
}

func mostRecentTime(minutes int, now time.Time) time.Time {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	candidate := start.Add(time.Duration(minutes) * time.Minute)
	if candidate.After(now) {
		candidate = candidate.AddDate(0, 0, -1)
	}
	return candidate
}

func boundaryKey(minutes int, now time.Time) string {
	return mostRecentTime(minutes, now).Format("2006-01-02")
}

func randomID() string {
	var bytes [16]byte
	_, _ = rand.Read(bytes[:])
	return hex.EncodeToString(bytes[:])
}

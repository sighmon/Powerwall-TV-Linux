package schedule

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"powerwall-tv-gtk/internal/api"
)

func TestDueBoundariesAreOrderedAndAppliedOnce(t *testing.T) {
	location := time.FixedZone("ACST", 9*60*60+30*60)
	now := time.Date(2026, 9, 5, 22, 0, 0, 0, location)
	store := Store{Enabled: true, Schedules: []Schedule{{
		ID: "one", Name: "Peak", Enabled: true, EnergySiteID: 42,
		StartMinutes: 15 * 60, EndMinutes: 21 * 60,
		StartMode: TimeBasedControl, EndMode: SelfPowered,
	}}}
	due := dueBoundaries(store, now)
	if len(due) != 2 || !due[0].start || due[1].start {
		t.Fatalf("unexpected due boundaries: %#v", due)
	}
	store.Schedules[0].LastAppliedStart = "2026-09-05"
	store.Schedules[0].LastAppliedEnd = "2026-09-05"
	if got := dueBoundaries(store, now); len(got) != 0 {
		t.Fatalf("already applied boundaries returned: %#v", got)
	}
}

func TestReplaceMarksEditedEnabledScheduleCurrent(t *testing.T) {
	now := time.Date(2026, 9, 5, 22, 0, 0, 0, time.UTC)
	old := Schedule{ID: "one", Enabled: true, EnergySiteID: 42, StartMinutes: 15 * 60, EndMinutes: 21 * 60, StartMode: TimeBasedControl, EndMode: SelfPowered}
	manager := &Manager{path: filepath.Join(t.TempDir(), "schedules.json"), store: Store{Enabled: true, Schedules: []Schedule{old}}}
	edited := old
	edited.StartMinutes++
	if err := manager.Replace(Store{Enabled: true, Schedules: []Schedule{edited}}, now); err != nil {
		t.Fatal(err)
	}
	got := manager.Snapshot().Schedules[0]
	if got.LastAppliedStart != "2026-09-05" || got.LastAppliedEnd != "2026-09-05" {
		t.Fatalf("edited schedule boundaries = %q, %q", got.LastAppliedStart, got.LastAppliedEnd)
	}
}

func TestOvernightBoundaryUsesYesterday(t *testing.T) {
	now := time.Date(2026, 9, 5, 1, 0, 0, 0, time.UTC)
	if got := mostRecentTime(21*60, now); got.Day() != 4 || got.Hour() != 21 {
		t.Fatalf("mostRecentTime = %v", got)
	}
}

func TestExecuteWithRetryUsesExponentialDelay(t *testing.T) {
	attempts := 0
	var delays []time.Duration
	err := executeWithRetry(func() error {
		attempts++
		if attempts < 4 {
			return errTemporary
		}
		return nil
	}, func(delay time.Duration) {
		delays = append(delays, delay)
	}, 3)
	if err != nil {
		t.Fatalf("executeWithRetry returned %v", err)
	}
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4", attempts)
	}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	for i := range want {
		if delays[i] != want[i] {
			t.Fatalf("delays = %v, want %v", delays, want)
		}
	}
}

func TestExecuteWithRetryStopsOnFleetPermissionError(t *testing.T) {
	attempts := 0
	waits := 0
	err := executeWithRetry(func() error {
		attempts++
		return &api.HTTPStatusError{StatusCode: 403, Status: "403 Forbidden"}
	}, func(time.Duration) { waits++ }, 3)
	if attempts != 1 || waits != 0 {
		t.Fatalf("attempts = %d, waits = %d; want 1, 0", attempts, waits)
	}
	var statusErr *api.HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != 403 {
		t.Fatalf("error = %#v, want wrapped HTTPStatusError", err)
	}
}

var errTemporary = &temporaryError{}

type temporaryError struct{}

func (*temporaryError) Error() string { return "temporary" }

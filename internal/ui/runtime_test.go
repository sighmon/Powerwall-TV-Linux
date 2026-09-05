package ui

import (
	"testing"
	"time"
)

func TestRuntimeEstimateString(t *testing.T) {
	tests := []struct {
		watts, count, percent, reserve float64
		want                           string
	}{
		{-100, 1, 80, 0, "Optimised charging"},
		{-1350, 1, 90, 0, "1 hour to 100%"},
		{1350, 1, 20, 10, "1 hour to 10%"},
		{135, 2, 50, 0, "4 days 4 hours to 0%"},
		{0, 1, 50, 0, ""},
		{1000, 0, 50, 0, ""},
	}
	for _, tt := range tests {
		if got := runtimeEstimateString(tt.watts, tt.count, tt.percent, tt.reserve, 40); got != tt.want {
			t.Errorf("runtimeEstimateString(%v, %v, %v, %v) = %q, want %q", tt.watts, tt.count, tt.percent, tt.reserve, got, tt.want)
		}
	}
}

func TestRuntimeEstimatorResetsOnDirectionChange(t *testing.T) {
	var estimator runtimeEstimator
	now := time.Now()
	estimator.record(1000, now.Add(-time.Minute))
	estimator.record(2000, now)
	if got := estimator.average(2000, 40, now); got != 1500 {
		t.Fatalf("average = %v, want 1500", got)
	}
	estimator.record(-500, now)
	if got := estimator.average(-500, 40, now); got != -500 {
		t.Fatalf("average after direction change = %v, want -500", got)
	}
}

package ui

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	batteryCapacityWh = 13_500.0
	runtimeWindow     = 5 * time.Minute
)

type batteryPowerSample struct {
	at    time.Time
	watts float64
}

type runtimeEstimator struct {
	mu      sync.Mutex
	samples []batteryPowerSample
}

func powerDirection(watts float64) int {
	if watts > 0 {
		return 1
	}
	if watts < 0 {
		return -1
	}
	return 0
}

func (e *runtimeEstimator) record(watts float64, at time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	direction := powerDirection(watts)
	if len(e.samples) > 0 && direction != 0 && powerDirection(e.samples[len(e.samples)-1].watts) != 0 && direction != powerDirection(e.samples[len(e.samples)-1].watts) {
		e.samples = nil
	}
	e.samples = append(e.samples, batteryPowerSample{at: at, watts: watts})
	e.prune(at)
}

func (e *runtimeEstimator) average(currentWatts, idleThreshold float64, now time.Time) float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.prune(now)
	direction := powerDirection(currentWatts)
	if direction == 0 || math.Abs(currentWatts) <= idleThreshold {
		return currentWatts
	}
	total := 0.0
	count := 0
	for _, sample := range e.samples {
		if powerDirection(sample.watts) == direction && math.Abs(sample.watts) > idleThreshold {
			total += sample.watts
			count++
		}
	}
	if count == 0 {
		return currentWatts
	}
	return total / float64(count)
}

func (e *runtimeEstimator) prune(now time.Time) {
	cutoff := now.Add(-runtimeWindow)
	first := 0
	for first < len(e.samples) && e.samples[first].at.Before(cutoff) {
		first++
	}
	e.samples = e.samples[first:]
}

func runtimeEstimateString(batteryWatts, batteryCount, percentage, reserve, idleThreshold float64) string {
	if percentage >= 79 && percentage <= 81 && batteryWatts <= 0 && -batteryWatts < 250 {
		return "Optimised charging"
	}
	if batteryCount <= 0 || percentage <= 0 || percentage >= 100 || math.Abs(batteryWatts) <= idleThreshold {
		return ""
	}
	reserve = clamp(reserve, 0, 100)
	target := int(math.Round(reserve))
	remainingPercent := percentage - reserve
	if batteryWatts < 0 {
		target = 100
		remainingPercent = 100 - percentage
	}
	remainingWh := batteryCapacityWh * batteryCount * remainingPercent / 100
	if remainingWh <= 0 {
		return ""
	}
	minutes := int(math.Round(remainingWh / math.Abs(batteryWatts) * 60))
	if minutes <= 0 {
		return ""
	}
	hours, minuteRemainder := minutes/60, minutes%60
	var duration string
	if hours >= 24 {
		days, hourRemainder := hours/24, hours%24
		duration = plural(days, "day")
		if hourRemainder > 0 {
			duration += " " + plural(hourRemainder, "hour")
		}
	} else if hours == 0 {
		duration = plural(minuteRemainder, "minute")
	} else {
		duration = plural(hours, "hour")
		if minuteRemainder > 0 {
			duration += " " + plural(minuteRemainder, "minute")
		}
	}
	return fmt.Sprintf("%s to %d%%", duration, target)
}

func plural(value int, unit string) string {
	if value == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", value, unit)
}

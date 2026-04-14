package detector

import (
	"github.com/chaso/ai-usage-monitor/internal/usage"
)

// EventType identifies what kind of event occurred.
type EventType string

const (
	FiveHourReset EventType = "five_hour_reset"
	WeeklyReset   EventType = "weekly_reset"
	FiveHourLow   EventType = "five_hour_low"
	WeeklyLow     EventType = "weekly_low"
)

// ResetEvent describes a single detected event.
type ResetEvent struct {
	Provider  string
	EventType EventType
	Prev      usage.WindowUsage
	Curr      usage.WindowUsage
}

const (
	resetDropThreshold = 20.0
	lowThreshold       = 95.0 // remaining < 5% when used_percent >= 95
)

// Detect compares two ProviderUsage snapshots and returns any events found.
// Events include resets (usage became available) and low-usage warnings (< 5% remaining).
// prev may be zero-valued on the first run; in that case no events are emitted.
func Detect(provider string, prev, curr usage.ProviderUsage) []ResetEvent {
	var events []ResetEvent

	if isReset(prev.FiveHour, curr.FiveHour) {
		events = append(events, ResetEvent{
			Provider:  provider,
			EventType: FiveHourReset,
			Prev:      prev.FiveHour,
			Curr:      curr.FiveHour,
		})
	}

	if isReset(prev.Weekly, curr.Weekly) {
		events = append(events, ResetEvent{
			Provider:  provider,
			EventType: WeeklyReset,
			Prev:      prev.Weekly,
			Curr:      curr.Weekly,
		})
	}

	if crossedLowThreshold(prev.FiveHour, curr.FiveHour) {
		events = append(events, ResetEvent{
			Provider:  provider,
			EventType: FiveHourLow,
			Prev:      prev.FiveHour,
			Curr:      curr.FiveHour,
		})
	}

	if crossedLowThreshold(prev.Weekly, curr.Weekly) {
		events = append(events, ResetEvent{
			Provider:  provider,
			EventType: WeeklyLow,
			Prev:      prev.Weekly,
			Curr:      curr.Weekly,
		})
	}

	return events
}

// isReset returns true when a window has reset between two observations.
func isReset(prev, curr usage.WindowUsage) bool {
	if prev.ResetAt.IsZero() {
		return false
	}
	if prev.UsedPercent-curr.UsedPercent >= resetDropThreshold {
		return true
	}
	if !curr.ResetAt.IsZero() && curr.ResetAt.After(prev.ResetAt) {
		return true
	}
	return false
}

// crossedLowThreshold returns true when used_percent crossed the 95% boundary
// upward (remaining dropped below 5%) since the last observation.
func crossedLowThreshold(prev, curr usage.WindowUsage) bool {
	if prev.ResetAt.IsZero() {
		return false
	}
	return prev.UsedPercent < lowThreshold && curr.UsedPercent >= lowThreshold
}

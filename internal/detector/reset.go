package detector

import (
	"github.com/chaso/ai-usage-monitor/internal/usage"
)

// EventType identifies what kind of reset occurred.
type EventType string

const (
	FiveHourReset EventType = "five_hour_reset"
	WeeklyReset   EventType = "weekly_reset"
)

// ResetEvent describes a single detected reset.
type ResetEvent struct {
	Provider  string
	EventType EventType
	// Previous and current window snapshots for notification context.
	Prev usage.WindowUsage
	Curr usage.WindowUsage
}

const resetDropThreshold = 20.0

// Detect compares two ProviderUsage snapshots and returns any resets found.
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

	return events
}

// isReset returns true when a window has reset between two observations.
// Two signals are checked independently:
//  1. Usage percent dropped by >= resetDropThreshold points.
//  2. The reset timestamp advanced to a strictly later value.
func isReset(prev, curr usage.WindowUsage) bool {
	// Skip the first run where prev is zero.
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

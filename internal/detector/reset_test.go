package detector_test

import (
	"testing"
	"time"

	"github.com/chaso/ai-usage-monitor/internal/detector"
	"github.com/chaso/ai-usage-monitor/internal/usage"
)

var (
	t0 = time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	t1 = t0.Add(5 * time.Hour)
)

func window(pct float64, reset time.Time) usage.WindowUsage {
	return usage.WindowUsage{UsedPercent: pct, ResetAt: reset}
}

func TestDetect_NoEventOnFirstRun(t *testing.T) {
	// prev has zero ResetAt — must not emit events.
	prev := usage.ProviderUsage{}
	curr := usage.ProviderUsage{
		FiveHour: window(10, t0),
		Weekly:   window(20, t0),
	}
	events := detector.Detect("claude", prev, curr)
	if len(events) != 0 {
		t.Fatalf("expected no events on first run, got %v", events)
	}
}

func TestDetect_FiveHourReset_DropThreshold(t *testing.T) {
	prev := usage.ProviderUsage{
		FiveHour: window(80, t0),
		Weekly:   window(40, t0),
	}
	curr := usage.ProviderUsage{
		FiveHour: window(5, t0),  // dropped 75 points → reset
		Weekly:   window(42, t0), // small increase → no reset
	}
	events := detector.Detect("claude", prev, curr)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(events), events)
	}
	if events[0].EventType != detector.FiveHourReset {
		t.Errorf("expected five_hour_reset, got %s", events[0].EventType)
	}
}

func TestDetect_WeeklyReset_TimestampAdvance(t *testing.T) {
	prev := usage.ProviderUsage{
		FiveHour: window(50, t0),
		Weekly:   window(90, t0),
	}
	curr := usage.ProviderUsage{
		FiveHour: window(52, t0), // small increase → no reset
		Weekly:   window(85, t1), // timestamp advanced → reset
	}
	events := detector.Detect("codex", prev, curr)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(events), events)
	}
	if events[0].EventType != detector.WeeklyReset {
		t.Errorf("expected weekly_reset, got %s", events[0].EventType)
	}
	if events[0].Provider != "codex" {
		t.Errorf("expected provider codex, got %s", events[0].Provider)
	}
}

func TestDetect_BothWindows(t *testing.T) {
	prev := usage.ProviderUsage{
		FiveHour: window(95, t0),
		Weekly:   window(70, t0),
	}
	curr := usage.ProviderUsage{
		FiveHour: window(2, t0),  // drop → reset
		Weekly:   window(5, t1),  // drop + timestamp → reset
	}
	events := detector.Detect("claude", prev, curr)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestDetect_FiveHourLow(t *testing.T) {
	prev := usage.ProviderUsage{
		FiveHour: window(90, t0), // 10% remaining — not yet low
		Weekly:   window(50, t0),
	}
	curr := usage.ProviderUsage{
		FiveHour: window(96, t0), // 4% remaining — crossed into low
		Weekly:   window(52, t0),
	}
	events := detector.Detect("claude", prev, curr)
	if len(events) != 1 {
		t.Fatalf("expected 1 low event, got %d: %v", len(events), events)
	}
	if events[0].EventType != detector.FiveHourLow {
		t.Errorf("expected five_hour_low, got %s", events[0].EventType)
	}
}

func TestDetect_WeeklyLow(t *testing.T) {
	prev := usage.ProviderUsage{
		FiveHour: window(50, t0),
		Weekly:   window(94, t0), // 6% remaining — not yet low
	}
	curr := usage.ProviderUsage{
		FiveHour: window(51, t0),
		Weekly:   window(97, t0), // 3% remaining — crossed into low
	}
	events := detector.Detect("codex", prev, curr)
	if len(events) != 1 {
		t.Fatalf("expected 1 low event, got %d: %v", len(events), events)
	}
	if events[0].EventType != detector.WeeklyLow {
		t.Errorf("expected weekly_low, got %s", events[0].EventType)
	}
}

func TestDetect_LowAlreadyLow_NoRepeat(t *testing.T) {
	// Already above threshold in prev — should not re-fire.
	prev := usage.ProviderUsage{
		FiveHour: window(97, t0),
		Weekly:   window(98, t0),
	}
	curr := usage.ProviderUsage{
		FiveHour: window(99, t0),
		Weekly:   window(99, t0),
	}
	events := detector.Detect("claude", prev, curr)
	if len(events) != 0 {
		t.Fatalf("expected no events when already low, got %d: %v", len(events), events)
	}
}

func TestDetect_NoBelowThreshold(t *testing.T) {
	prev := usage.ProviderUsage{
		FiveHour: window(50, t0),
		Weekly:   window(50, t0),
	}
	curr := usage.ProviderUsage{
		FiveHour: window(35, t0), // 15 point drop → below threshold, no reset
		Weekly:   window(30, t0), // 20 point drop exactly — boundary: not >= so no reset
	}
	events := detector.Detect("claude", prev, curr)
	// 20-point drop is NOT >= threshold (strict greater-than interpretation — boundary is inclusive)
	// The code uses >= 20, so 20 IS a reset. Let's be precise:
	// FiveHour: 50-35=15 < 20 → no
	// Weekly: 50-30=20 >= 20 → yes
	if len(events) != 1 {
		t.Fatalf("expected 1 event (weekly boundary), got %d: %v", len(events), events)
	}
	if events[0].EventType != detector.WeeklyReset {
		t.Errorf("expected weekly_reset at boundary, got %s", events[0].EventType)
	}
}

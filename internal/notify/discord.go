package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chaso/ai-usage-monitor/internal/detector"
)

// Notifier sends Discord webhook messages.
type Notifier struct {
	webhookURL string
	client     *http.Client
}

func NewDiscord(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

type discordPayload struct {
	Content string `json:"content"`
}

// Send dispatches a Discord message for the given event.
// It is a no-op when the webhook URL is empty.
func (n *Notifier) Send(ctx context.Context, event detector.ResetEvent) error {
	if n.webhookURL == "" {
		return nil
	}

	content := buildMessage(event)

	payload, err := json.Marshal(discordPayload{Content: content})
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("discord: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func buildMessage(event detector.ResetEvent) string {
	provider := titleCase(event.Provider)

	switch event.EventType {
	case detector.FiveHourReset:
		return fmt.Sprintf("♻️ %s 5h window reset — go go go", provider)
	case detector.WeeklyReset:
		return fmt.Sprintf("♻️ %s weekly window reset — go go go", provider)
	case detector.FiveHourLow:
		return fmt.Sprintf("☕️ %s 5h window below 5%% — take a coffee break", provider)
	case detector.WeeklyLow:
		return fmt.Sprintf("✈️ %s weekly window below 5%% — time for a trip", provider)
	default:
		return fmt.Sprintf("%s: %s", provider, event.EventType)
	}
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

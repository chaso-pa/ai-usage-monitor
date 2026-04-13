package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/chaso/ai-usage-monitor/internal/usage"
)

const (
	defaultEndpoint = "https://api.claude.ai/api/usage"
	envTokenKey     = "CLAUDE_CODE_OAUTH_TOKEN"
)

// HTTPClient is a narrow interface for dependency injection in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Provider fetches Claude Code usage.
type Provider struct {
	tokenEnv string
	endpoint string
	client   HTTPClient
}

// Option configures the Provider.
type Option func(*Provider)

func WithTokenEnv(env string) Option {
	return func(p *Provider) { p.tokenEnv = env }
}

func WithEndpoint(url string) Option {
	return func(p *Provider) { p.endpoint = url }
}

func WithHTTPClient(c HTTPClient) Option {
	return func(p *Provider) { p.client = c }
}

func New(opts ...Option) *Provider {
	p := &Provider{
		tokenEnv: envTokenKey,
		endpoint: defaultEndpoint,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Provider) Name() string { return "claude" }

// apiResponse is the expected shape from the Claude usage endpoint.
// Adjust field names when the real API is available.
type apiResponse struct {
	FiveHour struct {
		UsedPercent float64 `json:"used_percent"`
		ResetAt     string  `json:"reset_at"`
	} `json:"five_hour"`
	Weekly struct {
		UsedPercent float64 `json:"used_percent"`
		ResetAt     string  `json:"reset_at"`
	} `json:"weekly"`
}

func (p *Provider) Fetch(ctx context.Context) (usage.ProviderUsage, error) {
	token := os.Getenv(p.tokenEnv)
	if token == "" {
		return p.mockUsage(), nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("claude: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		// Network error — return mock so the daemon stays alive.
		return p.mockUsage(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.mockUsage(), nil
	}

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("claude: decode response: %w", err)
	}

	fiveHourReset, _ := time.Parse(time.RFC3339, ar.FiveHour.ResetAt)
	weeklyReset, _ := time.Parse(time.RFC3339, ar.Weekly.ResetAt)

	return usage.ProviderUsage{
		FiveHour: usage.WindowUsage{
			UsedPercent: ar.FiveHour.UsedPercent,
			ResetAt:     fiveHourReset,
		},
		Weekly: usage.WindowUsage{
			UsedPercent: ar.Weekly.UsedPercent,
			ResetAt:     weeklyReset,
		},
	}, nil
}

// mockUsage returns a zeroed usage so the daemon has something to work with
// when the endpoint is unavailable or the token is missing.
func (p *Provider) mockUsage() usage.ProviderUsage {
	now := time.Now().UTC()
	return usage.ProviderUsage{
		FiveHour: usage.WindowUsage{UsedPercent: 0, ResetAt: now.Add(5 * time.Hour)},
		Weekly:   usage.WindowUsage{UsedPercent: 0, ResetAt: now.Add(7 * 24 * time.Hour)},
	}
}

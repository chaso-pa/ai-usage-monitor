package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chaso/ai-usage-monitor/internal/usage"
)

const (
	defaultEndpoint = "https://api.openai.com/v1/usage/codex"
	envTokenKey     = "OPENAI_SESSION_TOKEN"
)

// HTTPClient is a narrow interface for dependency injection in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Provider fetches Codex usage, trying in order:
//  1. Remote API (session-cookie strategy)
//  2. Local file ~/.codex/usage.json
//  3. Mock zeroed usage
type Provider struct {
	tokenEnv  string
	endpoint  string
	localFile string
	client    HTTPClient
}

type Option func(*Provider)

func WithTokenEnv(env string) Option {
	return func(p *Provider) { p.tokenEnv = env }
}

func WithEndpoint(url string) Option {
	return func(p *Provider) { p.endpoint = url }
}

func WithLocalFile(path string) Option {
	return func(p *Provider) { p.localFile = path }
}

func WithHTTPClient(c HTTPClient) Option {
	return func(p *Provider) { p.client = c }
}

func New(opts ...Option) *Provider {
	home, _ := os.UserHomeDir()
	p := &Provider{
		tokenEnv:  envTokenKey,
		endpoint:  defaultEndpoint,
		localFile: filepath.Join(home, ".codex", "usage.json"),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Provider) Name() string { return "codex" }

// localFile schema — adapt when the real format is known.
type localUsageFile struct {
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
	if token != "" {
		if u, err := p.fetchRemote(ctx, token); err == nil {
			return u, nil
		}
	}

	if u, err := p.fetchLocal(); err == nil {
		return u, nil
	}

	return p.mockUsage(), nil
}

func (p *Provider) fetchRemote(ctx context.Context, token string) (usage.ProviderUsage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("codex: build request: %w", err)
	}
	req.Header.Set("Cookie", "session="+token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("codex: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return usage.ProviderUsage{}, fmt.Errorf("codex: unexpected status %d", resp.StatusCode)
	}

	var lf localUsageFile
	if err := json.NewDecoder(resp.Body).Decode(&lf); err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("codex: decode: %w", err)
	}
	return parseLocalFile(lf), nil
}

func (p *Provider) fetchLocal() (usage.ProviderUsage, error) {
	data, err := os.ReadFile(p.localFile)
	if err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("codex: read local file: %w", err)
	}
	var lf localUsageFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return usage.ProviderUsage{}, fmt.Errorf("codex: parse local file: %w", err)
	}
	return parseLocalFile(lf), nil
}

func parseLocalFile(lf localUsageFile) usage.ProviderUsage {
	fiveHourReset, _ := time.Parse(time.RFC3339, lf.FiveHour.ResetAt)
	weeklyReset, _ := time.Parse(time.RFC3339, lf.Weekly.ResetAt)
	return usage.ProviderUsage{
		FiveHour: usage.WindowUsage{UsedPercent: lf.FiveHour.UsedPercent, ResetAt: fiveHourReset},
		Weekly:   usage.WindowUsage{UsedPercent: lf.Weekly.UsedPercent, ResetAt: weeklyReset},
	}
}

func (p *Provider) mockUsage() usage.ProviderUsage {
	now := time.Now().UTC()
	return usage.ProviderUsage{
		FiveHour: usage.WindowUsage{UsedPercent: 0, ResetAt: now.Add(5 * time.Hour)},
		Weekly:   usage.WindowUsage{UsedPercent: 0, ResetAt: now.Add(7 * 24 * time.Hour)},
	}
}

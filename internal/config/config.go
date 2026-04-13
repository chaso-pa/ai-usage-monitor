package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	TokenEnv string `yaml:"token_env"`
}

type ProvidersConfig struct {
	Claude ProviderConfig `yaml:"claude"`
	Codex  ProviderConfig `yaml:"codex"`
}

type Config struct {
	PollInterval   time.Duration   `yaml:"poll_interval"`
	DiscordWebhook string          `yaml:"discord_webhook"`
	CachePath      string          `yaml:"cache_path"`
	Providers      ProvidersConfig `yaml:"providers"`
}

// Load reads a YAML config file and returns a Config.
// Environment variable references in string values are expanded.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.CachePath = os.ExpandEnv(cfg.CachePath)

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Minute
	}
	if cfg.CachePath == "" {
		home, _ := os.UserHomeDir()
		cfg.CachePath = home + "/.cache/ai-usage.json"
	}

	return &cfg, nil
}

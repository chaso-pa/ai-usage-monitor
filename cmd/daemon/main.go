package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chaso/ai-usage-monitor/internal/cache"
	"github.com/chaso/ai-usage-monitor/internal/claude"
	"github.com/chaso/ai-usage-monitor/internal/codex"
	"github.com/chaso/ai-usage-monitor/internal/config"
	"github.com/chaso/ai-usage-monitor/internal/detector"
	"github.com/chaso/ai-usage-monitor/internal/notify"
	"github.com/chaso/ai-usage-monitor/internal/usage"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	providers := []usage.Provider{
		claude.New(claude.WithTokenEnv(cfg.Providers.Claude.TokenEnv)),
		codex.New(codex.WithTokenEnv(cfg.Providers.Codex.TokenEnv)),
	}

	store := cache.New(cfg.CachePath)
	notifier := notify.NewDiscord(cfg.DiscordWebhook)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("daemon started", "poll_interval", cfg.PollInterval, "cache_path", cfg.CachePath)

	// Run once immediately, then on each tick.
	poll(ctx, logger, providers, store, notifier)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			poll(ctx, logger, providers, store, notifier)
		case <-ctx.Done():
			logger.Info("daemon shutting down")
			return
		}
	}
}

func poll(
	ctx context.Context,
	logger *slog.Logger,
	providers []usage.Provider,
	store *cache.Store,
	notifier *notify.Notifier,
) {
	prev := store.Previous()

	snap := usage.Snapshot{UpdatedAt: time.Now().UTC()}

	for _, p := range providers {
		u, err := p.Fetch(ctx)
		if err != nil {
			logger.Warn("fetch failed", "provider", p.Name(), "error", err)
			continue
		}

		switch p.Name() {
		case "claude":
			snap.Claude = u
		case "codex":
			snap.Codex = u
		}

		logger.Info("fetched usage",
			"provider", p.Name(),
			"5h_used_pct", u.FiveHour.UsedPercent,
			"weekly_used_pct", u.Weekly.UsedPercent,
		)
	}

	if prev != nil {
		detectAndNotify(ctx, logger, notifier, "claude", prev.Claude, snap.Claude)
		detectAndNotify(ctx, logger, notifier, "codex", prev.Codex, snap.Codex)
	}

	if err := store.Write(snap); err != nil {
		logger.Error("cache write failed", "error", err)
	}
}

func detectAndNotify(
	ctx context.Context,
	logger *slog.Logger,
	notifier *notify.Notifier,
	provider string,
	prev, curr usage.ProviderUsage,
) {
	events := detector.Detect(provider, prev, curr)
	for _, ev := range events {
		logger.Info("reset detected", "provider", ev.Provider, "event", ev.EventType)
		if err := notifier.Send(ctx, ev); err != nil {
			logger.Warn("discord notify failed", "error", err)
		}
	}
}

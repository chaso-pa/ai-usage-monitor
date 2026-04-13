# ai-usage-monitor

A production-ready daemon that continuously monitors Claude Code and Codex AI usage, detects quota resets, sends Discord notifications, and exposes a compact tmux status-right formatter.

---

## Features

- Polls Claude Code and Codex usage every 5 minutes (configurable)
- Tracks both the rolling **5-hour** and **weekly** usage windows
- Detects quota resets via usage-percent drop or reset-timestamp advancement
- Sends Discord webhook notifications on reset with remaining capacity and next-reset time (JST)
- Writes an atomic local cache (`~/.cache/ai-usage.json`) for fast tmux reads
- `tmuxfmt` CLI outputs a compact `CL:28/59 CD:44/80` string for `status-right`

---

## Project Layout

```
ai-usage-monitor/
  cmd/
    daemon/main.go      # long-running poller
    tmuxfmt/main.go     # one-shot tmux formatter
  internal/
    usage/              # shared data model + Provider interface
    claude/             # Claude Code usage provider
    codex/              # Codex usage provider (remote + local file fallback)
    detector/           # reset detection logic
    notify/             # Discord webhook notifier
    cache/              # atomic JSON cache (read/write)
    config/             # YAML config loader
  configs/
    config.yaml         # default configuration
  scripts/
    install_tmux.sh     # appends status-right to ~/.tmux.conf
  Makefile
```

---

## Environment Variables

| Variable | Provider | Purpose |
|---|---|---|
| `CLAUDE_CODE_OAUTH_TOKEN` | Claude | Bearer token for the Claude usage API |
| `OPENAI_SESSION_TOKEN` | Codex | Session cookie for the Codex usage API |

Both are optional — without them, the respective provider returns a zeroed mock response so the daemon keeps running.

---

## Installation

### 1. Clone and build

```bash
git clone https://github.com/chaso/ai-usage-monitor
cd ai-usage-monitor
make build          # produces bin/daemon and bin/tmuxfmt
```

### 2. Install binaries to ~/ops

```bash
make install        # copies binaries to ~/ops/ai-usage-monitor/bin/
```

### 3. Configure

Edit `configs/config.yaml`:

```yaml
poll_interval: 5m
discord_webhook: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_TOKEN"
cache_path: ~/.cache/ai-usage.json

providers:
  claude:
    token_env: CLAUDE_CODE_OAUTH_TOKEN
  codex:
    token_env: OPENAI_SESSION_TOKEN
```

### 4. Set environment variables

```bash
export CLAUDE_CODE_OAUTH_TOKEN="your-claude-token"
export OPENAI_SESSION_TOKEN="your-openai-session-cookie"
```

---

## Running the Daemon

```bash
# Foreground (development)
make run-daemon

# Background (production)
nohup ./bin/daemon -config configs/config.yaml > daemon.log 2>&1 &

# Or via systemd — see extension points below
```

---

## tmux Setup

```bash
make tmux-setup     # appends status-right config to ~/.tmux.conf
tmux source-file ~/.tmux.conf
```

This adds:

```
set -g status-interval 30
set -g status-right "#(~/ops/ai-usage-monitor/bin/tmuxfmt)"
```

The formatter outputs:

```
CL:28/59 CD:44/80
```

Where columns are: `CL:<5h-remaining>/<weekly-remaining> CD:<5h-remaining>/<weekly-remaining>`

If the cache is unavailable: `CL:-- CD:--`

---

## Sample Discord Notification

```
♻️ Claude 5h window reset
Remaining capacity: 100%
Next reset: 2025-01-15 14:30 JST
```

```
♻️ Codex weekly reset
Remaining capacity: 100%
Next reset: 2025-01-20 09:00 JST
```

---

## Development

```bash
make test    # go test ./...
make lint    # golangci-lint run ./...
make clean   # remove bin/
```

---

## Extension Points

| Area | How to extend |
|---|---|
| **New provider** | Implement `usage.Provider` in a new `internal/<name>/` package and register it in `cmd/daemon/main.go` |
| **Real Claude API** | Update `internal/claude/provider.go` with the actual endpoint and response schema |
| **Real Codex API** | Update `internal/codex/provider.go` with the correct session-cookie endpoint |
| **MCP adapter** | Add an `internal/mcp/` package that wraps providers as MCP tools |
| **Systemd service** | Write a `.service` unit pointing to the installed binary |
| **Prometheus metrics** | Add an HTTP metrics handler in the daemon exposing `usage_percent{provider,window}` gauges |
| **Slack notifier** | Implement `notify.Notifier` interface for Slack alongside the Discord implementation |

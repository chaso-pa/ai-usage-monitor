#!/usr/bin/env bash
set -euo pipefail

TMUX_CONF="${HOME}/.tmux.conf"
BINARY_PATH="${HOME}/ops/ai-usage-monitor/bin/tmuxfmt"

# Ensure the target binary directory exists.
mkdir -p "$(dirname "$BINARY_PATH")"

MARKER="# ai-usage-monitor"
if grep -qF "$MARKER" "$TMUX_CONF" 2>/dev/null; then
  echo "tmux config already contains ai-usage-monitor entries — skipping."
  exit 0
fi

cat >> "$TMUX_CONF" <<EOF

${MARKER}
set -g status-interval 30
set -g status-right "#(${BINARY_PATH})"
EOF

echo "Appended ai-usage-monitor status-right to ${TMUX_CONF}"
echo ""
echo "Next steps:"
echo "  1. Copy or symlink the tmuxfmt binary to: ${BINARY_PATH}"
echo "     e.g.  ln -sf \$(pwd)/bin/tmuxfmt ${BINARY_PATH}"
echo "  2. Reload tmux config: tmux source-file ~/.tmux.conf"

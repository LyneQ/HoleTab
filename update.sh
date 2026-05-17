#!/usr/bin/env bash
set -euo pipefail

APP=holetab
BIN="$HOME/.local/bin/$APP"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE="$SERVICE_DIR/$APP.service"

# ── Guards ────────────────────────────────────────────────────────────────────

if [[ $EUID -eq 0 ]]; then
  echo "error: do not run as root (user-level install)" && exit 1
fi

if ! systemctl --user is-enabled --quiet "$APP" 2>/dev/null; then
  echo "$APP is not installed. Run ./install.sh first." && exit 1
fi

if ! command -v make &>/dev/null; then
  echo "error: 'make' not found in PATH" && exit 1
fi

# ── Build ─────────────────────────────────────────────────────────────────────

echo "==> Building..."
make build

# ── Swap ──────────────────────────────────────────────────────────────────────

echo "==> Stopping service..."
systemctl --user stop "$APP"

echo "==> Swapping binary..."
cp "./bin/$APP" "$BIN"
chmod 755 "$BIN"

echo "==> Updating service file..."
cp "$APP.service" "$SERVICE"
systemctl --user daemon-reload

# ── Restart ───────────────────────────────────────────────────────────────────

echo "==> Restarting service..."
systemctl --user start "$APP"

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "Done! Status:"
systemctl --user status "$APP" --no-pager

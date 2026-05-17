#!/usr/bin/env bash
set -euo pipefail

APP=holetab
BIN="$HOME/.local/bin/$APP"
CONFIG_DIR="$HOME/.config/$APP"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE="$SERVICE_DIR/$APP.service"

# ── Guards ────────────────────────────────────────────────────────────────────

if [[ $EUID -eq 0 ]]; then
  echo "error: do not run as root (user-level install)" && exit 1
fi

# ── Service ───────────────────────────────────────────────────────────────────

echo "==> Stopping and disabling service..."
systemctl --user stop "$APP"   2>/dev/null || true
systemctl --user disable "$APP" 2>/dev/null || true

echo "==> Removing service file..."
rm -f "$SERVICE"
systemctl --user daemon-reload

# ── Binary ────────────────────────────────────────────────────────────────────

echo "==> Removing binary..."
rm -f "$BIN"

# ── Config / data (prompt) ────────────────────────────────────────────────────

echo ""
read -rp "Remove config and data ($CONFIG_DIR)? This includes your DB. [y/N] " confirm
if [[ "$confirm" =~ ^[Yy]$ ]]; then
  rm -rf "$CONFIG_DIR"
  echo "    Config and data removed."
else
  echo "    Config and data kept at $CONFIG_DIR"
fi

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "Done! $APP has been uninstalled."

#!/usr/bin/env bash
set -euo pipefail

APP=holetab
BIN_DIR="$HOME/.local/bin"
BIN="$BIN_DIR/$APP"
CONFIG_DIR="$HOME/.config/$APP"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE="$SERVICE_DIR/$APP.service"

# ── Guards ────────────────────────────────────────────────────────────────────

if [[ $EUID -eq 0 ]]; then
  echo "error: do not run as root (user-level install)" && exit 1
fi

if systemctl --user is-active --quiet "$APP" 2>/dev/null; then
  echo "$APP is already running. Use ./update.sh instead." && exit 1
fi

if ! command -v make &>/dev/null; then
  echo "error: 'make' not found in PATH" && exit 1
fi

# ── Build ─────────────────────────────────────────────────────────────────────

echo "==> Building..."
make build

# ── Binary ────────────────────────────────────────────────────────────────────

echo "==> Installing binary..."
mkdir -p "$BIN_DIR"
cp "./bin/$APP" "$BIN"
chmod 755 "$BIN"

# ── Config ────────────────────────────────────────────────────────────────────

echo "==> Creating config directory..."
mkdir -p "$CONFIG_DIR"

if [[ ! -f "$CONFIG_DIR/config.toml" ]]; then
  cp config.example.toml "$CONFIG_DIR/config.toml"
  echo "    Config installed — edit $CONFIG_DIR/config.toml before starting"
else
  echo "    Config already exists, skipping"
fi

# ── Systemd user service ──────────────────────────────────────────────────────

echo "==> Installing service..."
mkdir -p "$SERVICE_DIR"
cp "$APP.service" "$SERVICE"
systemctl --user daemon-reload
systemctl --user enable --now "$APP"

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "Done! Status:"
systemctl --user status "$APP" --no-pager

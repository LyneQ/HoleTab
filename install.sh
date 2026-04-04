#!/usr/bin/env bash
set -euo pipefail

APP=holetab
BIN=/usr/local/bin/$APP
SERVICE=/etc/systemd/system/$APP.service
CONFIG_DIR=/etc/$APP
DATA_DIR=/var/lib/$APP

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo ./install.sh" && exit 1
fi

if systemctl is-active --quiet $APP; then
  echo "$APP is already installed. Use ./update.sh instead." && exit 1
fi

if [[ ! -f ./bin/$APP ]]; then
  echo "Binary not found. Run 'make build' first." && exit 1
fi

echo "==> Creating user..."
useradd -r -s /sbin/nologin -d $DATA_DIR $APP 2>/dev/null || true

echo "==> Installing binary..."
cp ./bin/$APP $BIN
chmod 755 $BIN

echo "==> Creating directories..."
mkdir -p $CONFIG_DIR $DATA_DIR
chown $APP:$APP $DATA_DIR

echo "==> Installing config..."
if [[ ! -f $CONFIG_DIR/config.toml ]]; then
  cp config.example.toml $CONFIG_DIR/config.toml
  echo "    Edit $CONFIG_DIR/config.toml before starting"
else
  echo "    Config already exists, skipping"
fi

echo "==> Installing service..."
cp $APP.service $SERVICE
systemctl daemon-reload
systemctl enable --now $APP

echo ""
echo "Done! Status:"
systemctl status $APP --no-pager
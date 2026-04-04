#!/usr/bin/env bash
set -euo pipefail

APP=holetab
BIN=/usr/local/bin/$APP
SERVICE=/etc/systemd/system/$APP.service

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo ./update.sh" && exit 1
fi

if ! systemctl is-enabled --quiet $APP 2>/dev/null; then
  echo "$APP is not installed. Run ./install.sh first." && exit 1
fi

if [[ ! -f ./bin/$APP ]]; then
  echo "Binary not found. Run 'make build' first." && exit 1
fi

echo "==> Stopping service..."
systemctl stop $APP

echo "==> Swapping binary..."
cp ./bin/$APP $BIN
chmod 755 $BIN

echo "==> Updating service..."
cp $APP.service $SERVICE
systemctl daemon-reload

echo "==> Restarting service..."
systemctl start $APP

echo ""
echo "Done! Status:"
systemctl status $APP --no-pager
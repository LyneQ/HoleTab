# HoleTab

A self-hosted, single-binary new-tab page server.  
No Node.js, no extensions, no CGO — just one Go binary.

## Requirements

- Go 1.23+

> `templ` is installed automatically by `make build` if not present.

## Install

```sh
make install
```

This builds the binary, installs it to `~/.local/bin/holetab`, copies the default config to `~/.config/holetab/config.toml`, and enables the systemd user service.

Edit the config after installing (if needed):

```sh
$EDITOR ~/.config/holetab/config.toml
```

## Update

```sh
make update
```

## Uninstall

```sh
make uninstall
```

The config and database (`~/.config/holetab/`) are kept by default — you will be prompted.

## Service

```sh
systemctl --user status holetab
systemctl --user start holetab
systemctl --user stop holetab
systemctl --user restart holetab

# Logs
journalctl --user -u holetab -f
```

## Development

```sh
make build   # build the binary (output: ./bin/holetab)
make dev     # templ generate + go run
```

Open `http://localhost:3654`.

## Project layout

```
cmd/holetab/        — entry point
internal/config/    — config loading
internal/db/        — bbolt CRUD
internal/handler/   — HTTP handlers
internal/favicon/   — favicon resolver
internal/model/     — Link struct
web/templates/      — templ templates
web/static/         — embedded static assets (htmx, css)
install.sh          — first install
update.sh           — update binary + service
uninstall.sh        — uninstall
holetab.service     — systemd user unit
config.example.toml
```

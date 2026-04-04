```markdown
# HoleTab
A self-hosted, single-binary new-tab page server.  
No Node.js, no extensions, no CGO — just one Go binary.

## Requirements
- Go 1.23+
- [`templ`](https://templ.guide) CLI: `go install github.com/a-h/templ/cmd/templ@latest`

## Development
```sh
# Build the binary (output: ./bin/holetab)
make build

# Run in development (templ generate + go run, auto-downloads htmx)
make dev
```
Open `http://localhost:8080` in your browser.

## Configuration
Copy `config.example.toml` and edit before starting:
```toml
[server]
port = 8080

[database]
path = "/var/lib/holetab/holetab.db"
```

## Installation (systemd)
```sh
make build
sudo ./install.sh
```

`install.sh` installs the binary to `/usr/local/bin/`, copies the config to `/etc/holetab/config.toml` and enables the service at startup.


## Update
```sh
make build
sudo ./update.sh
```

## Service management
```sh
sudo systemctl status holetab
sudo systemctl start holetab
sudo systemctl stop holetab
sudo systemctl restart holetab

# Logs
journalctl -u holetab -f
```

## Project layout
```
cmd/holetab/      — main entry point
internal/config/  — config loading
internal/db/      — bbolt CRUD
internal/handler/ — HTTP handlers (chi)
internal/favicon/ — favicon URL resolver
internal/model/   — Link struct
web/templates/    — templ templates
web/static/       — embedded static assets (htmx, css)
install.sh        — first install
update.sh         — update binary + service
holetab.service   — systemd unit file
config.example.toml
```
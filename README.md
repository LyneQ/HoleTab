# HoleTab

A self-hosted, single-binary new-tab page server.  
No Node.js, no extensions, no CGO — just one Go binary.

## Requirements

- Go 1.22+
- [`templ`](https://templ.guide) CLI: `go install github.com/a-h/templ/cmd/templ@latest`

## Usage

```sh
# Build the binary (output: ./bin/holetab)
make build

# Run in development (templ generate + go run, auto-downloads htmx)
make dev
```

Open `http://localhost:3654` in your browser.

## Configuration

On first run, a `config.toml` is created in the working directory:

```toml
[server]
port = "3654"

[database]
path = "./holetab.db"
```

Edit the file and restart to apply changes.

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
```

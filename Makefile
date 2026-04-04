TEMPL ?= $(shell go env GOPATH)/bin/templ
.PHONY: build dev clean htmx install update

build: htmx
	$(TEMPL) generate ./web/templates/...
	go build -ldflags="-s -w" -o ./bin/holetab ./cmd/holetab

dev: htmx
	$(TEMPL) generate ./web/templates/... && go run ./cmd/holetab

clean:
	rm -rf ./bin

htmx:
	@if grep -q "PLACEHOLDER" web/static/htmx.min.js 2>/dev/null; then \
		echo "Downloading htmx.min.js..."; \
		curl -fsSL -o web/static/htmx.min.js \
			https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js; \
	fi

install:
	sudo ./install.sh

update:
	sudo ./update.sh
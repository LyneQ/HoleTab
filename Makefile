TEMPL ?= $(shell go env GOPATH)/bin/templ

.PHONY: build dev clean htmx

build: htmx
	$(TEMPL) generate ./web/templates/...
	go build -o ./bin/holetab ./cmd/holetab

dev: htmx
	$(TEMPL) generate ./web/templates/... && go run ./cmd/holetab

clean:
	rm -rf ./bin

# Download HTMX if the placeholder is still in place.
htmx:
	@if grep -q "PLACEHOLDER" web/static/htmx.min.js 2>/dev/null; then \
		echo "Downloading htmx.min.js..."; \
		curl -fsSL -o web/static/htmx.min.js \
			https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js; \
	fi

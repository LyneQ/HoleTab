TEMPL := $(shell go env GOPATH)/bin/templ
.PHONY: build dev clean htmx templ install update uninstall

templ:
	@if [ ! -f $(TEMPL) ]; then \
		echo "==> Installing templ..."; \
		go install github.com/a-h/templ/cmd/templ@latest; \
	fi

build: templ htmx
	$(TEMPL) generate ./web/templates/...
	go build -ldflags="-s -w" -o ./bin/holetab ./cmd/holetab

dev: templ htmx
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
	./install.sh

update:
	./update.sh

uninstall:
	./uninstall.sh
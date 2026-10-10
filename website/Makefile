GO       ?= go
NPM      ?= npm
VERSION  ?= $(shell cat VERSION)
GOOS     ?= linux
GOARCH   ?= amd64
# Set to false behind a TLS-intercepting proxy.
NPM_STRICT_SSL ?= true
# Absolute site URL baked into sitemap, RSS and Open Graph tags.
SITE_URL ?= http://localhost:8090
HOME_URL ?= http://localhost:8080
PKGS     := ./...

export NEXT_TELEMETRY_DISABLED := 1
# Slow proxy: fewer parallel downloads, long timeouts.
NPM_ENV := npm_config_strict_ssl=$(NPM_STRICT_SSL) npm_config_fetch_timeout=600000 \
	npm_config_fetch_retries=5 npm_config_fetch_retry_mintimeout=20000 npm_config_maxsockets=4

.PHONY: all
all: test build

web/node_modules: web/package-lock.json
	cd web && $(NPM_ENV) PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 $(NPM) ci
	touch $@

# Builds the Next.js static export and copies it into ui/dist for go:embed.
.PHONY: web
web: web/node_modules
	cd web && SITE_URL=$(SITE_URL) $(NPM) run build
	find ui/dist -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -R web/out/. ui/dist/

.PHONY: build
build: web
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o stock_website .

# Release tarball for the VM: static binary + systemd unit + env example.
.PHONY: dist
dist: web
	mkdir -p dist/stock_website-$(VERSION)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" \
		-o dist/stock_website-$(VERSION)/stock_website .
	cp ../deploy/vm/systemd/stock-website.service ../deploy/vm/env/stock-website.env.example dist/stock_website-$(VERSION)/
	tar -C dist -czf dist/stock_website-$(VERSION).$(GOOS)-$(GOARCH).tar.gz stock_website-$(VERSION)

.PHONY: test
test: web/node_modules
	$(GO) vet $(PKGS)
	$(GO) test -race $(PKGS)
	cd web && $(NPM) run typecheck && $(NPM) test

.PHONY: run
run: build
	./stock_website --demo.listen-address=:8091 --demo.home-url=$(HOME_URL)

# Needs stock_home on HOME_URL fed by the simulator (see docs/quickstart) and a Playwright Chromium.
.PHONY: screenshots
screenshots: web/node_modules
	cd web && npx playwright install chromium
	cd web && HOME_URL=$(HOME_URL) $(NPM) run screenshots

.PHONY: clean
clean:
	rm -rf stock_website dist web/out web/.next
	find ui/dist -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +

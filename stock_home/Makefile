GO       ?= go
NPM      ?= npm
VERSION  ?= $(shell cat VERSION)
IMAGE    ?= stock-home:$(VERSION)
# Set to false behind a TLS-intercepting proxy.
NPM_STRICT_SSL ?= true
PKGS     := . ./internal/... ./ui/...

export NEXT_TELEMETRY_DISABLED := 1

.PHONY: all
all: test build

web/node_modules: web/package-lock.json
	cd web && npm_config_strict_ssl=$(NPM_STRICT_SSL) $(NPM) ci
	touch $@

# Builds the Next.js static export and copies it into ui/dist for go:embed.
.PHONY: web
web: web/node_modules
	cd web && $(NPM) run build
	find ui/dist -mindepth 1 -maxdepth 1 ! -name .gitkeep -exec rm -rf {} +
	cp -R web/out/. ui/dist/

.PHONY: build
build: web
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o stock_home .

.PHONY: test
test: web/node_modules
	$(GO) vet $(PKGS)
	$(GO) test -race $(PKGS)
	cd web && $(NPM) run typecheck && $(NPM) test

.PHONY: run
run: build
	./stock_home --market.holidays-file=../stock_exporter/examples/holidays-nse-2026.yml

.PHONY: docker
docker:
	@test -x stock_home || { echo "stock_home binary missing: run 'make build' first"; exit 1; }
	docker build --pull=false -f Dockerfile.prebuilt -t $(IMAGE) .

# Builds UI and binary inside Docker (needs npm and Go module access from the build).
.PHONY: docker-multistage
docker-multistage:
	docker build --build-arg VERSION=$(VERSION) --build-arg NPM_STRICT_SSL=$(NPM_STRICT_SSL) -t $(IMAGE) .

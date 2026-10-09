# Builds every deploy service locally, packages the binaries into
# images and pushes them to Docker Hub or the OpenShift internal registry. See README.md.
SHELL := /bin/bash
.DEFAULT_GOAL := help

# dockerhub: push with your `docker login` credentials; ocp: OpenShift internal registry with your `oc` token.
PUSH_TO        ?= dockerhub
DOCKERHUB_USER ?= aseeni
OCP_REGISTRY   ?= default-route-openshift-image-registry.apps.ocp4poc.c4lab.local
NAMESPACE      ?= abdul
ifeq ($(PUSH_TO),ocp)
REGISTRY := $(OCP_REGISTRY)/$(NAMESPACE)
else ifeq ($(PUSH_TO),dockerhub)
REGISTRY := docker.io/$(DOCKERHUB_USER)
else
$(error PUSH_TO must be dockerhub or ocp, not '$(PUSH_TO)')
endif
# The lab registry and the Zscaler proxy present certificates that are not trusted locally.
TLS_VERIFY   ?= false
GO           ?= go
PNPM         ?= npx --yes pnpm@10

# Slow proxy: fewer parallel downloads, long timeouts.
NPM_ENV := npm_config_fetch_timeout=600000 npm_config_fetch_retries=5 \
	npm_config_fetch_retry_mintimeout=20000 npm_config_maxsockets=4
ELM_ENV :=
ifeq ($(TLS_VERIFY),false)
NPM_ENV += npm_config_strict_ssl=false
# Elm cannot skip TLS verification, so it trusts the proxy CA read from this probe connection.
PROXY_CA_PROBE ?= package.elm-lang.org:443
CA_DIR         := $(CURDIR)/.build/ca
ELM_ENV        := SYSTEM_CERTIFICATE_PATH=$(CA_DIR)
endif

EXPORTER_VERSION := $(shell cat stock_exporter/VERSION)
HOME_VERSION     := $(shell cat stock_home/VERSION)
PROM_VERSION     := $(shell cat prometheus/VERSION)
AM_VERSION       := $(shell cat alertmanager/VERSION)
OPENBB_DIR       := OpenBB/OpenBB
OPENBB_VERSION   := $(shell git -C $(OPENBB_DIR) rev-parse --short HEAD 2>/dev/null)
GRAFANA_IMAGE    ?= docker.io/grafana/grafana:13.2.3
# alpine-slim drops the optional njs/xslt/image-filter/geoip modules, which the portal does not load.
NGINX_IMAGE      ?= docker.io/nginxinc/nginx-unprivileged:1.28-alpine-slim
PYTHON_IMAGE     ?= docker.io/library/python:3.11-slim
BASE_IMAGES      := gcr.io/distroless/static-debian12:nonroot quay.io/prometheus/busybox-linux-amd64:latest

# Local name:tag of each image; pushed under the same name:tag to $(REGISTRY). Must match the Helm values.
IMG_stock-exporter := stock-exporter:$(EXPORTER_VERSION)
IMG_stock-home     := stock-home:$(HOME_VERSION)
IMG_prometheus     := prometheus:v$(PROM_VERSION)
IMG_alertmanager   := alertmanager:v$(AM_VERSION)
IMG_grafana        := $(notdir $(GRAFANA_IMAGE))
IMG_nginx          := $(notdir $(NGINX_IMAGE))
IMG_openbb         := openbb:$(OPENBB_VERSION)

BUILT    := stock-exporter stock-home prometheus alertmanager
MIRRORED := grafana nginx
# Built entirely inside docker (no local build-<name> step).
DOCKER_ONLY := openbb
IMAGES   := $(BUILT) $(MIRRORED) $(DOCKER_ONLY)

BUILD_USER := $(USER)@$(shell hostname)
BUILD_DATE := $(shell date -u +%Y%m%d-%H:%M:%S)
# $(call version_ldflags,<repo dir>,<version>): same version stamping as promu.
version_ldflags = -s -w \
	-X github.com/prometheus/common/version.Version=$(2) \
	-X github.com/prometheus/common/version.Revision=$(shell git -C $(1) rev-parse HEAD 2>/dev/null) \
	-X github.com/prometheus/common/version.Branch=$(shell git -C $(1) rev-parse --abbrev-ref HEAD 2>/dev/null) \
	-X github.com/prometheus/common/version.BuildUser=$(BUILD_USER) \
	-X github.com/prometheus/common/version.BuildDate=$(BUILD_DATE)

.PHONY: help
help:
	@echo "Usage: make <target> [PUSH_TO=dockerhub|ocp] [DOCKERHUB_USER=...] [OCP_REGISTRY=...] [NAMESPACE=...] [TLS_VERIFY=true|false]"
	@echo
	@echo "  build          Build binaries (incl. web UIs): $(BUILT)"
	@echo "  docker-build   Build/copy local images: $(IMAGES)"
	@echo "  docker-push    Push all images to $(REGISTRY)"
	@echo "  push           Same as docker-push (no build)"
	@echo "  helm-package   Lint and package the chart into .build/charts"
	@echo "  helm-push      Push the packaged chart to $(HELM_REGISTRY)"
	@echo "  all            build + docker-build + docker-push"
	@echo "  deploy         helm upgrade --install (deploy/Makefile deploy-ocp)"
	@echo "  clean          Remove build outputs"
	@echo "  build-<name>, docker-build-<name>, docker-push-<name> for a single service"
	@echo
	@echo "Images:"
	@$(foreach i,$(IMAGES),printf '  %-15s %s\n' "$(i)" "$(IMG_$(i))";)

.PHONY: all
all: build docker-build docker-push

# --- build ---------------------------------------------------------------------

.PHONY: build $(addprefix build-,$(BUILT))
build: $(addprefix build-,$(BUILT))

build-stock-exporter:
	$(MAKE) -C stock_exporter build

build-stock-home:
	$(MAKE) -C stock_home build NPM_STRICT_SSL=$(TLS_VERIFY)

build-prometheus:
	cd prometheus/web/ui && $(NPM_ENV) $(PNPM) install --frozen-lockfile --config.strict-ssl=$(TLS_VERIFY)
	cd prometheus/web/ui && $(NPM_ENV) CI="" $(PNPM) run build:mantine-ui
	cd prometheus && scripts/compress_assets.sh
	cd prometheus && for b in prometheus promtool; do \
		CGO_ENABLED=0 $(GO) build -trimpath -tags netgo,builtinassets \
			-ldflags "$(call version_ldflags,prometheus,$(PROM_VERSION))" \
			-o .build/linux-amd64/$$b ./cmd/$$b || exit 1; \
	done

build-alertmanager: $(if $(CA_DIR),$(CA_DIR)/proxy-ca.pem)
	cd alertmanager && $(NPM_ENV) $(ELM_ENV) $(MAKE) ui-elm ui-mantine
	cd alertmanager && for b in alertmanager amtool; do \
		CGO_ENABLED=0 $(GO) build -trimpath -tags netgo \
			-ldflags "$(call version_ldflags,alertmanager,$(AM_VERSION))" \
			-o .build/linux-amd64/$$b ./cmd/$$b || exit 1; \
	done

ifdef CA_DIR
# System CAs plus the last certificate the proxy presents (its intermediate CA).
$(CA_DIR)/proxy-ca.pem:
	mkdir -p $(CA_DIR)
	cp /etc/ssl/certs/ca-certificates.crt $(CA_DIR)/
	echo | openssl s_client -connect $(PROXY_CA_PROBE) -servername $(firstword $(subst :, ,$(PROXY_CA_PROBE))) -showcerts 2>/dev/null \
		| awk '/BEGIN CERT/{c=""} {c=c $$0 "\n"} /END CERT/{last=c} END{printf "%s", last}' > $@
	@test -s $@ || { echo "Could not read the proxy CA from $(PROXY_CA_PROBE)"; rm -f $@; exit 1; }
endif

# --- docker-build --------------------------------------------------------------

.PHONY: docker-build base-images $(addprefix docker-build-,$(IMAGES))
docker-build: base-images $(addprefix docker-build-,$(IMAGES))

base-images:
	for img in $(BASE_IMAGES); do \
		docker image inspect $$img >/dev/null 2>&1 || \
		skopeo copy --src-tls-verify=$(TLS_VERIFY) docker://$$img docker-daemon:$$img || exit 1; \
	done

# $(call require,<file>,<build target>)
require = @test -x $(1) || { echo "$(1) is missing: run 'make $(2)' first"; exit 1; }

docker-build-stock-exporter: base-images
	$(call require,stock_exporter/stock_exporter,build-stock-exporter)
	docker build --pull=false -f stock_exporter/Dockerfile.prebuilt -t $(IMG_stock-exporter) stock_exporter

docker-build-stock-home: base-images
	$(MAKE) -C stock_home docker IMAGE=$(IMG_stock-home)

docker-build-prometheus: base-images
	$(call require,prometheus/.build/linux-amd64/prometheus,build-prometheus)
	docker build --pull=false -t $(IMG_prometheus) prometheus

docker-build-alertmanager: base-images
	$(call require,alertmanager/.build/linux-amd64/alertmanager,build-alertmanager)
	docker build --pull=false -t $(IMG_alertmanager) alertmanager

docker-build-grafana:
	skopeo copy --src-tls-verify=$(TLS_VERIFY) docker://$(GRAFANA_IMAGE) docker-daemon:$(IMG_grafana)

docker-build-nginx:
	skopeo copy --src-tls-verify=$(TLS_VERIFY) docker://$(NGINX_IMAGE) docker-daemon:$(IMG_nginx)

# Context is a clean archive of the tracked platform sources; the submodule's .git is a gitfile.
docker-build-openbb:
	@test -n "$(OPENBB_VERSION)" || { echo "$(OPENBB_DIR) is not checked out"; exit 1; }
	docker image inspect $(PYTHON_IMAGE) >/dev/null 2>&1 || \
		skopeo copy --src-tls-verify=$(TLS_VERIFY) docker://$(PYTHON_IMAGE) docker-daemon:$(PYTHON_IMAGE)
	rm -rf .build/openbb && mkdir -p .build/openbb
	git -C $(OPENBB_DIR) archive --format=tar HEAD openbb_platform | tar -x -C .build/openbb
	docker build --pull=false -f deploy/openbb/Dockerfile \
		--build-arg PYTHON_IMAGE=$(PYTHON_IMAGE) --build-arg TLS_VERIFY=$(TLS_VERIFY) \
		-t $(IMG_openbb) .build/openbb

# --- docker-push ---------------------------------------------------------------

.PHONY: docker-push push registry-login
docker-push: registry-login $(addprefix docker-push-,$(IMAGES))

# Pushes the images already in the local docker daemon; never rebuilds.
push: docker-push

ifeq ($(PUSH_TO),ocp)
registry-login:
	oc whoami -t | skopeo login --tls-verify=$(TLS_VERIFY) -u unused --password-stdin $(OCP_REGISTRY)

# Staged on disk: streamed uploads of large layers get HTTP 400 from the registry route.
define push_rule
.PHONY: docker-push-$(1)
docker-push-$(1): registry-login
	rm -rf .build/push && mkdir -p .build
	skopeo copy --retry-times 3 docker-daemon:$(IMG_$(1)) dir:.build/push
	skopeo copy --retry-times 3 --dest-tls-verify=$(TLS_VERIFY) dir:.build/push docker://$(REGISTRY)/$(IMG_$(1))
	rm -rf .build/push
endef
else
# Credentials come from `docker login`; nothing is read or stored here.
registry-login:
	@docker info >/dev/null 2>&1 || { echo "docker daemon is not reachable"; exit 1; }

define push_rule
.PHONY: docker-push-$(1)
docker-push-$(1): registry-login
	@docker image inspect $(IMG_$(1)) >/dev/null 2>&1 || { echo "$(IMG_$(1)) is missing: run 'make docker-build-$(1)' first"; exit 1; }
	docker tag $(IMG_$(1)) $(REGISTRY)/$(IMG_$(1))
	docker push $(REGISTRY)/$(IMG_$(1))
endef
endif
$(foreach i,$(IMAGES),$(eval $(call push_rule,$(i))))

# --- deploy / clean -------------------------------------------------------------

# The chart is published as an OCI artifact: oci://registry-1.docker.io/<user>/stock-observability.
# Helm reuses the `docker login` credentials; otherwise run `helm registry login registry-1.docker.io`.
CHART_DIR     := deploy/charts/stock-observability
CHART_VERSION := $(shell awk '/^version:/ {print $$2}' $(CHART_DIR)/Chart.yaml)
CHART_PKG     := .build/charts/stock-observability-$(CHART_VERSION).tgz
HELM_REGISTRY ?= oci://registry-1.docker.io/$(DOCKERHUB_USER)

.PHONY: helm-package helm-push
helm-package:
	helm lint $(CHART_DIR)
	mkdir -p .build/charts
	helm package $(CHART_DIR) -d .build/charts

helm-push: helm-package
	helm push $(CHART_PKG) $(HELM_REGISTRY)

.PHONY: deploy
deploy:
	$(MAKE) -C deploy deploy-ocp

.PHONY: clean
clean:
	rm -rf .build prometheus/.build alertmanager/.build stock_exporter/stock_exporter stock_home/stock_home

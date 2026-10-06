# Copyright 2026 The Stock Exporter Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO        ?= go
VERSION   ?= $(shell cat VERSION)
REVISION  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
IMAGE     ?= stock-exporter:$(VERSION)
NIFTY_CSV ?= https://archives.nseindia.com/content/indices/ind_nifty200list.csv

LDFLAGS := -s -w \
	-X github.com/prometheus/common/version.Version=$(VERSION) \
	-X github.com/prometheus/common/version.Revision=$(REVISION) \
	-X github.com/prometheus/common/version.BuildDate=$(shell date -u +%Y%m%d-%H:%M:%S)

.PHONY: all
all: vet test build

.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o stock_exporter .

.PHONY: test
test:
	$(GO) test -race ./...

.PHONY: bench
bench:
	$(GO) test -run xxx -bench . -benchmem -count 6 .

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: run-sim
run-sim: build
	./stock_exporter --feed.source=sim --market.force-open \
		--stock.watchlist-file=examples/watchlist.yml \
		--market.holidays-file=examples/holidays-nse-2026.yml

.PHONY: watchlist
watchlist:
	mkdir -p .build
	curl -fsSL -A "Mozilla/5.0" -o .build/nifty200.csv $(NIFTY_CSV)
	$(GO) run ./tools/watchlistgen -constituents .build/nifty200.csv -out examples/watchlist.yml

.PHONY: docker
docker:
	docker build -t $(IMAGE) .

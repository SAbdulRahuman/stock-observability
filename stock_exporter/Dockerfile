# Copyright 2026 The Stock Exporter Authors
# Licensed under the Apache License, Version 2.0.

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/prometheus/common/version.Version=${VERSION}" \
      -o /out/stock_exporter .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/stock_exporter /bin/stock_exporter
COPY examples/watchlist.yml /etc/stock_exporter/watchlist.yml
COPY examples/holidays-nse-2026.yml /etc/stock_exporter/holidays.yml
# Kite token is persisted here; mount a volume so logins survive restarts.
WORKDIR /var/lib/stock_exporter
EXPOSE 9101
USER nonroot:nonroot
ENTRYPOINT ["/bin/stock_exporter"]
CMD ["--stock.watchlist-file=/etc/stock_exporter/watchlist.yml", \
     "--market.holidays-file=/etc/stock_exporter/holidays.yml", \
     "--kite.token-file=/var/lib/stock_exporter/kite_session.json"]

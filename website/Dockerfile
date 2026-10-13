FROM node:24-alpine AS web
WORKDIR /src/website/web
ENV NEXT_TELEMETRY_DISABLED=1
ARG NPM_STRICT_SSL=true
COPY website/web/package.json website/web/package-lock.json website/web/.npmrc ./
RUN npm_config_strict_ssl=${NPM_STRICT_SSL} npm ci
COPY website/web/ ./
COPY website/content/ /src/website/content/
COPY stock-observability-proposal/*.md /src/stock-observability-proposal/
ARG SITE_URL=http://localhost:8090
RUN SITE_URL=${SITE_URL} npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src/website
COPY website/go.mod website/go.sum ./
RUN go mod download
COPY website/ ./
COPY --from=web /src/website/web/out/ ./ui/dist/
ARG VERSION=dev
RUN mkdir -p /data \
    && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/stock_website . \
    && chown 65532:65532 /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/stock_website /bin/stock_website
COPY --from=build --chown=65532:65532 /data /data
WORKDIR /data
USER nonroot:nonroot
EXPOSE 8090 8091
ENTRYPOINT ["/bin/stock_website"]
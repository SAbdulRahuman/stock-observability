FROM node:24-alpine AS web
WORKDIR /src/web
ENV NEXT_TELEMETRY_DISABLED=1
ARG NPM_STRICT_SSL=true
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm_config_strict_ssl=${NPM_STRICT_SSL} npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/out/ ./ui/dist/
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/stock_home .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/stock_home /bin/stock_home
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/stock_home"]

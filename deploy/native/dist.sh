#!/usr/bin/env bash
# Builds the native-install release assets (install.sh / install.ps1) into $OUT:
#   stock-observability_linux_amd64.tar.gz, stock-observability_windows_amd64.zip,
#   install.sh, install.ps1, checksums.txt
# Needs go, npm (stock_home UI), uv (OpenBB wheels), python3 and the OpenBB/OpenBB submodule.
# SKIP_WEB=1 reuses an existing stock_home/ui/dist build.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
VERSION=${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty)}
OUT=${OUT:-$ROOT/.build/release}
WORK=$ROOT/.build/native
NATIVE=$ROOT/deploy/native
OPENBB_PLATFORM=$ROOT/OpenBB/OpenBB/openbb_platform
REVISION=$(git -C "$ROOT" rev-parse --short HEAD)

# Same package set as deploy/openbb/Dockerfile (OPENBB_PACKAGES).
OPENBB_PACKAGES=(
	core obbject_extensions/charting
	extensions/platform_api extensions/equity extensions/news extensions/etf extensions/index
	extensions/derivatives extensions/crypto extensions/currency extensions/economy
	extensions/fixedincome extensions/commodity extensions/technical extensions/quantitative
	providers/cboe providers/nasdaq providers/sec providers/tmx providers/fred
	providers/federal_reserve providers/government_us providers/cftc providers/famafrench
	providers/jodi providers/deribit
)

log() { printf '==> %s\n' "$*" >&2; }

rm -rf "$WORK" "$OUT"
mkdir -p "$WORK" "$OUT"

if [[ ${SKIP_WEB:-0} != 1 ]]; then
	log "Building stock_home UI"
	make -C "$ROOT/stock_home" web
fi
[[ -f $ROOT/stock_home/ui/dist/index.html ]] || { echo "stock_home/ui/dist is empty: build the UI first" >&2; exit 1; }

log "Building OpenBB wheels"
[[ -d $OPENBB_PLATFORM/core ]] || { echo "OpenBB/OpenBB submodule is not checked out" >&2; exit 1; }
for p in "${OPENBB_PACKAGES[@]}"; do
	uv build --quiet --wheel --out-dir "$WORK/wheels" "$OPENBB_PLATFORM/$p"
done

stage_config() {
	local c=$1/config
	mkdir -p "$c/prometheus/rules" "$c/alertmanager/templates" "$c/grafana/dashboards" "$c/stock_exporter/textfile"
	cp "$NATIVE/prometheus.yml" "$c/prometheus/"
	cp "$ROOT"/deploy/prometheus/rules/*.rules.yml "$c/prometheus/rules/"
	# Relative template paths resolve against the config file directory.
	sed 's#/etc/alertmanager/templates/#templates/#' "$ROOT/deploy/alertmanager/alertmanager.yml" >"$c/alertmanager/alertmanager.yml"
	cp "$ROOT"/deploy/alertmanager/templates/*.tmpl "$c/alertmanager/templates/"
	cp "$NATIVE/grafana/grafana.ini" "$c/grafana/"
	cp -R "$NATIVE/grafana/provisioning" "$c/grafana/"
	cp "$ROOT"/deploy/grafana/dashboards/*.json "$c/grafana/dashboards/"
	cp "$ROOT/stock_exporter/examples/watchlist.yml" "$c/stock_exporter/"
	cp "$ROOT/stock_exporter/examples/holidays-nse-2026.yml" "$c/stock_exporter/holidays.yml"
	cp "$ROOT"/deploy/stock_exporter/textfile/*.prom "$c/stock_exporter/textfile/"
	cp "$NATIVE/Caddyfile" "$c/"
}

for os in linux windows; do
	ext=""
	[[ $os == windows ]] && ext=.exe
	stage=$WORK/$os/stock-observability
	mkdir -p "$stage/bin" "$stage/openbb"
	log "Building $os/amd64 binaries"
	(cd "$ROOT/stock_exporter" && CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build -trimpath \
		-ldflags "-s -w -X github.com/prometheus/common/version.Version=$(cat VERSION) -X github.com/prometheus/common/version.Revision=$REVISION" \
		-o "$stage/bin/stock_exporter$ext" .)
	(cd "$ROOT/stock_home" && CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build -trimpath \
		-ldflags "-s -w -X main.version=$(cat VERSION)" \
		-o "$stage/bin/stock_home$ext" .)
	stage_config "$stage"
	cp -R "$WORK/wheels" "$stage/openbb/"
	cp "$NATIVE/manifest.env" "$stage/"
	printf '%s\n' "$VERSION" >"$stage/VERSION"
done

log "Packaging $VERSION"
tar -C "$WORK/linux" -czf "$OUT/stock-observability_linux_amd64.tar.gz" stock-observability
(cd "$WORK/windows" && python3 -m zipfile -c "$OUT/stock-observability_windows_amd64.zip" stock-observability)
cp "$ROOT/install.sh" "$ROOT/install.ps1" "$OUT/"
(cd "$OUT" && sha256sum -- *.tar.gz *.zip install.sh install.ps1 >checksums.txt)
ls -l "$OUT"

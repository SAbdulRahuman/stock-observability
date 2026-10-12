#!/usr/bin/env bash
# Builds the native-install release assets (install.sh / install.ps1) into $OUT:
#   stock-observability_linux_amd64.tar.gz, stock-observability_windows_amd64.zip,
#   install.sh, install.ps1, checksums.txt
# plus standalone <component>-<version>.<os>-<arch> archives of stock_exporter, stock_home and stock_website.
# Needs gcc, make, libpcre2-dev, strip, curl, unzip, go, npm (stock_home UI), uv (OpenBB wheels), python3 and OpenBB/OpenBB.
# SKIP_WEB=1 reuses existing stock_home/ui/dist and website/ui/dist builds; SKIP_STANDALONE=1 skips the standalone archives.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
VERSION=${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty)}
OUT=${OUT:-$ROOT/.build/release}
WORK=$ROOT/.build/native
NATIVE=$ROOT/deploy/native
OPENBB_PLATFORM=$ROOT/OpenBB/OpenBB/openbb_platform
REVISION=$(git -C "$ROOT" rev-parse --short HEAD)
SITE_URL=${SITE_URL:-https://iseeticks.com}
PLATFORMS=(linux/amd64 windows/amd64)
set -a
. "$NATIVE/manifest.env"
set +a

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

fetch_pinned() {
	local url=$1 sha=$2 dest=$3
	curl -fsSL --retry 3 -o "$dest" "$url"
	printf '%s  %s\n' "$sha" "$dest" | sha256sum -c --quiet -
}

rm -rf "$WORK" "$OUT"
mkdir -p "$WORK" "$OUT"

if [[ ${SKIP_WEB:-0} != 1 ]]; then
	log "Building stock_home UI"
	make -C "$ROOT/stock_home" web
fi
[[ -f $ROOT/stock_home/ui/dist/index.html ]] || { echo "stock_home/ui/dist is empty: build the UI first" >&2; exit 1; }

log "Building static Nginx $NGINX_VERSION"
nginx_src=$WORK/nginx-source
git clone --quiet --depth 1 --branch "release-$NGINX_VERSION" https://github.com/nginx/nginx.git "$nginx_src"
[[ $(git -C "$nginx_src" rev-parse HEAD) == "$NGINX_SOURCE_COMMIT" ]] || {
	echo "Nginx source tag does not match the pinned commit" >&2
	exit 1
}
(
	cd "$nginx_src"
	./auto/configure --prefix=/ --without-http_gzip_module --with-cc-opt=-static --with-ld-opt=-static
	make -j"${JOBS:-2}"
)
strip "$nginx_src/objs/nginx"
mkdir -p "$WORK/nginx-static" "$WORK/nginx-windows"
cp "$nginx_src/objs/nginx" "$WORK/nginx-static/nginx"
nginx_windows_zip=$WORK/nginx-windows.zip
fetch_pinned "$NGINX_WINDOWS_URL" "$NGINX_WINDOWS_SHA256" "$nginx_windows_zip"
unzip -q "$nginx_windows_zip" -d "$WORK/nginx-windows"

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
	cp "$NATIVE/nginx.conf" "$c/"
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
	if [[ $os == linux ]]; then
		cp "$WORK/nginx-static/nginx" "$stage/bin/nginx"
	else
		cp "$WORK/nginx-windows/nginx-$NGINX_VERSION/nginx.exe" "$stage/bin/nginx.exe"
	fi
done

log "Packaging $VERSION"
tar -C "$WORK/linux" -czf "$OUT/stock-observability_linux_amd64.tar.gz" stock-observability
(cd "$WORK/windows" && python3 -m zipfile -c "$OUT/stock-observability_windows_amd64.zip" stock-observability)
cp "$ROOT/install.sh" "$ROOT/install.ps1" "$OUT/"

# standalone <name> <source dir> <ldflags> [extra files...]
standalone() {
	local name=$1 src=$2 ldflags=$3; shift 3
	local ver; ver=$(cat "$src/VERSION")
	for p in "${PLATFORMS[@]}"; do
		local os=${p%/*} arch=${p#*/} ext=""
		[[ $os == windows ]] && ext=.exe
		local dir=$name-$ver.$os-$arch stage=$WORK/standalone
		mkdir -p "$stage/$dir"
		(cd "$src" && CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "-s -w $ldflags" -o "$stage/$dir/$name$ext" .)
		[[ $# -gt 0 ]] && cp "$@" "$stage/$dir/"
		if [[ $os == windows ]]; then
			(cd "$stage" && python3 -m zipfile -c "$OUT/$dir.zip" "$dir")
		else
			tar -C "$stage" -czf "$OUT/$dir.tar.gz" "$dir"
		fi
	done
}

if [[ ${SKIP_STANDALONE:-0} != 1 ]]; then
	if [[ ${SKIP_WEB:-0} != 1 ]]; then
		log "Building website UI for $SITE_URL"
		make -C "$ROOT/website" web SITE_URL="$SITE_URL"
	fi
	[[ -f $ROOT/website/ui/dist/index.html ]] || { echo "website/ui/dist is empty: build the UI first" >&2; exit 1; }
	ex=$ROOT/stock_exporter
	log "Building standalone stock_exporter"
	standalone stock_exporter "$ex" \
		"-X github.com/prometheus/common/version.Version=$(cat "$ex/VERSION") -X github.com/prometheus/common/version.Revision=$REVISION" \
		"$ex/LICENSE" "$ex/NOTICE" "$ex/examples/watchlist.yml" "$ex/examples/holidays-nse-2026.yml"
	log "Building standalone stock_home"
	standalone stock_home "$ROOT/stock_home" "-X main.version=$(cat "$ROOT/stock_home/VERSION")" \
		"$ex/examples/holidays-nse-2026.yml"
	log "Building standalone stock_website"
	standalone stock_website "$ROOT/website" "-X main.version=$(cat "$ROOT/website/VERSION")" \
		"$ROOT/deploy/vm/systemd/stock-website.service" "$ROOT/deploy/vm/env/stock-website.env.example"
fi

(cd "$OUT" && sha256sum -- *.tar.gz *.zip install.sh install.ps1 >checksums.txt)
ls -l "$OUT"

#!/usr/bin/env bash
# Stock Observability native installer for Linux (x86_64), per user: no root, no Docker.
#
#   curl -fsSL https://raw.githubusercontent.com/SAbdulRahuman/stock-observability/main/install.sh | bash
#   curl -fsSL .../install.sh | bash -s -- --version v0.1.1
#   curl -fsSL .../install.sh | bash -s -- --uninstall [--purge]
#
# Connect your Zerodha Kite account in Settings → Data Sources afterwards, or pass
# --kite-api-key / KITE_API_KEY and KITE_API_SECRET for unattended installs. Re-running upgrades and keeps config and data.
set -euo pipefail

REPO=${STOCKOBS_REPO:-SAbdulRahuman/stock-observability}
APP_DIR=${STOCKOBS_APP_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/stock-observability}
CONF_DIR=${STOCKOBS_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/stock-observability}
DATA_DIR=$APP_DIR/data
UNIT_DIR=${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user
SERVICES=(prometheus alertmanager grafana exporter openbb home portal)
MIN_FREE_KB=$((3 * 1024 * 1024))

VERSION=latest
ARCHIVE=""
KITE_KEY=${KITE_API_KEY:-}
BIND=""
PORT=""
START=1
OPEN_SETUP=1
ACTION=install
PURGE=0
TMP=""

usage() {
	cat <<'EOF'
Usage: install.sh [options]

  --version TAG        Release to install (default: latest)
  --kite-api-key KEY   Kite Connect API key (or KITE_API_KEY), with the secret in KITE_API_SECRET;
					   optional: you can enter both in Settings → Data Sources instead
  --listen ADDR        Portal bind address (default 127.0.0.1; 0.0.0.0 exposes it to the
                       network, and Prometheus/Alertmanager have no authentication)
  --port PORT          Portal port (default 8080)
  --archive FILE       Install from a local stock-observability_linux_amd64.tar.gz
  --no-start           Install without starting the services
	--no-browser         Do not open Settings → Data Sources in a browser after installing
  --uninstall          Stop and remove the services and programs (config and data are kept)
  --purge              With --uninstall, also delete config and data
  -h, --help           Show this help
EOF
}

log() { printf '\033[1;34m==>\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
die() {
	printf '\033[1;31merror:\033[0m %s\n' "$*" >&2
	exit 1
}

parse_args() {
	while (($#)); do
		case $1 in
		--version) VERSION=${2:?--version needs a value}; shift ;;
		--kite-api-key) KITE_KEY=${2:?--kite-api-key needs a value}; shift ;;
		--listen) BIND=${2:?--listen needs a value}; shift ;;
		--port) PORT=${2:?--port needs a value}; shift ;;
		--archive) ARCHIVE=${2:?--archive needs a value}; shift ;;
		--no-start) START=0 ;;
		--no-browser) OPEN_SETUP=0 ;;
		--uninstall) ACTION=uninstall ;;
		--purge) PURGE=1 ;;
		-h | --help) usage; exit 0 ;;
		*) usage >&2; die "unknown option: $1" ;;
		esac
		shift
	done
	[[ -z $ARCHIVE || -f $ARCHIVE ]] || die "archive not found: $ARCHIVE"
	[[ -z $PORT || $PORT =~ ^[0-9]+$ ]] || die "invalid --port: $PORT"
	[[ -z $BIND || $BIND =~ ^[0-9a-fA-F.:]+$ ]] || die "invalid --listen address: $BIND"
	[[ $VERSION =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid --version: $VERSION"
}

need() {
	local c
	for c in "$@"; do
		command -v "$c" >/dev/null || die "'$c' is required but not installed"
	done
}

preflight() {
	[[ $(uname -s) == Linux ]] || die "this installer supports Linux only (use install.ps1 on Windows)"
	[[ $(uname -m) == x86_64 ]] || die "only x86_64 is supported, not $(uname -m)"
	((EUID != 0)) || die "run as your normal user, not root: services are installed per user"
	need curl tar sha256sum systemctl loginctl
	systemctl --user show-environment >/dev/null 2>&1 ||
		die "systemd user services are not available (on WSL, enable systemd in /etc/wsl.conf and restart WSL)"
	mkdir -p "$APP_DIR"
	local free
	free=$(df -Pk "$APP_DIR" | awk 'NR == 2 { print $4 }')
	((free >= MIN_FREE_KB)) || die "at least 3 GB free space is needed in $APP_DIR"
}

kite_valid() { [[ $1 =~ ^[A-Za-z0-9]{6,128}$ ]]; }

# kite_credentials writes $CONF_DIR/kite_credentials.json, which stock_exporter owns and stock_home edits.
# Credentials are optional: without them, configure Kite from Settings → Data Sources.
kite_credentials() {
	local file=$CONF_DIR/kite_credentials.json old_secret=$CONF_DIR/kite_api_secret secret=${KITE_API_SECRET:-}
	umask 077
	mkdir -p "$CONF_DIR"
	# The v0.1.0 installer kept the key in stockobs.env and the secret in kite_api_secret.
	if [[ -z $KITE_KEY && -z $secret && ! -f $file && -s $old_secret && -f $CONF_DIR/stockobs.env ]]; then
		local old_key
		old_key=$(sed -n 's/^KITE_API_KEY=//p' "$CONF_DIR/stockobs.env")
		if [[ -n $old_key ]]; then
			log "Moving Kite credentials to $file"
			KITE_KEY=$old_key
			secret=$(<"$old_secret")
		fi
	fi
	if [[ -n $KITE_KEY || -n $secret ]]; then
		kite_valid "$KITE_KEY" || die "invalid Kite API key (--kite-api-key or KITE_API_KEY): 6-128 letters or digits"
		kite_valid "$secret" || die "KITE_API_SECRET must be set with the API key: 6-128 letters or digits"
		printf '{"api_key":"%s","api_secret":"%s"}\n' "$KITE_KEY" "$secret" >"$file"
		chmod 600 "$file"
	fi
	rm -f "$old_secret"
	umask 022
	KITE_CONFIGURED=0
	if [[ -s $file ]] && grep -q '"api_key":"[A-Za-z0-9]' "$file"; then
		KITE_CONFIGURED=1
	fi
}

fetch() {
	curl -fsSL --proto '=https' --tlsv1.2 --retry 3 -o "$2" "$1" || die "download failed: $1"
}

verify() {
	printf '%s  %s\n' "$2" "$1" | sha256sum -c --quiet - >/dev/null 2>&1 || die "checksum mismatch: $(basename "$1")"
}

download_release() {
	local asset=stock-observability_linux_amd64.tar.gz base
	if [[ -n $ARCHIVE ]]; then
		log "Using local archive $ARCHIVE"
		cp "$ARCHIVE" "$TMP/$asset"
	else
		if [[ $VERSION == latest ]]; then
			base=https://github.com/$REPO/releases/latest/download
		else
			base=https://github.com/$REPO/releases/download/$VERSION
		fi
		log "Downloading $REPO release ($VERSION)"
		fetch "$base/checksums.txt" "$TMP/checksums.txt"
		fetch "$base/$asset" "$TMP/$asset"
		verify "$TMP/$asset" "$(awk -v a="$asset" '$2 == a { print $1 }' "$TMP/checksums.txt")"
	fi
	tar -C "$TMP" -xzf "$TMP/$asset"
	REL=$TMP/stock-observability
	[[ -f $REL/manifest.env ]] || die "release archive is missing manifest.env"
	RELEASE_VERSION=$(<"$REL/VERSION")
}

# manifest.env is parsed, not sourced.
manifest() {
	local v
	v=$(sed -n "s/^$1=//p" "$REL/manifest.env")
	[[ -n $v ]] || die "manifest.env has no $1"
	printf '%s' "$v"
}

# vendor NAME: downloads the pinned upstream archive into $APP_DIR/vendor/NAME-VERSION once.
vendor() {
	local name=$1 key=${1^^} version dir file
	version=$(manifest "${key}_VERSION")
	dir=$APP_DIR/vendor/$name-$version
	if [[ ! -f $dir/.complete ]]; then
		log "Downloading $name $version"
		file=$TMP/$name.tar.gz
		fetch "$(manifest "${key}_LINUX_URL")" "$file"
		verify "$file" "$(manifest "${key}_LINUX_SHA256")"
		rm -rf "$dir"
		mkdir -p "$dir"
		if [[ $name == caddy ]]; then
			tar -C "$dir" -xzf "$file"
		else
			tar -C "$dir" -xzf "$file" --strip-components=1
		fi
		touch "$dir/.complete"
	fi
	printf '%s' "$dir"
}

install_openbb() {
	local venv=$APP_DIR/openbb stamp=$APP_DIR/openbb/.stockobs-version wheels=() w
	if [[ -f $stamp && $(<"$stamp") == "$RELEASE_VERSION" ]]; then
		return
	fi
	log "Installing OpenBB (Python $(manifest OPENBB_PYTHON_VERSION)); this takes a few minutes"
	export UV_PYTHON_INSTALL_DIR=$APP_DIR/python UV_CACHE_DIR=$APP_DIR/cache/uv UV_NO_CONFIG=1
	"$UV" python install --quiet "$(manifest OPENBB_PYTHON_VERSION)"
	rm -rf "$venv"
	"$UV" venv --quiet --python "$(manifest OPENBB_PYTHON_VERSION)" "$venv"
	for w in "$REL"/openbb/wheels/*.whl; do
		[[ $(basename "$w") == openbb_core-* ]] && w="${w}[pandas]"
		wheels+=("$w")
	done
	"$UV" pip install --quiet --python "$venv/bin/python" --find-links "$REL/openbb/wheels" \
		"${wheels[@]}" "openbb-yfinance==$(manifest OPENBB_YFINANCE_VERSION)"
	(cd "$venv" && OPENBB_AUTO_BUILD=false "$venv/bin/python" -c "import openbb; openbb.build()" >/dev/null)
	printf '%s\n' "$RELEASE_VERSION" >"$stamp"
}

sed_escape() { printf '%s' "$1" | sed 's/[\\&|]/\\&/g'; }

render() {
	sed -e "s|@CONFIG_DIR@|$(sed_escape "$CONF_DIR")|g" \
		-e "s|@DATA_DIR@|$(sed_escape "$DATA_DIR")|g" \
		-e "s|@PORTAL_PORT@|$PORT|g" \
		-e "s|@GRAFANA_ADMIN_PASSWORD@|${GRAFANA_PASSWORD:-}|g" "$1" >"$2"
}

# User-editable files are never overwritten: a changed default is written next to them as .new.
keep_user_file() {
	local src=$1 dst=$2
	if [[ ! -f $dst ]]; then
		cp "$src" "$dst"
	elif ! cmp -s "$src" "$dst"; then
		cp "$src" "$dst.new"
		warn "kept your $dst; the new default is in $dst.new"
	fi
}

replace_dir() {
	rm -rf "$2"
	mkdir -p "$(dirname "$2")"
	cp -R "$1" "$2"
}

install_config() {
	local c=$REL/config pw_file=$CONF_DIR/grafana-admin-password
	log "Writing configuration to $CONF_DIR"
	umask 077
	mkdir -p "$CONF_DIR"/{prometheus,alertmanager,grafana/provisioning/datasources,grafana/provisioning/dashboards,stock_exporter} \
		"$DATA_DIR"/{prometheus,alertmanager,grafana,kite,caddy}

	keep_user_file "$c/prometheus/prometheus.yml" "$CONF_DIR/prometheus/prometheus.yml"
	keep_user_file "$c/alertmanager/alertmanager.yml" "$CONF_DIR/alertmanager/alertmanager.yml"
	keep_user_file "$c/stock_exporter/watchlist.yml" "$CONF_DIR/stock_exporter/watchlist.yml"
	keep_user_file "$c/Caddyfile" "$CONF_DIR/Caddyfile"

	replace_dir "$c/prometheus/rules" "$CONF_DIR/prometheus/rules"
	replace_dir "$c/alertmanager/templates" "$CONF_DIR/alertmanager/templates"
	replace_dir "$c/grafana/dashboards" "$CONF_DIR/grafana/dashboards"
	replace_dir "$c/stock_exporter/textfile" "$CONF_DIR/stock_exporter/textfile"
	cp "$c/stock_exporter/holidays.yml" "$CONF_DIR/stock_exporter/holidays.yml"
	cp "$c/grafana/provisioning/datasources/datasources.yml" "$CONF_DIR/grafana/provisioning/datasources/"
	render "$c/grafana/provisioning/dashboards/dashboards.yml" "$CONF_DIR/grafana/provisioning/dashboards/dashboards.yml"

	# Grafana applies the admin password only when it creates its database.
	if [[ ! -f $CONF_DIR/grafana/grafana.ini ]]; then
		GRAFANA_PASSWORD=$(head -c 48 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')
		GRAFANA_PASSWORD=${GRAFANA_PASSWORD:0:20}
		printf '%s\n' "$GRAFANA_PASSWORD" >"$pw_file"
		render "$c/grafana/grafana.ini" "$CONF_DIR/grafana/grafana.ini"
	fi

	cat >"$CONF_DIR/stockobs.env" <<EOF
# Read by the stockobs-* systemd user services.
STOCKOBS_BIND=$BIND
STOCKOBS_PORT=$PORT
EOF
	umask 022
}

unit() {
	local name=$1 desc=$2 exec=$3 env=${4:-}
	cat >"$UNIT_DIR/stockobs-$name.service" <<EOF
[Unit]
Description=Stock Observability: $desc
PartOf=stockobs.target

[Service]
EnvironmentFile=$CONF_DIR/stockobs.env
${env}ExecStart=$exec
Restart=on-failure
RestartSec=5
TimeoutStopSec=60

[Install]
WantedBy=stockobs.target
EOF
}

install_units() {
	local prom=$1 am=$2 grafana=$3 caddy=$4 c=$CONF_DIR d=$DATA_DIR
	log "Installing systemd user services"
	mkdir -p "$UNIT_DIR"
	unit prometheus Prometheus "\"$prom/prometheus\" \"--config.file=$c/prometheus/prometheus.yml\" \"--storage.tsdb.path=$d/prometheus\" --web.listen-address=127.0.0.1:9090 --web.external-url=http://localhost:\${STOCKOBS_PORT}/prometheus/ --web.route-prefix=/prometheus --web.enable-lifecycle"
	unit alertmanager Alertmanager "\"$am/alertmanager\" \"--config.file=$c/alertmanager/alertmanager.yml\" \"--storage.path=$d/alertmanager\" --web.listen-address=127.0.0.1:9093 --web.external-url=http://localhost:\${STOCKOBS_PORT}/alertmanager/ --web.route-prefix=/alertmanager --cluster.listen-address="
	unit grafana Grafana "\"$grafana/bin/grafana\" server \"--homepath=$grafana\" \"--config=$c/grafana/grafana.ini\"" \
		"WorkingDirectory=$grafana
"
	unit exporter "Kite stock exporter" "\"$APP_DIR/bin/stock_exporter\" --web.listen-address=127.0.0.1:9101 --feed.source=kite \"--kite.credentials-file=$c/kite_credentials.json\" \"--kite.post-login-redirect=/settings?tab=data-sources&source=kite\" \"--kite.token-file=$d/kite/kite_session.json\" \"--stock.watchlist-file=$c/stock_exporter/watchlist.yml\" \"--market.holidays-file=$c/stock_exporter/holidays.yml\" \"--collector.textfile.directory=$c/stock_exporter/textfile\""
	unit openbb "OpenBB Platform API" "\"$APP_DIR/openbb/bin/openbb-api\"" \
		"Environment=OPENBB_API_HOST=127.0.0.1 OPENBB_API_PORT=6900 OPENBB_AUTO_BUILD=false PYTHONUNBUFFERED=1
WorkingDirectory=$APP_DIR/openbb
"
	unit home "Stock Home" "\"$APP_DIR/bin/stock_home\" --web.listen-address=127.0.0.1:8081 --prometheus.url=http://127.0.0.1:9090/prometheus --alertmanager.url=http://127.0.0.1:9093/alertmanager --grafana.url=http://127.0.0.1:3000/grafana \"--market.holidays-file=$c/stock_exporter/holidays.yml\" --openbb.url=http://127.0.0.1:6900 --exporter.url=http://127.0.0.1:9101"
	unit portal "Portal (Caddy)" "\"$caddy/caddy\" run \"--config=$c/Caddyfile\" --adapter=caddyfile" \
		"Environment=\"XDG_DATA_HOME=$d/caddy\" \"XDG_CONFIG_HOME=$d/caddy\"
"
	cat >"$UNIT_DIR/stockobs.target" <<EOF
[Unit]
Description=Stock Observability

[Install]
WantedBy=default.target
EOF
	systemctl --user daemon-reload
	systemctl --user enable --quiet stockobs.target "${SERVICES[@]/#/stockobs-}"
}

stop_services() {
	[[ -f $UNIT_DIR/stockobs.target ]] || return 0
	log "Stopping running services"
	systemctl --user stop stockobs.target "${SERVICES[@]/#/stockobs-}" 2>/dev/null || true
}

check_ports() {
	local p busy=()
	command -v ss >/dev/null || return 0
	for p in 9090 9093 3000 9101 6900 8081 "$PORT"; do
		[[ -z $(ss -Hltn "sport = :$p") ]] || busy+=("$p")
	done
	((${#busy[@]} == 0)) || die "ports already in use: ${busy[*]} (stop the programs using them and re-run)"
}

wait_ready() {
	local name=$1 url=$2 i
	for ((i = 0; i < 90; i++)); do
		# Any HTTP answer counts for OpenBB, which has no health endpoint.
		if [[ $name == openbb ]]; then
			curl -s -o /dev/null --max-time 2 "$url" && return 0
		else
			curl -fs -o /dev/null --max-time 2 "$url" && return 0
		fi
		sleep 2
	done
	warn "$name is not ready yet; check: journalctl --user -u stockobs-$name"
}

start_services() {
	log "Starting services"
	systemctl --user start stockobs.target
	wait_ready prometheus http://127.0.0.1:9090/prometheus/-/ready
	wait_ready alertmanager http://127.0.0.1:9093/alertmanager/-/ready
	wait_ready grafana http://127.0.0.1:3000/grafana/api/health
	wait_ready exporter http://127.0.0.1:9101/
	wait_ready home http://127.0.0.1:8081/healthz
	wait_ready openbb http://127.0.0.1:6900/
	wait_ready portal "http://127.0.0.1:$PORT/healthz"

	if [[ $(loginctl show-user "$USER" -p Linger --value 2>/dev/null) != yes ]] &&
		! loginctl enable-linger "$USER" 2>/dev/null; then
		warn "services stop when you log out; to keep them running run: sudo loginctl enable-linger $USER"
	fi
}

prune_vendor() {
	local keep=("$@") d k
	for d in "$APP_DIR"/vendor/*; do
		[[ -d $d ]] || continue
		for k in "${keep[@]}"; do
			[[ $d == "$k" ]] && continue 2
		done
		rm -rf "$d"
	done
}

# open_setup opens the Kite data source settings when a desktop browser is available.
open_setup() {
	local url="http://localhost:$PORT/settings?tab=data-sources&source=kite"
	((OPEN_SETUP && START && !KITE_CONFIGURED)) || return 0
	[[ -n ${DISPLAY:-}${WAYLAND_DISPLAY:-} || -n ${WSL_DISTRO_NAME:-} ]] || return 0
	if command -v wslview >/dev/null; then
		wslview "$url" >/dev/null 2>&1 &
	elif command -v xdg-open >/dev/null; then
		xdg-open "$url" >/dev/null 2>&1 &
	fi
}

summary() {
	local url=http://localhost:$PORT kite
	if ((KITE_CONFIGURED)); then
		kite="Kite Connect: log in once per trading day at $url/exporter/login
  (your app's redirect URL must be http://127.0.0.1:$PORT/exporter/kite/callback)"
	else
		kite="Next step: connect your Zerodha account in Settings → Data Sources at $url"
	fi
	cat <<EOF

Stock Observability $RELEASE_VERSION is installed.

  Home          $url/
  Grafana       $url/grafana/   (user admin, password in $CONF_DIR/grafana-admin-password)
  Prometheus    $url/prometheus/
  Alertmanager  $url/alertmanager/

$kite

Manage:  systemctl --user {status|restart|stop} stockobs.target
Logs:    journalctl --user -u 'stockobs-*' -f
Config:  $CONF_DIR
Remove:  curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | bash -s -- --uninstall
EOF
}

do_install() {
	preflight
	kite_credentials
	if [[ -f $CONF_DIR/stockobs.env ]]; then
		[[ -n $BIND ]] || BIND=$(sed -n 's/^STOCKOBS_BIND=//p' "$CONF_DIR/stockobs.env")
		[[ -n $PORT ]] || PORT=$(sed -n 's/^STOCKOBS_PORT=//p' "$CONF_DIR/stockobs.env")
	fi
	BIND=${BIND:-127.0.0.1}
	PORT=${PORT:-8080}

	TMP=$(mktemp -d)
	trap 'rm -rf "$TMP"' EXIT
	download_release
	log "Installing Stock Observability $RELEASE_VERSION into $APP_DIR"
	local prom am grafana caddy uv_dir
	prom=$(vendor prometheus)
	am=$(vendor alertmanager)
	grafana=$(vendor grafana)
	caddy=$(vendor caddy)
	uv_dir=$(vendor uv)
	UV=$uv_dir/uv

	stop_services
	check_ports
	mkdir -p "$APP_DIR/bin"
	install -m 0755 "$REL/bin/stock_exporter" "$REL/bin/stock_home" "$APP_DIR/bin/"
	install_openbb
	install_config
	install_units "$prom" "$am" "$grafana" "$caddy"
	prune_vendor "$prom" "$am" "$grafana" "$caddy" "$uv_dir"

	if ((START)); then
		start_services
	fi
	summary
	open_setup
}

do_uninstall() {
	need systemctl
	log "Removing Stock Observability services"
	systemctl --user disable --now stockobs.target "${SERVICES[@]/#/stockobs-}" 2>/dev/null || true
	rm -f "$UNIT_DIR"/stockobs.target "$UNIT_DIR"/stockobs-*.service
	systemctl --user daemon-reload 2>/dev/null || true
	rm -rf "${APP_DIR:?}"/{bin,vendor,openbb,python,cache}
	if ((PURGE)); then
		log "Deleting config and data"
		rm -rf "${APP_DIR:?}" "${CONF_DIR:?}"
	else
		log "Kept config in $CONF_DIR and data in $DATA_DIR (use --purge to delete them)"
	fi
}

main() {
	parse_args "$@"
	case $ACTION in
	install) do_install ;;
	uninstall) do_uninstall ;;
	esac
}

# Everything runs from main, so a partially downloaded script does nothing.
main "$@"

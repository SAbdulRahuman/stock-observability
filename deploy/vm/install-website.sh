#!/usr/bin/env bash
# Installs or upgrades stock_website on the VM from a `make -C website dist` tarball.
# Usage: deploy/vm/install-website.sh ubuntu@<vm-ip> website/dist/stock_website-<ver>.linux-amd64.tar.gz [ssh-key]
set -euo pipefail

host=${1:?usage: $0 user@host tarball [ssh-key]}
tarball=${2:?usage: $0 user@host tarball [ssh-key]}
ssh_opts=()
[[ -n ${3:-} ]] && ssh_opts=(-i "$3")

name=$(basename "$tarball" | sed -E 's/\.linux-[a-z0-9]+\.tar\.gz$//')
scp "${ssh_opts[@]}" "$tarball" "$host:/tmp/$name.tar.gz"
ssh "${ssh_opts[@]}" "$host" sudo -n bash -s -- "$name" <<'EOF'
set -euo pipefail
name=$1
tmp=$(mktemp -d)
tar -C "$tmp" -xzf "/tmp/$name.tar.gz"
install -d -m 0755 /opt/stock-observability/bin /etc/stock-observability
install -m 0755 "$tmp/$name/stock_website" /opt/stock-observability/bin/stock_website
install -m 0644 "$tmp/$name/stock-website.service" /etc/systemd/system/stock-website.service
# Keep an existing env file: it holds the domain and URLs for this VM.
[[ -f /etc/stock-observability/stock-website.env ]] ||
  install -m 0644 "$tmp/$name/stock-website.env.example" /etc/stock-observability/stock-website.env
rm -rf "$tmp" "/tmp/$name.tar.gz"
systemctl daemon-reload
systemctl enable stock-website.service
systemctl restart stock-website.service
systemctl --no-pager --lines=5 status stock-website.service
EOF

#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
mode=${1:-user}
case "$mode" in
    user) compose=(docker compose -f docker-compose.yml) ;;
    admin) compose=(docker compose -f docker-compose.admin.yml) ;;
    *) echo "Usage: $0 [user|admin]" >&2; exit 2 ;;
esac

"${compose[@]}" --profile tls run --rm certbot renew
"${compose[@]}" exec -T nginx nginx -s reload
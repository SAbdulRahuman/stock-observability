#!/bin/sh
set -eu

case "${1:-}" in
    issue)
        [ -n "${TLS_EMAIL:-}" ] || { echo "Set TLS_EMAIL before requesting a certificate." >&2; exit 2; }
        set -- certonly --webroot --webroot-path /var/www/certbot \
            --cert-name stock-observability --non-interactive --agree-tos \
            --email "$TLS_EMAIL" --keep-until-expiring
        domain_count=0
        for domain in ${CERT_DOMAINS:-}; do
            case "$domain" in
                *[!A-Za-z0-9.-]*) echo "Invalid certificate domain: $domain" >&2; exit 2 ;;
            esac
            set -- "$@" -d "$domain"
            domain_count=$((domain_count + 1))
        done
        [ "$domain_count" -gt 0 ] || { echo "Set CERT_DOMAINS to one or more public DNS names." >&2; exit 2; }
        certbot "$@"
        sh /scripts/install-certificate.sh
        ;;
    renew)
        certbot renew --deploy-hook "sh /scripts/install-certificate.sh"
        ;;
    *)
        echo "Usage: certbot.sh issue|renew" >&2
        exit 2
        ;;
esac
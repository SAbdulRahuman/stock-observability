#!/bin/sh
set -eu

certificate=/etc/letsencrypt/live/stock-observability
cp "$certificate/fullchain.pem" /certs/fullchain.pem
cp "$certificate/privkey.pem" /certs/privkey.pem
chown 101:101 /certs/fullchain.pem /certs/privkey.pem
chmod 644 /certs/fullchain.pem
chmod 640 /certs/privkey.pem
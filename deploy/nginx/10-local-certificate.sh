#!/bin/sh
set -eu

if [ ! -s /etc/nginx/certs/fullchain.pem ] || [ ! -s /etc/nginx/certs/privkey.pem ]; then
    openssl req -x509 -nodes -days 30 -newkey rsa:2048 \
        -keyout /etc/nginx/certs/privkey.pem \
        -out /etc/nginx/certs/fullchain.pem \
        -subj '/CN=localhost' \
        -addext 'subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1'
fi
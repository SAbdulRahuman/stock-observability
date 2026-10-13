ARG NGINX_BASE_IMAGE=nginxinc/nginx-unprivileged:1.28-alpine-slim
FROM ${NGINX_BASE_IMAGE}

USER root
RUN apk add --no-cache openssl \
    && mkdir -p /etc/nginx/certs /etc/nginx/auth \
    && chown -R 101:101 /etc/nginx/certs /etc/nginx/auth
COPY 10-local-certificate.sh /docker-entrypoint.d/10-local-certificate.sh
RUN chmod 755 /docker-entrypoint.d/10-local-certificate.sh
USER nginx
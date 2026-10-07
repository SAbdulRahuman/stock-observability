# OpenBB Platform API server for OpenShift. Build context is a `git archive` of
# OpenBB/OpenBB (see `make docker-build-openbb`), so the submodule stays untouched.
ARG PYTHON_IMAGE=python:3.11-slim

FROM ${PYTHON_IMAGE} AS builder
ARG TLS_VERIFY=true
ARG OPENBB_YFINANCE_VERSION=2.0.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

# Skipped: bls, ecb, eia, finra, imf and oecd regenerate catalogs from their APIs at
# build time (eia needs a key); cli, devtools and econometrics are not served by the API.
ARG OPENBB_PACKAGES="\
    core[pandas] \
    obbject_extensions/charting \
    extensions/platform_api extensions/equity extensions/news extensions/etf extensions/index \
    extensions/derivatives extensions/crypto extensions/currency extensions/economy \
    extensions/fixedincome extensions/commodity extensions/technical extensions/quantitative \
    providers/cboe providers/nasdaq providers/sec providers/tmx providers/fred \
    providers/federal_reserve providers/government_us providers/cftc providers/famafrench \
    providers/jodi providers/deribit"

# yfinance is not in the develop checkout; it adds NSE/BSE (.NS/.BO) coverage.
RUN if [ "$TLS_VERIFY" = "false" ]; then \
        export PIP_TRUSTED_HOST="pypi.org files.pythonhosted.org" \
               UV_INSECURE_HOST="pypi.org files.pythonhosted.org"; \
    fi \
    && pip install --no-cache-dir uv \
    && cd openbb_platform \
    && python -m uv pip install --python "$(command -v python)" \
        $(for p in $OPENBB_PACKAGES; do printf -- '--editable %s ' "$p"; done) \
        "openbb-yfinance==${OPENBB_YFINANCE_VERSION}" \
    && OPENBB_AUTO_BUILD=false python -c "import openbb; openbb.build(verbose=True)"

FROM ${PYTHON_IMAGE}

# OpenShift runs an arbitrary UID in group 0: HOME must be group-writable.
RUN useradd --uid 1001 --gid 0 --create-home --home-dir /home/openbb openbb \
    && chmod -R g=u /home/openbb

COPY --from=builder /usr/local /usr/local
COPY --from=builder /app /app

USER 1001
# No OPENBB_CONFIG: its [launcher] host would override OPENBB_API_HOST.
ENV HOME=/home/openbb \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    OPENBB_AUTO_BUILD=false \
    OPENBB_API_HOST=0.0.0.0 \
    OPENBB_API_PORT=6900

WORKDIR /app/openbb_platform
EXPOSE 6900
CMD ["openbb-api"]

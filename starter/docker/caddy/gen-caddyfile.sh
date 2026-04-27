#!/bin/sh
set -eu

DOMAIN="${1:?Usage: gen-caddyfile.sh DOMAIN TLS_CONFIG [--dev]}"
TLS_CONFIG="${2:?Usage: gen-caddyfile.sh DOMAIN TLS_CONFIG [--dev]}"
DEV_MODE="${3:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ "$DEV_MODE" = "--dev" ]; then
  cat > "${SCRIPT_DIR}/Caddyfile" <<EOF
{
	admin off
	log reverse-proxy {
		level ERROR
		include http.handlers.reverse_proxy
	}
	log {
		exclude http.handlers.reverse_proxy
	}
}

${DOMAIN} {
	${TLS_CONFIG}

	@sse path /__air_internal/*
	reverse_proxy @sse app:8080 {
		flush_interval -1
	}

	reverse_proxy app:8080
}
EOF
else
  cat > "${SCRIPT_DIR}/Caddyfile" <<EOF
{
	admin off
}

${DOMAIN} {
	${TLS_CONFIG}

	reverse_proxy app:8080
}
EOF
fi

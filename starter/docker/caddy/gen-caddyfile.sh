#!/bin/sh
set -eu

DOMAIN="${1:?Usage: gen-caddyfile.sh DOMAIN TLS_CONFIG [--dev]}"
TLS_CONFIG="${2:?Usage: gen-caddyfile.sh DOMAIN TLS_CONFIG [--dev]}"
DEV_MODE="${3:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

case "$DOMAIN" in
  *[!a-zA-Z0-9.:_-]*) echo "ERROR: DOMAIN='$DOMAIN' contains invalid characters" >&2; exit 1 ;;
esac

case "$TLS_CONFIG" in
  *[!a-zA-Z0-9./:_\ -]*) echo "ERROR: TLS_CONFIG contains invalid characters" >&2; exit 1 ;;
esac

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

	# Strip inbound X-Forwarded-Host — always rebuilt from the Host header.
	# For Cloudflare deployments, add: trusted_proxies cloudflare
	# inside the reverse_proxy block below.
	reverse_proxy app:8080 {
		header_up -X-Forwarded-Host
	}
}
EOF
fi

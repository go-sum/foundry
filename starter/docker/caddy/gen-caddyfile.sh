#!/bin/sh
set -eu

DOMAIN="${1:?Usage: gen-caddyfile.sh DOMAIN TLS_CONFIG}"
TLS_CONFIG="${2:?Usage: gen-caddyfile.sh DOMAIN TLS_CONFIG}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cat > "${SCRIPT_DIR}/Caddyfile" <<EOF
{
	admin off
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

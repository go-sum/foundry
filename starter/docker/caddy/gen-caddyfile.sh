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

	reverse_proxy app:8080
}
EOF

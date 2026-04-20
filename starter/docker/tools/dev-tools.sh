#!/usr/bin/env bash
set -eu

# ── Architecture ─────────────────────────────────────────────────────────────
case "${TARGETARCH}" in
  amd64) TW_ARCH="x64" ;;
  arm64) TW_ARCH="arm64" ;;
  *)     echo "unsupported: ${TARGETARCH}" >&2; exit 1 ;;
esac

# ── System packages ──────────────────────────────────────────────────────────
apt-get update && apt-get install -y --no-install-recommends libgit2-dev pkgconf
rm -rf /var/lib/apt/lists/*

# ── Tailwind CSS ─────────────────────────────────────────────────────────────
curl -fsSLo /usr/local/bin/tailwindcss \
  "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-linux-${TW_ARCH}"
chmod +x /usr/local/bin/tailwindcss

# ── Air (hot-reload) ─────────────────────────────────────────────────────────
go install github.com/air-verse/air@v${AIR_VERSION}

# ── golangci-lint ─────────────────────────────────────────────────────────────
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
  | sh -s -- -b /usr/local/bin v${GOLANGCI_LINT_VERSION}

# ── mkcert (local TLS certificate authority) ──────────────────────────────────
curl -fsSLo /usr/local/bin/mkcert \
  "https://github.com/FiloSottile/mkcert/releases/download/v${MKCERT_VERSION}/mkcert-v${MKCERT_VERSION}-linux-${TARGETARCH}"
chmod +x /usr/local/bin/mkcert

#!/usr/bin/env bash
set -eu

# ── Architecture ─────────────────────────────────────────────────────────────
case "${TARGETARCH}" in
  amd64) TW_ARCH="x64"; HUGO_ARCH="amd64" ;;
  arm64) TW_ARCH="arm64"; HUGO_ARCH="arm64" ;;
  *)     echo "unsupported: ${TARGETARCH}" >&2; exit 1 ;;
esac

# ── Tailwind CSS ─────────────────────────────────────────────────────────────
curl -fsSLo /usr/local/bin/tailwindcss \
  "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-linux-${TW_ARCH}"
chmod +x /usr/local/bin/tailwindcss

# ── Hugo (extended) ──────────────────────────────────────────────────────────
curl -fsSL \
  "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-${HUGO_ARCH}.tar.gz" \
  | tar -xzf - -C /usr/local/bin hugo
chmod +x /usr/local/bin/hugo

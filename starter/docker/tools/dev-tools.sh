#!/usr/bin/env bash
set -eu

# ── Air (hot-reload) ─────────────────────────────────────────────────────────
go install github.com/air-verse/air@v${AIR_VERSION}

# ── golangci-lint ─────────────────────────────────────────────────────────────
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
  | sh -s -- -b /usr/local/bin v${GOLANGCI_LINT_VERSION}

# ── mkcert (local TLS certificate authority) ──────────────────────────────────
# NOTE: FiloSottile/mkcert does not publish an official SHA256 checksums file.
# After upgrading MKCERT_VERSION, verify the binary manually:
#   sha256sum /usr/local/bin/mkcert
# and cross-check against the binary published:
curl -fsSLo /usr/local/bin/mkcert \
  "https://github.com/FiloSottile/mkcert/releases/download/v${MKCERT_VERSION}/mkcert-v${MKCERT_VERSION}-linux-${TARGETARCH}"
chmod +x /usr/local/bin/mkcert

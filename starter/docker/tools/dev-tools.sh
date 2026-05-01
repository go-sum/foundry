#!/usr/bin/env bash
set -eu

# ── Air (hot-reload) ─────────────────────────────────────────────────────────
go install github.com/air-verse/air@v${AIR_VERSION}

# ── golangci-lint ─────────────────────────────────────────────────────────────
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
  | sh -s -- -b /usr/local/bin v${GOLANGCI_LINT_VERSION}

# ── mkcert (local TLS certificate authority) ──────────────────────────────────
# Install from the pinned Go module version instead of an unsigned release asset.
# This relies on Go module resolution and checksum verification, and keeps the
# mkcert trust boundary in the dev tools image rather than the Caddy runtime.
GOBIN=/usr/local/bin go install filippo.io/mkcert@v${MKCERT_VERSION}

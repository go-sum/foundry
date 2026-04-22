#!/usr/bin/env bash
set -eu

# ── System packages (libgit2 for foundry CLI subtree operations) ─────────────
apt-get update && apt-get install -y --no-install-recommends libgit2-dev pkgconf
rm -rf /var/lib/apt/lists/*

# ── Air (hot-reload) ─────────────────────────────────────────────────────────
go install github.com/air-verse/air@v${AIR_VERSION}

# ── golangci-lint ─────────────────────────────────────────────────────────────
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
  | sh -s -- -b /usr/local/bin v${GOLANGCI_LINT_VERSION}

# ── mkcert (local TLS certificate authority) ──────────────────────────────────
curl -fsSLo /usr/local/bin/mkcert \
  "https://github.com/FiloSottile/mkcert/releases/download/v${MKCERT_VERSION}/mkcert-v${MKCERT_VERSION}-linux-${TARGETARCH}"
chmod +x /usr/local/bin/mkcert

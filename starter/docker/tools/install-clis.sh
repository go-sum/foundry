#!/usr/bin/env bash
set -eu
export GONOSUMDB='*'
export GOPRIVATE='github.com/go-sum/foundry'

# ── Assets CLI ────────────────────────────────────────────────────────────────
if [ -n "${ASSETS_CLI_VERSION:-}" ]; then
  echo "installing assets CLI v${ASSETS_CLI_VERSION}..."
  go install "github.com/go-sum/foundry/pkg/assets/cli@v${ASSETS_CLI_VERSION}"
  mv "${GOPATH}/bin/cli" /usr/local/bin/assets
else
  echo "skipping assets CLI (ASSETS_CLI_VERSION not set)"
fi

# ── Docs CLI ──────────────────────────────────────────────────────────────────
if [ -n "${DOCS_CLI_VERSION:-}" ]; then
  echo "installing docs CLI v${DOCS_CLI_VERSION}..."
  go install "github.com/go-sum/foundry/pkg/docs/cli@v${DOCS_CLI_VERSION}"
  mv "${GOPATH}/bin/cli" /usr/local/bin/docs
else
  echo "skipping docs CLI (DOCS_CLI_VERSION not set)"
fi

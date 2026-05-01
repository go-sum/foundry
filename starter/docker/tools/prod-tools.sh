#!/usr/bin/env bash
set -eu

# ── Architecture + checksums ──────────────────────────────────────────────────
# Checksums sourced from official release files:
#   tailwindcss: sha256sums.txt  (github.com/tailwindlabs/tailwindcss/releases)
#   hugo:        hugo_<ver>_checksums.txt  (github.com/gohugoio/hugo/releases)
# Update both TAILWIND_VERSION/.versions AND these hashes on every version bump.
case "${TARGETARCH}" in
  amd64)
    TW_ARCH="x64"
    HUGO_ARCH="amd64"
    TW_SHA256="c4062128a4b7a0450f0f0980bc4fb71afe1567a6bcf0e7edbf345bbab0ee3f64"
    HUGO_SHA256="8dc6e2d7c7c1a3ecf7abf756068af1b30c8197108ae0cc7ffd83ef2b88f0c26d"
    ;;
  arm64)
    TW_ARCH="arm64"
    HUGO_ARCH="arm64"
    TW_SHA256="b45ed3af9109935ccbd062732e4fa31b2984cdaa1b713fa8b0157e1864ecefc1"
    HUGO_SHA256="dc85bc0101c03a8eb5238bbc1e44ed9ce586f4eb294696f9f8d8a6d4232934b5"
    ;;
  *) echo "unsupported: ${TARGETARCH}" >&2; exit 1 ;;
esac

# ── Tailwind CSS ─────────────────────────────────────────────────────────────
curl -fsSLo /usr/local/bin/tailwindcss \
  "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-linux-${TW_ARCH}"
echo "${TW_SHA256}  /usr/local/bin/tailwindcss" | sha256sum -c -
chmod +x /usr/local/bin/tailwindcss

# ── Hugo (extended) ──────────────────────────────────────────────────────────
HUGO_TGZ="/tmp/hugo.tar.gz"
curl -fsSLo "${HUGO_TGZ}" \
  "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-${HUGO_ARCH}.tar.gz"
echo "${HUGO_SHA256}  ${HUGO_TGZ}" | sha256sum -c -
tar -xzf "${HUGO_TGZ}" -C /usr/local/bin hugo
rm "${HUGO_TGZ}"
chmod +x /usr/local/bin/hugo

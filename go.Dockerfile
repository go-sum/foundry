ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-bookworm
ARG HTMX_VERSION=2.0.4
ARG TAILWIND_VERSION=4.1.3
ARG TARGETARCH

# ────────────────────────────────────────────────────────────────── apt-get ───
RUN apt-get update && apt-get install -y --no-install-recommends curl sudo \
    && rm -rf /var/lib/apt/lists/*

# ───────────────────────────────────────────────────── https://taskfile.dev ───
RUN sh -c "$(curl -fsSL https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin

# ────────────────────────────────────────────────────────────── tailwindcss ───
RUN case "${TARGETARCH}" in \
      amd64) TW_ARCH="x64"   ;; \
      arm64) TW_ARCH="arm64" ;; \
      *)     echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
    && curl -fsSLo /usr/local/bin/tailwindcss \
         "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-linux-${TW_ARCH}" \
    && chmod +x /usr/local/bin/tailwindcss

# ───────────────────────────────────────────────────────────────────── htmx ───
RUN curl -fsSL \
    "https://unpkg.com/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" \
    -o /usr/local/lib/htmx.min.js

# Local replace directives (go.work) cannot be verified by the checksum database
ENV GONOSUMDB=*

EXPOSE 8080

CMD ["task", "dev"]

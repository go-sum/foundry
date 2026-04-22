# check=skip=InvalidDefaultArgInFrom
ARG GO_VERSION
ARG AIR_VERSION
ARG ASSETS_CLI_VERSION
ARG DOCS_CLI_VERSION
ARG GOLANGCI_LINT_VERSION
ARG HUGO_VERSION
ARG MKCERT_VERSION
ARG TAILWIND_VERSION

# ───────────────────────────────────────────────────── production_toolchain ───
FROM golang:${GO_VERSION}-bookworm AS production_toolchain
ARG HUGO_VERSION
ARG TAILWIND_VERSION
ARG TARGETARCH

ENV GOPATH=/root/go
ENV PATH="${GOPATH}/bin:/usr/local/go/bin:${PATH}"

RUN git config --global --add safe.directory /app

COPY prod-tools.sh /tmp/
RUN bash /tmp/prod-tools.sh && rm /tmp/prod-tools.sh

WORKDIR /app

# ──────────────────────────────────────────────────────── dev_toolchain ───
FROM production_toolchain AS dev_toolchain
ARG AIR_VERSION
ARG ASSETS_CLI_VERSION
ARG DOCS_CLI_VERSION
ARG GOLANGCI_LINT_VERSION
ARG MKCERT_VERSION
ARG TARGETARCH

COPY dev-tools.sh /tmp/
RUN bash /tmp/dev-tools.sh && rm /tmp/dev-tools.sh

COPY install-clis.sh /tmp/
RUN bash /tmp/install-clis.sh && rm /tmp/install-clis.sh

WORKDIR /app

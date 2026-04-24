# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG GO_VERSION
ARG AIR_VERSION
ARG ASSETS_CLI_VERSION
ARG DOCS_CLI_VERSION
ARG GOLANGCI_LINT_VERSION
ARG HUGO_VERSION
ARG MKCERT_VERSION
ARG TAILWIND_VERSION

# ──────────────────────────────────────────────────────── cli_builder ───
FROM golang:${GO_VERSION}-bookworm AS cli_builder
ARG TARGETARCH
ENV GONOSUMDB='*' GOPRIVATE='github.com/go-sum/*' GOWORK=off
WORKDIR /build

# ── assets CLI (self-contained, no replace directives) ──
COPY --from=workspace pkg/assets/ ./pkg/assets/
RUN cd pkg/assets && go mod download && CGO_ENABLED=0 go build -ldflags='-s -w' -o /usr/local/bin/assets ./cli

# ── db CLI (replace: github.com/go-sum/config => ../config) ──
COPY --from=workspace pkg/db/ ./pkg/db/
COPY --from=workspace pkg/config/ ./pkg/config/
RUN cd pkg/db && go mod download && CGO_ENABLED=0 go build -ldflags='-s -w' -o /usr/local/bin/db ./cli

# ── docs CLI (replace: github.com/go-sum/web => ../web) ──
COPY --from=workspace pkg/docs/ ./pkg/docs/
COPY --from=workspace pkg/web/ ./pkg/web/
RUN cd pkg/docs && go mod download && CGO_ENABLED=0 go build -ldflags='-s -w' -o /usr/local/bin/docs ./cli

# ── foundry CLI (requires CGO for libgit2/git2go) ──
RUN apt-get update && apt-get install -y --no-install-recommends libgit2-dev pkgconf && rm -rf /var/lib/apt/lists/*
COPY --from=workspace tools/ ./tools/
RUN cd tools && go mod download && CGO_ENABLED=1 go build -ldflags='-s -w' -o /usr/local/bin/foundry ./cmd/foundry

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

# ─────────────────────────────────────────── dev_toolchain_monorepo ───
FROM dev_toolchain AS dev_toolchain_monorepo
COPY --from=cli_builder /usr/local/bin/assets /usr/local/bin/assets
COPY --from=cli_builder /usr/local/bin/db /usr/local/bin/db
COPY --from=cli_builder /usr/local/bin/docs /usr/local/bin/docs
COPY --from=cli_builder /usr/local/bin/foundry /usr/local/bin/foundry

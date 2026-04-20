# check=skip=InvalidDefaultArgInFrom
ARG DRAGONFLY_VERSION
ARG GO_VERSION

# ────────────────────────────────────────────────────── dragonfly_toolchain ───
FROM docker.dragonflydb.io/dragonflydb/dragonfly:v${DRAGONFLY_VERSION} AS dragonfly_toolchain

# ────────────────────────────────────────────────────── healthcheck_builder ───
FROM golang:${GO_VERSION}-bookworm AS healthcheck_builder
WORKDIR /build
COPY healthcheck.go .
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /healthcheck healthcheck.go

# ──────────────────────────────────────────────────────── production_target ───
FROM cgr.dev/chainguard/wolfi-base:latest AS production_target

RUN apk update && apk add --no-cache \
    ca-certificates-bundle libgcc libstdc++ libunwind openssl tini zlib

COPY --from=dragonfly_toolchain /usr/local/bin/dragonfly /usr/local/bin/dragonfly
COPY --from=healthcheck_builder /healthcheck /usr/local/bin/healthcheck
COPY --chmod=755 entrypoint.sh /usr/local/bin/entrypoint.sh

RUN addgroup -g 999 dfly && adduser -u 999 -G dfly -D -H dfly && \
    chmod 755 /usr/local/bin/entrypoint.sh

EXPOSE 6379
USER dfly

ENTRYPOINT ["tini", "--", "entrypoint.sh"]
CMD ["dragonfly", "--logtostderr"]

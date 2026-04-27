# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
# ── Version pins ──────────────────────────────────────────────────────────────
ARG GO_VERSION
ARG HTMX_VERSION
ARG HUGO_VERSION
ARG TAILWIND_VERSION
ARG TOOLS_DEV_IMAGE=foundry-tools:dev
ARG TOOLS_PROD_IMAGE=foundry-tools:prod

FROM ${TOOLS_DEV_IMAGE} AS dev_toolchain
FROM ${TOOLS_PROD_IMAGE} AS production_toolchain

# ===================================================================== dev ====
# ─────────────────────────────────────────────────────────────── dev_target ───
FROM cgr.dev/chainguard/wolfi-base:latest AS dev_target

COPY --from=dev_toolchain /usr/local/go /usr/local/go
COPY --from=dev_toolchain /root/go/bin/air /usr/local/bin/air
COPY --from=dev_toolchain /usr/local/bin/tailwindcss /usr/local/bin/tailwindcss
COPY --from=dev_toolchain /usr/local/bin/assets /usr/local/bin/assets
COPY --from=dev_toolchain /usr/local/bin/db /usr/local/bin/db
COPY --from=dev_toolchain /usr/local/bin/docs /usr/local/bin/docs
COPY --from=dev_toolchain /usr/local/bin/hugo /usr/local/bin/hugo
RUN apk update && apk add --no-cache bash ca-certificates-bundle curl git libstdc++

ENV GOPATH=/root/go
ENV PATH="${GOPATH}/bin:/usr/local/go/bin:${PATH}"
ENV GONOSUMDB=*

RUN git config --global --add safe.directory /app

WORKDIR /app

CMD ["air", "-c", "docker/app/.air.toml"]

# =============================================================== production ===
# ─────────────────────────────────────────────────────────── assets_builder ───
# This stage mirrors `task build:assets` (.assets.yaml + assets CLI).
# Keep HTMX_VERSION and Tailwind invocation in sync with .assets.yaml and .versions.
FROM production_toolchain AS assets_builder
ARG HTMX_VERSION
ARG TAILWIND_VERSION
WORKDIR /app
COPY . .
RUN mkdir -p public/static/css public/static/js && \
    tailwindcss --input ./static/tailwind.css \
                --output ./public/static/css/app.css \
                --minify && \
    curl -fsSL \
      "https://unpkg.com/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" \
      -o ./public/static/js/htmx.min.js

# ────────────────────────────────────────────────────── healthcheck_builder ───
# healthcheck.go uses only stdlib — no workspace modules needed.
FROM production_toolchain AS healthcheck_builder
ARG GO_VERSION
WORKDIR /build
COPY docker/app/healthcheck.go main.go
RUN printf 'module healthcheck\ngo %s\n' "${GO_VERSION}" > go.mod && \
    CGO_ENABLED=0 go build -ldflags='-s -w' -o /healthcheck .

# ─────────────────────────────────────────────────────── production_builder ───
FROM cgr.dev/chainguard/wolfi-base:latest AS production_builder
COPY --from=production_toolchain /usr/local/go /usr/local/go
RUN apk update && apk add --no-cache ca-certificates-bundle git
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /app
# go.mod has no replace directives — pkg/ modules resolve from the monorepo.
COPY go.mod go.mod
COPY go.sum go.sum
# Private modules require GITHUB_ACCESS_TOKEN passed as a build secret.
RUN --mount=type=secret,id=github_token \
    TOKEN="$(cat /run/secrets/github_token 2>/dev/null)" && \
    if [ -n "${TOKEN}" ]; then \
      printf "machine github.com\nlogin x-access-token\npassword %s\n" "${TOKEN}" > /root/.netrc && \
      chmod 600 /root/.netrc; \
    fi && \
    GONOSUMDB='github.com/go-sum/*' GOPRIVATE='github.com/go-sum/foundry' go mod download
COPY cmd/ ./cmd/
COPY config/ ./config/
RUN rm -f config/*.development.yaml
COPY internal/ ./internal/
COPY static/ ./static/
ARG APP_VERSION
RUN GONOSUMDB='github.com/go-sum/*' GOPRIVATE='github.com/go-sum/foundry' \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${APP_VERSION}" -o /bin/server ./cmd/server

# ──────────────────────────────────────────────────────── production_target ───
FROM cgr.dev/chainguard/static:latest AS production_target
ARG APP_FOLDER=/opt/server
WORKDIR ${APP_FOLDER}
ENV APP_FOLDER=${APP_FOLDER} \
    PUBLIC_DIR=${APP_FOLDER}/public \
    PUBLIC_PREFIX=/public
COPY --from=production_builder  --chown=nonroot:nonroot /bin/server         ${APP_FOLDER}/server
COPY --from=healthcheck_builder --chown=nonroot:nonroot /healthcheck         /opt/healthcheck
COPY --from=assets_builder      --chown=nonroot:nonroot /app/public/         ${APP_FOLDER}/public/
COPY --from=production_builder  --chown=nonroot:nonroot /app/config/*.yaml  ${APP_FOLDER}/config/
USER nonroot
EXPOSE 8080
CMD ["/opt/server/server"]

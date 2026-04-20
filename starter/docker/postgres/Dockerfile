# ──────────────────────────────────────────────────────── production_target ───
FROM cgr.dev/chainguard/wolfi-base:latest AS production_target

ARG PG_VERSION

RUN addgroup -S postgres && adduser -S -G postgres -D -H postgres

RUN apk update && apk add --no-cache \
      postgresql-${PG_VERSION} \
      postgresql-${PG_VERSION}-client \
      postgresql-${PG_VERSION}-contrib \
      postgresql-${PG_VERSION}-oci-entrypoint \
      gosu

ENV PGDATA=/var/lib/postgresql/data
ENV LANG=C.UTF-8
ENV POSTGRES_INITDB_ARGS="--locale=C"

# Wolfi installs extension .so files in /usr/lib/postgresql<ver>/ but
# PostgreSQL expects them in /usr/lib/postgresql<ver>/lib/.
RUN mkdir -p /usr/lib/postgresql${PG_VERSION}/lib && \
    ln -sf /usr/lib/postgresql${PG_VERSION}/*.so /usr/lib/postgresql${PG_VERSION}/lib/

RUN mkdir -p /var/lib/postgresql/data /var/run/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql

VOLUME /var/lib/postgresql/data
EXPOSE 5432

ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]
CMD ["postgres"]

# ─────────────────────────────────────────────────────────────── dev_target ───
FROM production_target AS dev_target
COPY db/init-dev/ /var/lib/postgres/initdb/
RUN chown postgres:postgres /var/lib/postgres/initdb/*.sh && \
    chmod +x /var/lib/postgres/initdb/*.sh

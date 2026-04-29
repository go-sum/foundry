#!/bin/bash
set -e

# Create additional databases for tests.
# The main database ($POSTGRES_DB) is created by the entrypoint.
# Same credentials — simplifies development.

if [[ ! "$POSTGRES_DB" =~ ^[a-zA-Z0-9_]+$ ]]; then
    echo "ERROR: POSTGRES_DB='$POSTGRES_DB' contains invalid characters (only [a-zA-Z0-9_] allowed)" >&2
    exit 1
fi

echo "Creating database: ${POSTGRES_DB}_test"
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE "${POSTGRES_DB}_test"'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${POSTGRES_DB}_test')
    \gexec
EOSQL

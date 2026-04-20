#!/bin/sh
# Read KV_PASSWORD from Docker secret if available, then exec the main process.
if [ -f /run/secrets/KV_PASSWORD ]; then
  export DFLY_requirepass="$(cat /run/secrets/KV_PASSWORD)"
fi
exec "$@"

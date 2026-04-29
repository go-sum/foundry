#!/bin/sh
# Read KV_PASSWORD from Docker secret if available, then exec the main process.
# SECURITY: Dragonfly does not support a file-based password flag; the env var
# DFLY_requirepass is the only supported mechanism. The password is visible in
# /proc/<pid>/environ but the container namespace isolates it from external processes.
if [ -f /run/secrets/KV_PASSWORD ]; then
  export DFLY_requirepass="$(cat /run/secrets/KV_PASSWORD)"
fi
exec "$@"

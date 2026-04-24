#!/usr/bin/env bash
# libgit2@1.5 is no longer available in Homebrew (only 1.7+ are offered).
# git2go/v34 requires exactly v1.5.x and rejects newer versions at compile time.
#
# Split commands (push / release / status / deploy) run inside Docker automatically
# via the tasks in tools/Taskfile.yml. No local libgit2 installation is needed.
#
# To verify the Docker build environment is ready:
#   docker compose -f starter/docker-compose.dev.yml build foundry
echo "libgit2@1.5 is no longer available in Homebrew."
echo "Split commands route through Docker automatically — no local install required."
echo ""
echo "To build the Docker environment:"
echo "  docker compose -f starter/docker-compose.dev.yml build foundry"

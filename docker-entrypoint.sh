#!/bin/sh
set -e

# A platform-mounted volume at /app/storages is typically root-owned. Fix the
# ownership while we still have root, then drop to the unprivileged user so the
# gateway process itself never runs as root.
mkdir -p /app/storages/media
chown -R gateway:gateway /app/storages

exec su-exec gateway:gateway /app/waapi-gateway "$@"

#!/usr/bin/env sh
set -eu

if [ "$#" -ne 2 ]; then
  echo "Usage: restore-postgres.sh <backup-s3-key> <empty-target-database>" >&2
  exit 2
fi

exec "${WHATOMATE_BIN:-whatomate}" restore \
  -config "${WHATOMATE_CONFIG:-config.toml}" \
  -key "$1" \
  -target-database "$2"

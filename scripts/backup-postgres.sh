#!/usr/bin/env sh
set -eu

# Intended for cron or a platform scheduler. The binary writes only the S3 key
# to stdout; failures return a non-zero exit code for alerting.
exec "${WHATOMATE_BIN:-whatomate}" backup -config "${WHATOMATE_CONFIG:-config.toml}"

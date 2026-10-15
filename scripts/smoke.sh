#!/bin/sh
# Offline runtime verification. Never opens a browser or accesses H3yun.
set -eu
binary=${1:?Usage: sh scripts/smoke.sh /absolute/path/to/h3yun}
"$binary" version
"$binary" schema
"$binary" schema 'records create' --compact --format json
"$binary" records create --schema TEST_SCHEMA --data '{"F0000001":"offline test"}' --dry-run
"$binary" records list --schema TEST_SCHEMA --limit 2 --dry-run
status=0
"$binary" records get --schema TEST_SCHEMA || status=$?
if [ "$status" -ne 2 ]; then
  printf '%s\n' 'FAILED: invalid arguments must return exit code 2' >&2
  exit 1
fi
printf '%s\n' 'Offline executable smoke checks passed.'

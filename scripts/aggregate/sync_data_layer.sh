#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

exec ./music-garden sync-data-layer "$@"

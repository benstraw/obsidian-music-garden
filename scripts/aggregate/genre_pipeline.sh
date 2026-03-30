#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

./music-garden aggregate-genres
exec ./music-garden generate-genre-pages "$@"

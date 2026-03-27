#!/bin/zsh
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/../.." && pwd -P)"
cd "${REPO_DIR}"

exec ./music-garden genre-report "$@"

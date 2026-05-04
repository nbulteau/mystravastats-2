#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export FIXTURE_PATH="${FIXTURE_PATH:-$ROOT_DIR/test-fixtures/routes/strava-art-smoke.json}"

"$ROOT_DIR/scripts/manual-strava-art-smoke-check.sh"

#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-go}"
case "$MODE" in
  go|kotlin)
    ;;
  *)
    echo "Usage: $0 [go|kotlin]" >&2
    exit 64
    ;;
esac

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/docker-compose-$MODE.yml"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-mystravastats-smoke-$MODE}"
FRONT_URL="${FRONT_URL:-http://localhost}"
API_HEALTH_URL="${API_HEALTH_URL:-$FRONT_URL/api/health/details}"

cleanup() {
  docker compose -p "$COMPOSE_PROJECT_NAME" -f "$COMPOSE_FILE" down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_for_url() {
  local label="$1"
  local url="$2"
  local attempts="${3:-60}"

  for attempt in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null; then
      echo "$label is reachable at $url"
      return 0
    fi
    echo "Waiting for $label ($attempt/$attempts)..."
    sleep 2
  done

  docker compose -p "$COMPOSE_PROJECT_NAME" -f "$COMPOSE_FILE" ps
  docker compose -p "$COMPOSE_PROJECT_NAME" -f "$COMPOSE_FILE" logs --tail=200
  echo "$label did not become reachable at $url" >&2
  return 1
}

docker compose -p "$COMPOSE_PROJECT_NAME" -f "$COMPOSE_FILE" up --build -d
wait_for_url "Frontend" "$FRONT_URL" "${FRONT_WAIT_ATTEMPTS:-60}"
wait_for_url "API health through frontend proxy" "$API_HEALTH_URL" "${API_WAIT_ATTEMPTS:-60}"

if [[ "${RUN_STRAVA_ART_SMOKE:-false}" == "true" || "${RUN_STRAVA_ART_SMOKE:-0}" == "1" ]]; then
  echo "Running Strava Art smoke check through $FRONT_URL"
  BACKEND_URL="${STRAVA_ART_BACKEND_URL:-$FRONT_URL}" "$ROOT_DIR/scripts/smoke-strava-art.sh"
fi

echo "Docker smoke test passed for $MODE stack"

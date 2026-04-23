#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8080}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Ride}"
VALIDATION_YEAR="${VALIDATION_YEAR:-1990}"
ROUTE_TYPE="${ROUTE_TYPE:-RIDE}"
VARIANT_COUNT="${VARIANT_COUNT:-12}"

# Dense Rennes center loop (compact urban grid)
DENSE_START_LAT="${DENSE_START_LAT:-48.1175}"
DENSE_START_LNG="${DENSE_START_LNG:--1.6780}"
DENSE_SHAPE="${DENSE_SHAPE:-[[48.1175,-1.6780],[48.1215,-1.6740],[48.1190,-1.6660],[48.1120,-1.6640],[48.1080,-1.6700],[48.1100,-1.6780],[48.1175,-1.6780]]}"

# Rennes north-east outskirts loop (peri-rural roads)
RURAL_START_LAT="${RURAL_START_LAT:-48.157563}"
RURAL_START_LNG="${RURAL_START_LNG:--1.587309}"
RURAL_SHAPE="${RURAL_SHAPE:-[[48.157563,-1.587309],[48.1670,-1.5850],[48.1690,-1.6070],[48.1500,-1.6150],[48.1420,-1.5970],[48.157563,-1.587309]]}"

DENSE_EXPECTED_MODE="${DENSE_EXPECTED_MODE:-projected waypoints}"
RURAL_EXPECTED_MODE="${RURAL_EXPECTED_MODE:-road-first anchors}"

log() {
  printf '[route-shape-tuning-check] %s\n' "$1"
}

fail() {
  printf '[route-shape-tuning-check] ERROR: %s\n' "$1" >&2
  exit 1
}

require_bin() {
  local bin="$1"
  command -v "$bin" >/dev/null 2>&1 || fail "Missing required binary: $bin"
}

require_bin curl
require_bin jq

run_shape_case() {
  local case_name="$1"
  local start_lat="$2"
  local start_lng="$3"
  local shape_json="$4"
  local expected_mode="$5"

  local payload
  payload="$(jq -n \
    --argjson lat "$start_lat" \
    --argjson lng "$start_lng" \
    --arg shape "$shape_json" \
    --arg routeType "$ROUTE_TYPE" \
    --argjson variantCount "$VARIANT_COUNT" \
    '{
      shapeInputType: "draw",
      shapeData: $shape,
      startPoint: { lat: $lat, lng: $lng },
      routeType: $routeType,
      variantCount: $variantCount
    }'
  )"

  local endpoint
  endpoint="${BACKEND_URL}/api/routes/generate/shape?activityType=${ACTIVITY_TYPE}&year=${VALIDATION_YEAR}"
  local response_json
  response_json="$(curl -sf -X POST "$endpoint" -H 'Content-Type: application/json' -d "$payload")" \
    || fail "Shape request failed for case ${case_name}."

  local routes_count
  routes_count="$(echo "$response_json" | jq '.routes | length')"
  if [[ "$routes_count" -lt 1 ]]; then
    echo "$response_json" | jq '{routesCount: (.routes | length), diagnostics}'
    fail "No route returned for ${case_name}."
  fi

  local modes
  modes="$(echo "$response_json" | jq -r '[.routes[].reasons[]? | select(startswith("Shape mode: ")) | ltrimstr("Shape mode: ")] | unique')"
  if ! echo "$modes" | jq -e --arg expected "$expected_mode" '. | index($expected) != null' >/dev/null 2>&1; then
    echo "$response_json" | jq '{routesCount: (.routes | length), firstRoute: .routes[0]}'
    fail "Expected shape mode '${expected_mode}' not found for ${case_name}."
  fi

  local top_mode
  top_mode="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Shape mode: "))' | head -n 1)"
  local top_similarity
  top_similarity="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Shape similarity: "))' | head -n 1)"
  local drift_penalties
  drift_penalties="$(echo "$response_json" | jq -r '[.routes[].reasons[]? | select(startswith("Shape drift penalty: "))] | unique')"

  log "${case_name}: routes=${routes_count}, topMode='${top_mode:-n/a}', topSimilarity='${top_similarity:-n/a}'"
  echo "$response_json" | jq --arg case "$case_name" --arg expectedMode "$expected_mode" --argjson modes "$modes" --argjson drift "$drift_penalties" '{
    case: $case,
    expectedMode: $expectedMode,
    routesCount: (.routes | length),
    topRoute: {
      routeId: .routes[0].routeId,
      title: .routes[0].title,
      distanceKm: .routes[0].distanceKm,
      elevationGainM: .routes[0].elevationGainM,
      shapeScore: .routes[0].score.shape
    },
    availableModes: $modes,
    driftPenalties: $drift
  }'
}

log "Checking backend health at ${BACKEND_URL}/api/health/details"
health_json="$(curl -sf "${BACKEND_URL}/api/health/details")" || fail "Backend is not reachable."
routing_status="$(echo "$health_json" | jq -r '.routing.status // "unknown"')"
routing_reachable="$(echo "$health_json" | jq -r '.routing.reachable // false')"
log "Routing status=${routing_status} reachable=${routing_reachable}"
if [[ "$routing_status" != "up" || "$routing_reachable" != "true" ]]; then
  fail "Routing backend is not ready (status=${routing_status}, reachable=${routing_reachable})."
fi

log "Running dense urban scenario"
run_shape_case "dense-urban" "$DENSE_START_LAT" "$DENSE_START_LNG" "$DENSE_SHAPE" "$DENSE_EXPECTED_MODE"

log "Running rural/peri-rural scenario"
run_shape_case "rural-peri" "$RURAL_START_LAT" "$RURAL_START_LNG" "$RURAL_SHAPE" "$RURAL_EXPECTED_MODE"

log "Shape tuning validation passed for both scenarios."

#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Ride}"
FIXTURE_PATH="${FIXTURE_PATH:-test-fixtures/routes/strava-art-smoke.json}"
SHAPE_INPUT_TYPE="${SHAPE_INPUT_TYPE:-draw}"
CURL_MAX_TIME="${CURL_MAX_TIME:-15}"
ROUTE_TYPE="${ROUTE_TYPE:-RIDE}"
VARIANT_COUNT="${VARIANT_COUNT:-3}"
START_LAT="${START_LAT:-48.121}"
START_LNG="${START_LNG:--1.635}"
SHAPE_DATA="${SHAPE_DATA:-}"

log() {
  printf '[strava-art-smoke] %s\n' "$1"
}

fail() {
  printf '[strava-art-smoke] ERROR: %s\n' "$1" >&2
  exit 1
}

require_bin() {
  local bin="$1"
  command -v "$bin" >/dev/null 2>&1 || fail "Missing required binary: $bin"
}

require_bin curl
require_bin jq

log "Checking backend health at ${BACKEND_URL}/api/health/details"
HEALTH_JSON="$(curl -sS -f -m "$CURL_MAX_TIME" "${BACKEND_URL}/api/health/details")" || fail "Backend is not reachable."
ROUTING_STATUS="$(echo "$HEALTH_JSON" | jq -r '.routing.status // "unknown"')"
ROUTING_REACHABLE="$(echo "$HEALTH_JSON" | jq -r '.routing.reachable // false')"
log "Routing status=${ROUTING_STATUS} reachable=${ROUTING_REACHABLE}"

run_case() {
  local case_name="$1"
  local shape_input_type="$2"
  local shape_data="$3"
  local route_type="$4"
  local start_lat="$5"
  local start_lng="$6"
  local variant_count="$7"
  local request_id
  request_id="$(printf '%s' "$case_name" | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | tr -cd '[:alnum:]-_')"
  request_id="strava-art-smoke-${request_id:-case}"

  local request_body
  request_body="$(jq -n \
    --arg shapeInputType "$shape_input_type" \
    --arg shapeData "$shape_data" \
    --arg routeType "$route_type" \
    --argjson lat "$start_lat" \
    --argjson lng "$start_lng" \
    --argjson variantCount "$variant_count" \
    '{
      shapeInputType: $shapeInputType,
      shapeData: $shapeData,
      startPoint: { lat: $lat, lng: $lng },
      routeType: $routeType,
      variantCount: $variantCount
    }'
  )"

  local generate_endpoint="${BACKEND_URL}/api/routes/generate/shape?activityType=${ACTIVITY_TYPE}"
  log "Calling ${generate_endpoint} (${case_name})"
  local response_json
  response_json="$(curl -sS -f -m "$CURL_MAX_TIME" -X POST "$generate_endpoint" -H 'Content-Type: application/json' -H "X-Request-Id: ${request_id}" -d "$request_body")" \
    || fail "Strava Art generation call failed for ${case_name}."

  local routes_count
  routes_count="$(echo "$response_json" | jq '.routes | length')"
  if [[ "$routes_count" -lt 1 ]]; then
    echo "$response_json" | jq --arg caseName "$case_name" '{case: $caseName, routesCount: (.routes | length), diagnostics}'
    fail "No route returned for Strava Art smoke shape ${case_name}."
  fi

  local route_id
  route_id="$(echo "$response_json" | jq -r '.routes[0].routeId // empty')"
  if [[ -z "$route_id" ]]; then
    echo "$response_json" | jq --arg caseName "$case_name" '{case: $caseName, routesCount: (.routes | length), firstRoute: .routes[0]}'
    fail "First generated route has no routeId for ${case_name}."
  fi

  local encoded_route_id
  encoded_route_id="$(jq -nr --arg routeId "$route_id" '$routeId | @uri')"
  local gpx_endpoint="${BACKEND_URL}/api/routes/${encoded_route_id}/gpx"
  log "Exporting ${gpx_endpoint} (${case_name})"
  local gpx_payload
  gpx_payload="$(curl -sS -f -m "$CURL_MAX_TIME" "$gpx_endpoint" -H 'Accept: application/gpx+xml')" || fail "GPX export failed for routeId=${route_id}."
  if [[ "$gpx_payload" != *"<gpx"* || "$gpx_payload" != *"<trkpt"* ]]; then
    fail "GPX export did not contain expected GPX track points for ${case_name}."
  fi

  echo "$response_json" | jq --arg caseName "$case_name" '{
    case: $caseName,
    routesCount: (.routes | length),
    selectedRouteId: .routes[0].routeId,
    selectedDistanceKm: .routes[0].distanceKm,
    selectedElevationGainM: .routes[0].elevationGainM,
    artFit: .routes[0].score.shape,
    routeQuality: .routes[0].score.global,
    selectedShapeMode: ([.routes[0].reasons[]? | select(startswith("Shape mode:"))] | join("; ")),
    selectedProfile: ([.routes[0].reasons[]? | select(startswith("Selection profile:"))] | join("; ")),
    selectedPriority: ([.routes[0].reasons[]? | select(startswith("Selection priority:"))] | join("; ")),
    topRoutes: [
      .routes[] | {
        routeId,
        distanceKm,
        artFit: .score.shape,
        routeQuality: .score.global,
        shapeMode: ([.reasons[]? | select(startswith("Shape mode:"))] | join("; ")),
        selectionProfile: ([.reasons[]? | select(startswith("Selection profile:"))] | join("; "))
      }
    ],
    diagnostics,
    selectedReasons: .routes[0].reasons
  }'
}

if [[ -n "$SHAPE_DATA" ]]; then
  run_case "custom" "$SHAPE_INPUT_TYPE" "$SHAPE_DATA" "$ROUTE_TYPE" "$START_LAT" "$START_LNG" "$VARIANT_COUNT"
else
  [[ -f "$FIXTURE_PATH" ]] || fail "Missing smoke fixture: ${FIXTURE_PATH}"
  CASE_COUNT="$(jq '.cases // [.] | length' "$FIXTURE_PATH")"
  [[ "$CASE_COUNT" -gt 0 ]] || fail "Smoke fixture does not contain any cases: ${FIXTURE_PATH}"
  INDEX=0
  while [[ "$INDEX" -lt "$CASE_COUNT" ]]; do
    CASE_JSON="$(jq -c ".cases // [.] | .[$INDEX]" "$FIXTURE_PATH")"
    CASE_NAME="$(echo "$CASE_JSON" | jq -r '.name // "case"')"
    CASE_SHAPE_INPUT_TYPE="$(echo "$CASE_JSON" | jq -r '.shapeInputType // "draw"')"
    CASE_SHAPE_DATA="$(echo "$CASE_JSON" | jq -r '.shapeData')"
    CASE_ROUTE_TYPE="$(echo "$CASE_JSON" | jq -r '.routeType // "RIDE"')"
    CASE_START_LAT="$(echo "$CASE_JSON" | jq -r '.startPoint.lat // 48.121')"
    CASE_START_LNG="$(echo "$CASE_JSON" | jq -r '.startPoint.lng // -1.635')"
    CASE_VARIANT_COUNT="$(echo "$CASE_JSON" | jq -r '.variantCount // 3')"
    run_case "$CASE_NAME" "$CASE_SHAPE_INPUT_TYPE" "$CASE_SHAPE_DATA" "$CASE_ROUTE_TYPE" "$CASE_START_LAT" "$CASE_START_LNG" "$CASE_VARIANT_COUNT"
    INDEX=$((INDEX + 1))
  done
fi

log "Smoke check passed."

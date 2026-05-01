#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8080}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Ride}"
ROUTE_TYPE="${ROUTE_TYPE:-RIDE}"
VARIANT_COUNT="${VARIANT_COUNT:-3}"
START_LAT="${START_LAT:-45.18}"
START_LNG="${START_LNG:-5.72}"
SHAPE_DATA="${SHAPE_DATA:-[[45.180000,5.720000],[45.190000,5.745000],[45.205000,5.725000],[45.180000,5.720000]]}"

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
HEALTH_JSON="$(curl -sf "${BACKEND_URL}/api/health/details")" || fail "Backend is not reachable."
ROUTING_STATUS="$(echo "$HEALTH_JSON" | jq -r '.routing.status // "unknown"')"
ROUTING_REACHABLE="$(echo "$HEALTH_JSON" | jq -r '.routing.reachable // false')"
log "Routing status=${ROUTING_STATUS} reachable=${ROUTING_REACHABLE}"

REQUEST_BODY="$(jq -n \
  --arg shapeData "$SHAPE_DATA" \
  --arg routeType "$ROUTE_TYPE" \
  --argjson lat "$START_LAT" \
  --argjson lng "$START_LNG" \
  --argjson variantCount "$VARIANT_COUNT" \
  '{
    shapeInputType: "draw",
    shapeData: $shapeData,
    startPoint: { lat: $lat, lng: $lng },
    routeType: $routeType,
    variantCount: $variantCount
  }'
)"

GENERATE_ENDPOINT="${BACKEND_URL}/api/routes/generate/shape?activityType=${ACTIVITY_TYPE}"
log "Calling ${GENERATE_ENDPOINT}"
RESPONSE_JSON="$(curl -sf -X POST "$GENERATE_ENDPOINT" -H 'Content-Type: application/json' -H 'X-Request-Id: strava-art-smoke' -d "$REQUEST_BODY")" \
  || fail "Strava Art generation call failed."

ROUTES_COUNT="$(echo "$RESPONSE_JSON" | jq '.routes | length')"
if [[ "$ROUTES_COUNT" -lt 1 ]]; then
  echo "$RESPONSE_JSON" | jq '{routesCount: (.routes | length), diagnostics}'
  fail "No route returned for Strava Art smoke shape."
fi

ROUTE_ID="$(echo "$RESPONSE_JSON" | jq -r '.routes[0].routeId // empty')"
if [[ -z "$ROUTE_ID" ]]; then
  echo "$RESPONSE_JSON" | jq '{routesCount: (.routes | length), firstRoute: .routes[0]}'
  fail "First generated route has no routeId."
fi

ENCODED_ROUTE_ID="$(jq -nr --arg routeId "$ROUTE_ID" '$routeId | @uri')"
GPX_ENDPOINT="${BACKEND_URL}/api/routes/${ENCODED_ROUTE_ID}/gpx"
log "Exporting ${GPX_ENDPOINT}"
GPX_PAYLOAD="$(curl -sf "$GPX_ENDPOINT" -H 'Accept: application/gpx+xml')" || fail "GPX export failed for routeId=${ROUTE_ID}."
if [[ "$GPX_PAYLOAD" != *"<gpx"* || "$GPX_PAYLOAD" != *"<trkpt"* ]]; then
  fail "GPX export did not contain expected GPX track points."
fi

log "Smoke check passed."
echo "$RESPONSE_JSON" | jq '{
  routesCount: (.routes | length),
  selectedRouteId: .routes[0].routeId,
  selectedDistanceKm: .routes[0].distanceKm,
  selectedElevationGainM: .routes[0].elevationGainM,
  diagnostics
}'

#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8090}"
FRONT_URL="${FRONT_URL:-http://127.0.0.1:5173}"
ACTIVITY_ID="${ACTIVITY_ID:-1112134846}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Run}"
ROUTE_TYPE="${ROUTE_TYPE:-RUN}"
START_DIRECTION="${START_DIRECTION:-N}"
DISTANCE_TARGET_KM="${DISTANCE_TARGET_KM:-30}"
VARIANT_COUNT="${VARIANT_COUNT:-1}"
START_LAT="${START_LAT:-48.157563}"
START_LNG="${START_LNG:--1.587309}"
EXPECTED_CODES="${EXPECTED_CODES:-DIRECTION_RELAXED,DIRECTION_BEST_EFFORT,BACKTRACKING_RELAXED,ROUTE_TYPE_FALLBACK,START_POINT_SNAPPED,ENGINE_FALLBACK_LEGACY,SELECTION_RELAXED,EMERGENCY_FALLBACK}"

log() {
  printf '[route-fallback-check] %s\n' "$1"
}

fail() {
  printf '[route-fallback-check] ERROR: %s\n' "$1" >&2
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
log "Routing status: ${ROUTING_STATUS}"

if [[ -n "${ACTIVITY_ID}" ]]; then
  log "Trying to load start point from activity ${ACTIVITY_ID}"
  if ACTIVITY_JSON="$(curl -sf "${BACKEND_URL}/api/activities/${ACTIVITY_ID}" 2>/dev/null)"; then
    CANDIDATE_LAT="$(echo "$ACTIVITY_JSON" | jq -r '.startLatlng[0] // empty')"
    CANDIDATE_LNG="$(echo "$ACTIVITY_JSON" | jq -r '.startLatlng[1] // empty')"
    if [[ -n "$CANDIDATE_LAT" && -n "$CANDIDATE_LNG" ]]; then
      START_LAT="$CANDIDATE_LAT"
      START_LNG="$CANDIDATE_LNG"
      log "Using activity start point: ${START_LAT},${START_LNG}"
    else
      log "Activity did not expose startLatlng. Using configured coordinates ${START_LAT},${START_LNG}"
    fi
  else
    log "Activity endpoint unavailable for ${ACTIVITY_ID}. Using configured coordinates ${START_LAT},${START_LNG}"
  fi
fi

REQUEST_BODY="$(jq -n \
  --argjson lat "$START_LAT" \
  --argjson lng "$START_LNG" \
  --arg routeType "$ROUTE_TYPE" \
  --arg startDirection "$START_DIRECTION" \
  --argjson distanceTargetKm "$DISTANCE_TARGET_KM" \
  --argjson variantCount "$VARIANT_COUNT" \
  '{
    startPoint: { lat: $lat, lng: $lng },
    routeType: $routeType,
    startDirection: $startDirection,
    distanceTargetKm: $distanceTargetKm,
    variantCount: $variantCount
  }'
)"

TARGET_ENDPOINT="${BACKEND_URL}/api/routes/generate/target?activityType=${ACTIVITY_TYPE}"
log "Calling ${TARGET_ENDPOINT}"
RESPONSE_JSON="$(curl -sf -X POST "$TARGET_ENDPOINT" -H 'Content-Type: application/json' -d "$REQUEST_BODY")" || fail "Target generation call failed."

ROUTES_COUNT="$(echo "$RESPONSE_JSON" | jq '.routes | length')"
if [[ "$ROUTES_COUNT" -eq 0 ]]; then
  echo "$RESPONSE_JSON" | jq '{routesCount: (.routes | length), diagnostics}'
  fail "No routes returned. Try a different ACTIVITY_ID or start coordinates."
fi

MATCHING_CODES="$(echo "$RESPONSE_JSON" | jq -r --arg expected "$EXPECTED_CODES" '
  ($expected | split(",")) as $expectedCodes
  | [.diagnostics[]?.code | select(. as $c | $expectedCodes | index($c))]
  | unique
  | join(",")
')"

if [[ -z "$MATCHING_CODES" ]]; then
  echo "$RESPONSE_JSON" | jq '{routesCount: (.routes | length), diagnostics}'
  fail "No fallback diagnostic found in expected set: ${EXPECTED_CODES}"
fi

log "API check passed. Matching fallback diagnostics: ${MATCHING_CODES}"
echo "$RESPONSE_JSON" | jq '{routesCount: (.routes | length), diagnostics}'

cat <<UI_CHECK

UI manual check
1. Open ${FRONT_URL}/routes
2. Use mode Target and set the same start point (${START_LAT}, ${START_LNG})
3. Keep direction ${START_DIRECTION} and distance ${DISTANCE_TARGET_KM} km, then click Generate
4. Confirm at least one warning toast appears when a non-blocking fallback diagnostic is present
5. Confirm diagnostics are visible below the generator panel even when routes are returned

Expected result
- Route generation succeeds.
- A fallback diagnostic (for example DIRECTION_RELAXED or ENGINE_FALLBACK_LEGACY) is shown to the user.
UI_CHECK

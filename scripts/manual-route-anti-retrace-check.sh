#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8080}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Ride}"
VALIDATION_YEAR="${VALIDATION_YEAR:-1990}"
ROUTE_TYPE="${ROUTE_TYPE:-RIDE}"
VARIANT_COUNT="${VARIANT_COUNT:-6}"

# Dense urban reference case
DENSE_START_LAT="${DENSE_START_LAT:-48.1175}"
DENSE_START_LNG="${DENSE_START_LNG:--1.6780}"
DENSE_DISTANCE_KM="${DENSE_DISTANCE_KM:-40}"
DENSE_ELEVATION_M="${DENSE_ELEVATION_M:-800}"

# Rural/peri-rural reference case
RURAL_START_LAT="${RURAL_START_LAT:-48.157563}"
RURAL_START_LNG="${RURAL_START_LNG:--1.587309}"
RURAL_DISTANCE_KM="${RURAL_DISTANCE_KM:-30}"
RURAL_ELEVATION_M="${RURAL_ELEVATION_M:-350}"

log() {
  printf '[route-anti-retrace-check] %s\n' "$1"
}

fail() {
  printf '[route-anti-retrace-check] ERROR: %s\n' "$1" >&2
  exit 1
}

require_bin() {
  local bin="$1"
  command -v "$bin" >/dev/null 2>&1 || fail "Missing required binary: $bin"
}

require_bin curl
require_bin jq

check_backend_health() {
  log "Checking backend health at ${BACKEND_URL}/api/health/details"
  local health_json
  health_json="$(curl -sf "${BACKEND_URL}/api/health/details")" || fail "Backend is not reachable."
  local routing_status
  local routing_reachable
  routing_status="$(echo "$health_json" | jq -r '.routing.status // "unknown"')"
  routing_reachable="$(echo "$health_json" | jq -r '.routing.reachable // false')"
  log "Routing status=${routing_status} reachable=${routing_reachable}"
  if [[ "$routing_status" != "up" || "$routing_reachable" != "true" ]]; then
    fail "Routing backend is not ready (status=${routing_status}, reachable=${routing_reachable})."
  fi
}

extract_axis_reuse_values() {
  local reason="$1"
  local parsed
  parsed="$(echo "$reason" | sed -E 's/.*: ([0-9]+)x \(limit ([0-9]+)x\).*/\1 \2/')"
  if [[ ! "$parsed" =~ ^[0-9]+\ [0-9]+$ ]]; then
    fail "Unable to parse axis reuse reason: ${reason}"
  fi
  echo "$parsed"
}

extract_opposite_overlap_values() {
  local reason="$1"
  local parsed
  parsed="$(echo "$reason" | sed -E 's/.*: ([0-9]+)% \(limit ([0-9]+)%\).*/\1 \2/')"
  if [[ ! "$parsed" =~ ^[0-9]+\ [0-9]+$ ]]; then
    fail "Unable to parse opposite-overlap reason: ${reason}"
  fi
  echo "$parsed"
}

validate_case() {
  local case_name="$1"
  local start_lat="$2"
  local start_lng="$3"
  local distance_km="$4"
  local elevation_m="$5"

  local payload
  payload="$(jq -n \
    --argjson lat "$start_lat" \
    --argjson lng "$start_lng" \
    --arg routeType "$ROUTE_TYPE" \
    --argjson distance "$distance_km" \
    --argjson elevation "$elevation_m" \
    --argjson variantCount "$VARIANT_COUNT" \
    '{
      startPoint: { lat: $lat, lng: $lng },
      routeType: $routeType,
      distanceTargetKm: $distance,
      elevationTargetM: $elevation,
      variantCount: $variantCount
    }'
  )"

  local endpoint
  endpoint="${BACKEND_URL}/api/routes/generate/target?activityType=${ACTIVITY_TYPE}&year=${VALIDATION_YEAR}"
  local response_json
  response_json="$(curl -sf -X POST "$endpoint" -H 'Content-Type: application/json' -d "$payload")" \
    || fail "Target generation call failed for case ${case_name}."

  local routes_count
  routes_count="$(echo "$response_json" | jq '.routes | length')"
  if [[ "$routes_count" -lt 1 ]]; then
    echo "$response_json" | jq '{routesCount: (.routes | length), diagnostics}'
    fail "No route returned for ${case_name}."
  fi

  local max_observed_reuse=0
  local max_observed_opposite=0
  local route_index=0
  while [[ "$route_index" -lt "$routes_count" ]]; do
    local axis_reason
    axis_reason="$(echo "$response_json" | jq -r ".routes[$route_index].reasons[]? | select(startswith(\"Max axis reuse outside start zone:\"))" | head -n 1)"
    [[ -n "$axis_reason" ]] || fail "Missing outside-start axis reuse reason in case ${case_name} route #${route_index}."

    local overlap_reason
    overlap_reason="$(echo "$response_json" | jq -r ".routes[$route_index].reasons[]? | select(startswith(\"Opposite-axis overlap outside start zone:\"))" | head -n 1)"
    [[ -n "$overlap_reason" ]] || fail "Missing outside-start opposite-overlap reason in case ${case_name} route #${route_index}."

    read -r axis_value axis_limit <<<"$(extract_axis_reuse_values "$axis_reason")"
    read -r opposite_value opposite_limit <<<"$(extract_opposite_overlap_values "$overlap_reason")"

    if [[ "$axis_limit" -ne 1 ]]; then
      fail "Unexpected axis-reuse limit for ${case_name} route #${route_index}: ${axis_limit} (expected 1)."
    fi
    if [[ "$axis_value" -gt "$axis_limit" ]]; then
      fail "Axis reuse outside start zone exceeded limit for ${case_name} route #${route_index}: ${axis_value} > ${axis_limit}."
    fi

    if [[ "$opposite_limit" -ne 0 ]]; then
      fail "Unexpected opposite-overlap limit for ${case_name} route #${route_index}: ${opposite_limit}% (expected 0%)."
    fi
    if [[ "$opposite_value" -gt "$opposite_limit" ]]; then
      fail "Opposite-axis overlap outside start zone exceeded limit for ${case_name} route #${route_index}: ${opposite_value}% > ${opposite_limit}%."
    fi

    if [[ "$axis_value" -gt "$max_observed_reuse" ]]; then
      max_observed_reuse="$axis_value"
    fi
    if [[ "$opposite_value" -gt "$max_observed_opposite" ]]; then
      max_observed_opposite="$opposite_value"
    fi

    route_index=$((route_index + 1))
  done

  log "${case_name}: routes=${routes_count}, maxOutsideReuse=${max_observed_reuse}x, maxOppositeOutside=${max_observed_opposite}%"
  echo "$response_json" | jq \
    --arg case "$case_name" \
    --argjson routesCount "$routes_count" \
    --argjson maxReuse "$max_observed_reuse" \
    --argjson maxOpposite "$max_observed_opposite" \
    '{
      case: $case,
      routesCount: $routesCount,
      maxAxisReuseOutsideStartZone: $maxReuse,
      maxOppositeOverlapOutsideStartZonePercent: $maxOpposite,
      topRoute: {
        routeId: .routes[0].routeId,
        title: .routes[0].title,
        distanceKm: .routes[0].distanceKm,
        elevationGainM: .routes[0].elevationGainM
      }
    }'
}

check_backend_health

log "Running dense urban anti-retrace case"
validate_case "dense-urban" "$DENSE_START_LAT" "$DENSE_START_LNG" "$DENSE_DISTANCE_KM" "$DENSE_ELEVATION_M"

log "Running rural/peri-rural anti-retrace case"
validate_case "rural-peri" "$RURAL_START_LAT" "$RURAL_START_LNG" "$RURAL_DISTANCE_KM" "$RURAL_ELEVATION_M"

log "Anti-retrace validation passed for all routes in both scenarios."

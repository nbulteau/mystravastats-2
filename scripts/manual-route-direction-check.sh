#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8080}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Ride}"
VALIDATION_YEAR="${VALIDATION_YEAR:-1990}"
ROUTE_TYPE="${ROUTE_TYPE:-RIDE}"
VARIANT_COUNT="${VARIANT_COUNT:-6}"
MIN_ALIGNMENT_PERCENT="${MIN_ALIGNMENT_PERCENT:-85}"
MIN_EXPLICIT_DIRECTION_MATCHES="${MIN_EXPLICIT_DIRECTION_MATCHES:-1}"

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
  printf '[route-direction-check] %s\n' "$1"
}

fail() {
  printf '[route-direction-check] ERROR: %s\n' "$1" >&2
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

direction_label() {
  local direction="$1"
  case "$direction" in
    N) echo "North" ;;
    E) echo "East" ;;
    S) echo "South" ;;
    W) echo "West" ;;
    *)
      fail "Unsupported direction '${direction}', expected one of: N E S W."
      ;;
  esac
}

run_direction_case() {
  local case_name="$1"
  local start_lat="$2"
  local start_lng="$3"
  local distance_km="$4"
  local elevation_m="$5"
  local direction="$6"

  local payload
  if [[ -n "$direction" ]]; then
    payload="$(jq -n \
      --argjson lat "$start_lat" \
      --argjson lng "$start_lng" \
      --arg routeType "$ROUTE_TYPE" \
      --argjson distance "$distance_km" \
      --argjson elevation "$elevation_m" \
      --argjson variantCount "$VARIANT_COUNT" \
      --arg startDirection "$direction" \
      '{
        startPoint: { lat: $lat, lng: $lng },
        routeType: $routeType,
        distanceTargetKm: $distance,
        elevationTargetM: $elevation,
        variantCount: $variantCount,
        startDirection: $startDirection
      }'
    )"
  else
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
  fi

  local endpoint
  endpoint="${BACKEND_URL}/api/routes/generate/target?activityType=${ACTIVITY_TYPE}&year=${VALIDATION_YEAR}"
  local response_json
  response_json="$(curl -sf -X POST "$endpoint" -H 'Content-Type: application/json' -d "$payload")" \
    || fail "Target generation call failed for case ${case_name} direction ${direction:-NONE}."

  local routes_count
  routes_count="$(echo "$response_json" | jq '.routes | length')"
  if [[ "$routes_count" -lt 1 ]]; then
    echo "$response_json" | jq '{routesCount: (.routes | length), diagnostics}'
    fail "No route returned for ${case_name} direction ${direction:-NONE}."
  fi

  local top_alignment
  top_alignment="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Directional alignment: ")) | capture("(?<p>[0-9]+)").p' | head -n 1)"
  if [[ ! "$top_alignment" =~ ^[0-9]+$ ]]; then
    fail "Missing 'Directional alignment' reason in case ${case_name} direction ${direction:-NONE}."
  fi

  local top_direction_reason
  top_direction_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Direction: "))' | head -n 1)"

  local diagnostics_json
  diagnostics_json="$(echo "$response_json" | jq '[.diagnostics[]?.code]')"
  local has_relax_diag
  has_relax_diag="$(echo "$diagnostics_json" | jq -e 'index("DIRECTION_RELAXED") != null or index("DIRECTION_BEST_EFFORT") != null' >/dev/null && echo true || echo false)"

  local has_relax_reason
  has_relax_reason="$(echo "$response_json" | jq -e '.routes[0].reasons[]? | startswith("Direction relaxed:")' >/dev/null && echo true || echo false)"

  local direction_status="none-requested"
  if [[ -n "$direction" ]]; then
    local expected_label
    expected_label="$(direction_label "$direction")"
    if [[ -n "$top_direction_reason" ]]; then
      if [[ "$top_direction_reason" != "Direction: ${expected_label}" ]]; then
        fail "Top route direction mismatch for ${case_name} direction ${direction}: got '${top_direction_reason}', expected 'Direction: ${expected_label}'."
      fi
      if [[ "$top_alignment" -lt "$MIN_ALIGNMENT_PERCENT" ]]; then
        fail "Top route direction alignment too low for ${case_name} direction ${direction}: ${top_alignment}% < ${MIN_ALIGNMENT_PERCENT}%."
      fi
      direction_status="matched"
    else
      if [[ "$has_relax_diag" != "true" && "$has_relax_reason" != "true" ]]; then
        fail "Direction ${direction} has no explicit match and no relaxation diagnostic in ${case_name}."
      fi
      direction_status="relaxed"
    fi
  fi

  local top_route_id
  top_route_id="$(echo "$response_json" | jq -r '.routes[0].routeId // ""')"
  local alignment_values
  alignment_values="$(echo "$response_json" | jq '[.routes[].reasons[]? | select(startswith("Directional alignment: ")) | capture("(?<p>[0-9]+)").p | tonumber]')"

  jq -n \
    --arg case "$case_name" \
    --arg direction "${direction:-NONE}" \
    --arg directionStatus "$direction_status" \
    --arg topDirectionReason "${top_direction_reason:-}" \
    --arg topRouteId "$top_route_id" \
    --argjson routesCount "$routes_count" \
    --argjson topAlignment "$top_alignment" \
    --argjson diagnostics "$diagnostics_json" \
    --argjson alignmentValues "$alignment_values" \
    '{
      case: $case,
      direction: $direction,
      directionStatus: $directionStatus,
      routesCount: $routesCount,
      topRouteId: $topRouteId,
      topAlignment: $topAlignment,
      topDirectionReason: (if $topDirectionReason == "" then null else $topDirectionReason end),
      diagnostics: $diagnostics,
      alignmentValues: $alignmentValues
    }'
}

run_scenario() {
  local case_name="$1"
  local start_lat="$2"
  local start_lng="$3"
  local distance_km="$4"
  local elevation_m="$5"

  local explicit_matches=0
  local direction
  for direction in "" N E S W; do
    local summary
    summary="$(run_direction_case "$case_name" "$start_lat" "$start_lng" "$distance_km" "$elevation_m" "$direction")"
    echo "$summary"
    if [[ "$direction" != "" ]]; then
      local status
      status="$(echo "$summary" | jq -r '.directionStatus')"
      if [[ "$status" == "matched" ]]; then
        explicit_matches=$((explicit_matches + 1))
      fi
    fi
  done

  if [[ "$explicit_matches" -lt "$MIN_EXPLICIT_DIRECTION_MATCHES" ]]; then
    fail "${case_name}: only ${explicit_matches} explicit direction matches, expected at least ${MIN_EXPLICIT_DIRECTION_MATCHES}."
  fi

  log "${case_name}: explicitDirectionMatches=${explicit_matches}"
}

check_backend_health

log "Running dense urban direction matrix"
run_scenario "dense-urban" "$DENSE_START_LAT" "$DENSE_START_LNG" "$DENSE_DISTANCE_KM" "$DENSE_ELEVATION_M"

log "Running rural/peri-rural direction matrix"
run_scenario "rural-peri" "$RURAL_START_LAT" "$RURAL_START_LNG" "$RURAL_DISTANCE_KM" "$RURAL_ELEVATION_M"

log "Direction validation passed for all scenarios."

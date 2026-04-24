#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://127.0.0.1:8080}"
ACTIVITY_TYPE="${ACTIVITY_TYPE:-Ride}"
VALIDATION_YEAR="${VALIDATION_YEAR:-1990}"
VARIANT_COUNT="${VARIANT_COUNT:-6}"

# Scenario A: dense urban, essentially paved-only (fallback expected for GRAVEL/MTB).
DENSE_START_LAT="${DENSE_START_LAT:-48.1175}"
DENSE_START_LNG="${DENSE_START_LNG:--1.6780}"
DENSE_DISTANCE_KM="${DENSE_DISTANCE_KM:-40}"
DENSE_ELEVATION_M="${DENSE_ELEVATION_M:-800}"

# Scenario B: mixed urban/peri-urban with small offroad share.
MIXED_START_LAT="${MIXED_START_LAT:-48.1141}"
MIXED_START_LNG="${MIXED_START_LNG:--1.6144}"
MIXED_DISTANCE_KM="${MIXED_DISTANCE_KM:-25}"
MIXED_ELEVATION_M="${MIXED_ELEVATION_M:-250}"

log() {
  printf '[route-surface-check] %s\n' "$1"
}

fail() {
  printf '[route-surface-check] ERROR: %s\n' "$1" >&2
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

extract_percent_from_reason() {
  local reason="$1"
  local parsed
  parsed="$(echo "$reason" | sed -E 's/.*: ([0-9]+)%.*/\1/')"
  if [[ ! "$parsed" =~ ^[0-9]+$ ]]; then
    fail "Unable to parse percent value from reason: ${reason}"
  fi
  echo "$parsed"
}

call_route_type() {
  local case_name="$1"
  local start_lat="$2"
  local start_lng="$3"
  local distance_km="$4"
  local elevation_m="$5"
  local route_type="$6"

  local payload
  payload="$(jq -n \
    --argjson lat "$start_lat" \
    --argjson lng "$start_lng" \
    --arg routeType "$route_type" \
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
  curl -sf -X POST "$endpoint" -H 'Content-Type: application/json' -d "$payload" \
    || fail "Target generation failed for ${case_name}/${route_type}."
}

assert_surface_reasons_present() {
  local case_name="$1"
  local route_type="$2"
  local response_json="$3"

  local routes_count
  routes_count="$(echo "$response_json" | jq '.routes | length')"
  if [[ "$routes_count" -lt 1 ]]; then
    echo "$response_json" | jq '{routesCount: (.routes | length), diagnostics}'
    fail "No route returned for ${case_name}/${route_type}."
  fi

  local surface_mix_reason
  local path_ratio_reason
  local surface_fitness_reason
  local surface_source_reason
  surface_mix_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Surface mix: "))' | head -n 1)"
  path_ratio_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Path ratio: "))' | head -n 1)"
  surface_fitness_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Surface fitness: "))' | head -n 1)"
  surface_source_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Surface source: "))' | head -n 1)"

  [[ -n "$surface_mix_reason" ]] || fail "Missing 'Surface mix' reason for ${case_name}/${route_type}."
  [[ -n "$path_ratio_reason" ]] || fail "Missing 'Path ratio' reason for ${case_name}/${route_type}."
  [[ -n "$surface_fitness_reason" ]] || fail "Missing 'Surface fitness' reason for ${case_name}/${route_type}."
  [[ -n "$surface_source_reason" ]] || fail "Missing 'Surface source' reason for ${case_name}/${route_type}."
}

scenario_dense() {
  local route_type
  for route_type in RIDE GRAVEL MTB; do
    local response_json
    response_json="$(call_route_type "dense-urban" "$DENSE_START_LAT" "$DENSE_START_LNG" "$DENSE_DISTANCE_KM" "$DENSE_ELEVATION_M" "$route_type")"
    assert_surface_reasons_present "dense-urban" "$route_type" "$response_json"

    local diagnostics_json
    diagnostics_json="$(echo "$response_json" | jq '[.diagnostics[]?.code]')"
    local has_route_type_fallback
    has_route_type_fallback="$(echo "$diagnostics_json" | jq -e 'index("ROUTE_TYPE_FALLBACK") != null' >/dev/null && echo true || echo false)"

    local path_ratio_reason
    local surface_fitness_reason
    path_ratio_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Path ratio: "))' | head -n 1)"
    surface_fitness_reason="$(echo "$response_json" | jq -r '.routes[0].reasons[]? | select(startswith("Surface fitness: "))' | head -n 1)"
    local path_ratio
    local surface_fitness
    path_ratio="$(extract_percent_from_reason "$path_ratio_reason")"
    surface_fitness="$(extract_percent_from_reason "$surface_fitness_reason")"

    case "$route_type" in
      RIDE)
        if [[ "$has_route_type_fallback" != "false" ]]; then
          fail "Dense scenario: RIDE should not trigger ROUTE_TYPE_FALLBACK."
        fi
        ;;
      GRAVEL|MTB)
        if [[ "$has_route_type_fallback" != "true" ]]; then
          fail "Dense scenario: ${route_type} should trigger ROUTE_TYPE_FALLBACK on paved-only profile."
        fi
        ;;
    esac

    if [[ "$path_ratio" -ne 0 ]]; then
      fail "Dense scenario: expected path ratio 0% for ${route_type}, got ${path_ratio}%."
    fi

    jq -n \
      --arg case "dense-urban" \
      --arg routeType "$route_type" \
      --argjson diagnostics "$diagnostics_json" \
      --argjson pathRatio "$path_ratio" \
      --argjson surfaceFitness "$surface_fitness" \
      --argjson routeTypeFallback "$has_route_type_fallback" \
      '{
        case: $case,
        routeType: $routeType,
        routeTypeFallback: $routeTypeFallback,
        pathRatio: $pathRatio,
        surfaceFitness: $surfaceFitness,
        diagnostics: $diagnostics
      }'
  done
}

scenario_mixed() {
  local ride_response
  local gravel_response
  local mtb_response
  ride_response="$(call_route_type "mixed-urban-paths" "$MIXED_START_LAT" "$MIXED_START_LNG" "$MIXED_DISTANCE_KM" "$MIXED_ELEVATION_M" "RIDE")"
  gravel_response="$(call_route_type "mixed-urban-paths" "$MIXED_START_LAT" "$MIXED_START_LNG" "$MIXED_DISTANCE_KM" "$MIXED_ELEVATION_M" "GRAVEL")"
  mtb_response="$(call_route_type "mixed-urban-paths" "$MIXED_START_LAT" "$MIXED_START_LNG" "$MIXED_DISTANCE_KM" "$MIXED_ELEVATION_M" "MTB")"

  assert_surface_reasons_present "mixed-urban-paths" "RIDE" "$ride_response"
  assert_surface_reasons_present "mixed-urban-paths" "GRAVEL" "$gravel_response"
  assert_surface_reasons_present "mixed-urban-paths" "MTB" "$mtb_response"

  local ride_path gravel_path mtb_path
  local ride_fitness gravel_fitness mtb_fitness
  ride_path="$(extract_percent_from_reason "$(echo "$ride_response" | jq -r '.routes[0].reasons[]? | select(startswith("Path ratio: "))' | head -n 1)")"
  gravel_path="$(extract_percent_from_reason "$(echo "$gravel_response" | jq -r '.routes[0].reasons[]? | select(startswith("Path ratio: "))' | head -n 1)")"
  mtb_path="$(extract_percent_from_reason "$(echo "$mtb_response" | jq -r '.routes[0].reasons[]? | select(startswith("Path ratio: "))' | head -n 1)")"
  ride_fitness="$(extract_percent_from_reason "$(echo "$ride_response" | jq -r '.routes[0].reasons[]? | select(startswith("Surface fitness: "))' | head -n 1)")"
  gravel_fitness="$(extract_percent_from_reason "$(echo "$gravel_response" | jq -r '.routes[0].reasons[]? | select(startswith("Surface fitness: "))' | head -n 1)")"
  mtb_fitness="$(extract_percent_from_reason "$(echo "$mtb_response" | jq -r '.routes[0].reasons[]? | select(startswith("Surface fitness: "))' | head -n 1)")"

  local ride_diags gravel_diags mtb_diags
  ride_diags="$(echo "$ride_response" | jq '[.diagnostics[]?.code]')"
  gravel_diags="$(echo "$gravel_response" | jq '[.diagnostics[]?.code]')"
  mtb_diags="$(echo "$mtb_response" | jq '[.diagnostics[]?.code]')"

  local ride_fallback gravel_fallback mtb_fallback
  ride_fallback="$(echo "$ride_diags" | jq -e 'index("ROUTE_TYPE_FALLBACK") != null' >/dev/null && echo true || echo false)"
  gravel_fallback="$(echo "$gravel_diags" | jq -e 'index("ROUTE_TYPE_FALLBACK") != null' >/dev/null && echo true || echo false)"
  mtb_fallback="$(echo "$mtb_diags" | jq -e 'index("ROUTE_TYPE_FALLBACK") != null' >/dev/null && echo true || echo false)"

  if [[ "$ride_fallback" != "false" ]]; then
    fail "Mixed scenario: RIDE should not trigger ROUTE_TYPE_FALLBACK."
  fi
  if [[ "$gravel_fallback" != "true" ]]; then
    fail "Mixed scenario: GRAVEL should trigger ROUTE_TYPE_FALLBACK when path ratio stays below 25%."
  fi
  if [[ "$mtb_fallback" != "false" ]]; then
    fail "Mixed scenario: MTB should stay in MTB mode (no ROUTE_TYPE_FALLBACK) for this calibration case."
  fi

  if [[ "$ride_fitness" -lt 80 ]]; then
    fail "Mixed scenario: RIDE surface fitness unexpectedly low (${ride_fitness}%)."
  fi
  if [[ "$gravel_fitness" -lt 80 ]]; then
    fail "Mixed scenario: fallback GRAVEL surface fitness unexpectedly low (${gravel_fitness}%)."
  fi
  if [[ "$mtb_fitness" -gt 10 ]]; then
    fail "Mixed scenario: MTB fitness should stay very low on mostly paved profile (got ${mtb_fitness}%)."
  fi
  if [[ "$mtb_path" -lt "$ride_path" ]]; then
    fail "Mixed scenario: MTB path ratio should be >= RIDE path ratio (ride=${ride_path}% mtb=${mtb_path}%)."
  fi

  jq -n \
    --arg case "mixed-urban-paths" \
    --argjson rideDiagnostics "$ride_diags" \
    --argjson gravelDiagnostics "$gravel_diags" \
    --argjson mtbDiagnostics "$mtb_diags" \
    --argjson ridePath "$ride_path" \
    --argjson gravelPath "$gravel_path" \
    --argjson mtbPath "$mtb_path" \
    --argjson rideFitness "$ride_fitness" \
    --argjson gravelFitness "$gravel_fitness" \
    --argjson mtbFitness "$mtb_fitness" \
    '{
      case: $case,
      ride: {
        pathRatio: $ridePath,
        surfaceFitness: $rideFitness,
        diagnostics: $rideDiagnostics
      },
      gravel: {
        pathRatio: $gravelPath,
        surfaceFitness: $gravelFitness,
        diagnostics: $gravelDiagnostics
      },
      mtb: {
        pathRatio: $mtbPath,
        surfaceFitness: $mtbFitness,
        diagnostics: $mtbDiagnostics
      }
    }'
}

check_backend_health

log "Running dense paved scenario"
scenario_dense

log "Running mixed urban/paths scenario"
scenario_mixed

log "Surface scoring validation passed for ROUTE-P1-01."

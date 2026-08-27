import type { GeneratedRoute, RouteGenerationDiagnostic } from "@/models/route-recommendation.model";
import { formatTime } from "@/utils/formatters";

export const nonBlockingGenerationDiagnosticCodes = new Set([
  "DIRECTION_RELAXED",
  "DIRECTION_BEST_EFFORT",
  "BACKTRACKING_RELAXED",
  "ROUTE_TYPE_FALLBACK",
  "START_POINT_SNAPPED",
  "ENGINE_FALLBACK_LEGACY",
  "SELECTION_RELAXED",
  "EMERGENCY_FALLBACK",
]);

export interface RouteBadge {
  id: string;
  label: string;
  tone: "strong" | "info" | "warn";
  icon: string;
}

export interface PresentedDiagnostic {
  code: string;
  title: string;
  message: string;
  tone: "info" | "warn" | "error";
  icon: string;
}

export function formatDistance(value: number): string {
  return `${value.toFixed(1)} km`;
}

export function formatElevation(value: number): string {
  return `${Math.round(value)} m`;
}

export function formatSignedDistanceDelta(value: number): string {
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(1)} km`;
}

export function formatSignedPercent(value: number): string {
  const rounded = Math.round(value);
  const sign = rounded > 0 ? "+" : "";
  return `${sign}${rounded}%`;
}

export function clampScore(value: number | undefined): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.min(100, value));
}

export function scoreMeterStyle(value: number | undefined) {
  return { width: `${Math.round(clampScore(value))}%` };
}

export function distanceDeltaClass(deltaRatio: number): string {
  const absoluteDelta = Math.abs(deltaRatio);
  if (absoluteDelta <= 12) {
    return "routes-comparison-value routes-comparison-value--strong";
  }
  if (absoluteDelta <= 35) {
    return "routes-comparison-value routes-comparison-value--mixed";
  }
  return "routes-comparison-value routes-comparison-value--warn";
}

export function coordinateDistanceKm(from: number[], to: number[]): number {
  if (from.length < 2 || to.length < 2) {
    return 0;
  }
  const [fromLat, fromLng] = from;
  const [toLat, toLng] = to;
  if (
    !Number.isFinite(fromLat)
    || !Number.isFinite(fromLng)
    || !Number.isFinite(toLat)
    || !Number.isFinite(toLng)
  ) {
    return 0;
  }
  const toRadians = (value: number) => (value * Math.PI) / 180;
  const earthRadiusKm = 6371;
  const deltaLat = toRadians(toLat - fromLat);
  const deltaLng = toRadians(toLng - fromLng);
  const startLat = toRadians(fromLat);
  const endLat = toRadians(toLat);
  const haversine = Math.sin(deltaLat / 2) ** 2
    + Math.cos(startLat) * Math.cos(endLat) * Math.sin(deltaLng / 2) ** 2;
  return 2 * earthRadiusKm * Math.atan2(Math.sqrt(haversine), Math.sqrt(1 - haversine));
}

export function polylineDistanceKm(points: number[][]): number {
  if (points.length < 2) {
    return 0;
  }
  let distance = 0;
  for (let index = 1; index < points.length; index += 1) {
    distance += coordinateDistanceKm(points[index - 1], points[index]);
  }
  return distance;
}

export function formatVariantType(value: string): string {
  return value
    .replaceAll("_", " ")
    .toLowerCase()
    .replace(/\b\w/g, (match) => match.toUpperCase());
}

export function artFitScore(route: GeneratedRoute): number {
  const global = clampScore(route.score.global);
  const shape = clampScore(route.score.shape);
  return Math.round((shape * 0.90) + (global * 0.10));
}

export function artFitLabel(route: GeneratedRoute): string {
  const score = artFitScore(route);
  if (score >= 90) {
    return "Crisp art";
  }
  if (score >= 82) {
    return "Readable art";
  }
  if (score >= 68) {
    return "Loose match";
  }
  return "Review shape";
}

export function visualMatchSummary(score: number): string {
  if (score >= 82) {
    return "Good match";
  }
  if (score >= 68) {
    return "Medium match";
  }
  return "Weak match";
}

export function visualMatchMessage(score: number): string {
  if (score >= 82) {
    return "The generated route keeps the sketch readable.";
  }
  if (score >= 68) {
    return "The route follows the idea, but some parts drift from the sketch.";
  }
  return "The route is usable as a fallback, but the drawing is hard to read.";
}

export function visualMatchClass(score: number): string {
  if (score >= 82) {
    return "routes-visual-match routes-visual-match--strong";
  }
  if (score >= 68) {
    return "routes-visual-match routes-visual-match--mixed";
  }
  return "routes-visual-match routes-visual-match--weak";
}

export function artFitClass(route: GeneratedRoute): string {
  const score = artFitScore(route);
  if (score >= 90) {
    return "route-quality-chip route-quality-chip--strong";
  }
  if (score >= 82) {
    return "route-quality-chip route-quality-chip--ok";
  }
  return "route-quality-chip route-quality-chip--warn";
}

export function routeQualityScore(route: GeneratedRoute): number {
  const global = clampScore(route.score.global);
  const roadFitness = clampScore(route.score.roadFitness);
  return Math.round((roadFitness * 0.60) + (global * 0.40));
}

export function routeQualityLabel(route: GeneratedRoute): string {
  const score = routeQualityScore(route);
  if (score >= 85) {
    return "Easy to ride";
  }
  if (score >= 70) {
    return "Usable ride";
  }
  if (score >= 55) {
    return "Check before riding";
  }
  return "Low confidence";
}

export function scoreBandClass(value: number | undefined): string {
  const score = clampScore(value);
  if (score >= 85) {
    return "route-score-row route-score-row--strong";
  }
  if (score >= 70) {
    return "route-score-row route-score-row--ok";
  }
  if (score >= 55) {
    return "route-score-row route-score-row--mixed";
  }
  return "route-score-row route-score-row--warn";
}

export function routeSourceLabel(route: GeneratedRoute): string {
  const shapeMode = routeShapeMode(route);
  if (shapeMode === "edited osrm control route") {
    return "Edited OSRM route";
  }
  if (shapeMode === "nearest-road trace") {
    return "Drawing-first road snap";
  }
  if (shapeMode === "segment stitched alternatives") {
    return "Segment road snap";
  }
  if (shapeMode.includes("fallback")) {
    return "Best-effort OSRM snap";
  }
  if (shapeMode.length > 0) {
    return "OSRM sketch anchors";
  }
  if (route.isRoadGraphGenerated) {
    return "OSRM road snap";
  }
  return formatVariantType(route.variantType);
}

export function routeReasons(route: GeneratedRoute): string[] {
  return route.reasons
    .map((reason) => reason.trim())
    .filter((reason) => reason.length > 0);
}

export function routeReasonPayload(route: GeneratedRoute, prefix: string): string {
  const normalizedPrefix = prefix.toLowerCase();
  const reason = routeReasons(route).find((candidate) =>
    candidate.toLowerCase().startsWith(normalizedPrefix)
  );
  if (!reason) {
    return "";
  }
  return reason.slice(prefix.length).trim();
}

export function hasRouteReason(route: GeneratedRoute, prefix: string): boolean {
  return routeReasonPayload(route, prefix).length > 0;
}

export function routeShapeMode(route: GeneratedRoute): string {
  return routeReasonPayload(route, "Shape mode:").toLowerCase();
}

export function routeSelectionProfile(route: GeneratedRoute): string {
  return routeReasonPayload(route, "Selection profile:").toLowerCase();
}

export function routeShapeSimilarity(route: GeneratedRoute): number | null {
  const payload = routeReasonPayload(route, "Shape similarity:");
  const match = payload.match(/^(\d+(?:\.\d+)?)%/);
  if (!match) {
    return null;
  }
  const value = Number.parseFloat(match[1]);
  return Number.isFinite(value) ? Math.round(value) : null;
}

export function routeProductBadges(route: GeneratedRoute): RouteBadge[] {
  const badges: RouteBadge[] = [];
  const shapeMode = routeShapeMode(route);
  const profile = routeSelectionProfile(route);

  if (shapeMode === "nearest-road trace") {
    badges.push({
      id: "mode-nearest",
      label: "Drawing-first snap",
      tone: "strong",
      icon: "fa-solid fa-magnet",
    });
  } else if (shapeMode === "edited osrm control route") {
    badges.push({
      id: "mode-edited",
      label: "Edited on roads",
      tone: "strong",
      icon: "fa-solid fa-magnet",
    });
  } else if (shapeMode === "segment stitched alternatives") {
    badges.push({
      id: "mode-segment",
      label: "Segment stitching",
      tone: "info",
      icon: "fa-solid fa-route",
    });
  } else if (shapeMode.includes("fallback")) {
    badges.push({
      id: "mode-fallback",
      label: "Fallback shape",
      tone: "warn",
      icon: "fa-solid fa-triangle-exclamation",
    });
  } else if (shapeMode.length > 0) {
    badges.push({
      id: "mode-osrm",
      label: "OSRM anchors",
      tone: "info",
      icon: "fa-solid fa-map-location-dot",
    });
  }

  if (profile.startsWith("strict")) {
    badges.push({
      id: "profile-strict",
      label: "Strict fit",
      tone: "strong",
      icon: "fa-solid fa-circle-check",
    });
  } else if (profile.startsWith("art-fit-diagnostic")) {
    badges.push({
      id: "profile-art-diagnostic",
      label: "Drawing wins",
      tone: "strong",
      icon: "fa-solid fa-pen-nib",
    });
  } else if (profile.startsWith("best-effort-soft")) {
    badges.push({
      id: "profile-soft",
      label: "Best effort",
      tone: "warn",
      icon: "fa-solid fa-life-ring",
    });
  } else if (profile.includes("emergency-fallback")) {
    badges.push({
      id: "profile-emergency",
      label: "Fully relaxed",
      tone: "warn",
      icon: "fa-solid fa-life-ring",
    });
  }

  if (hasRouteReason(route, "Selection priority: art-fit first")) {
    badges.push({
      id: "priority-art-fit",
      label: "Art fit first",
      tone: "strong",
      icon: "fa-solid fa-pen-nib",
    });
  }
  if (hasRouteReason(route, "Retrace policy:")) {
    badges.push({
      id: "retrace-art",
      label: "Overlap allowed",
      tone: "info",
      icon: "fa-solid fa-repeat",
    });
  }

  return badges.slice(0, 3);
}

export function routeProductSummary(route: GeneratedRoute): string {
  const shapeMode = routeShapeMode(route);
  const profile = routeSelectionProfile(route);
  if (shapeMode === "edited osrm control route") {
    return "Manual correction snapped to OSRM roads.";
  }
  if (shapeMode === "nearest-road trace") {
    return "Sketch order preserved on nearby routable roads.";
  }
  if (profile.includes("emergency-fallback")) {
    return "Exportable fallback; inspect the drawing before riding.";
  }
  if (profile.startsWith("art-fit-diagnostic")) {
    return "Drawing match selected; overlap is rideability context.";
  }
  if (profile.startsWith("best-effort-soft")) {
    return "Best-effort route kept available for export.";
  }
  if (shapeMode === "segment stitched alternatives") {
    return "OSRM alternatives stitched segment by segment.";
  }
  return routeQualityLabel(route);
}

export function highlightedRouteReasons(route: GeneratedRoute): string[] {
  const highlights: string[] = [];
  const shapeSimilarity = routeShapeSimilarity(route);
  const shapeMode = routeShapeMode(route);
  const profile = routeSelectionProfile(route);

  if (shapeSimilarity !== null) {
    highlights.push(`Visual match: ${shapeSimilarity}% shape similarity.`);
  }

  if (hasRouteReason(route, "Shape trace snap:")) {
    highlights.push("Road snap: nearest anchors, routed by OSRM.");
  } else if (shapeMode === "edited osrm control route") {
    highlights.push("Edit: control points snapped and rerouted by OSRM.");
  } else if (shapeMode === "segment stitched alternatives") {
    highlights.push("Routing: alternatives chosen per sketch segment.");
  } else if (shapeMode.includes("fallback")) {
    highlights.push("Routing: fallback kept an exportable route.");
  }

  const shapeTransform = routeReasonPayload(route, "Shape transform:");
  if (shapeTransform) {
    highlights.push(`Transform: ${shapeTransform}.`);
  }

  if (profile.startsWith("strict")) {
    highlights.push("Confidence: strict candidate selected.");
  } else if (profile.startsWith("art-fit-diagnostic")) {
    highlights.push("Priority: drawing resemblance selected first.");
  } else if (profile.startsWith("best-effort-soft")) {
    highlights.push("Confidence: relaxed to preserve the artwork.");
  } else if (profile.includes("emergency-fallback")) {
    highlights.push("Confidence: fully relaxed fallback.");
  }

  if (hasRouteReason(route, "Retrace policy:")) {
    highlights.push("Overlap: allowed when it keeps the drawing recognizable.");
  }

  if (hasRouteReason(route, "Shape similarity below ideal:")) {
    highlights.push("Review: visual match is below the ideal target.");
  }

  return [...new Set(highlights)].slice(0, 3);
}

export function routeTitle(route: GeneratedRoute, index: number): string {
  const title = route.title.trim();
  if (title.length > 0 && title !== route.routeId) {
    return title;
  }
  return `Proposal ${index + 1}`;
}

export function diagnosticTitle(code: string): string {
  switch (code) {
    case "OSRM_COVERAGE_MISMATCH":
      return "OSRM coverage mismatch";
    case "OSRM_COVERAGE_UNAVAILABLE":
      return "OSRM coverage unavailable";
    case "NO_CANDIDATE":
      return "No road match";
    case "FAILURE_SUMMARY":
      return "Generation blocked";
    case "ROUTE_TYPE_FALLBACK":
      return "Activity style adjusted";
    case "START_POINT_SNAPPED":
      return "Start point moved";
    case "NON_SHAPE_CANDIDATES_IGNORED":
      return "Older routes ignored";
    case "ENGINE_CACHE_FALLBACK":
      return "Historical route used";
    case "ENGINE_FALLBACK_LEGACY":
      return "Backup routing used";
    case "BACKTRACKING_RELAXED":
      return "Overlap rule softened";
    case "ART_FIT_RETRACE_ALLOWED":
      return "Drawing kept first";
    case "DIRECTION_RELAXED":
    case "DIRECTION_BEST_EFFORT":
      return "Heading softened";
    case "SELECTION_RELAXED":
      return "Selection softened";
    case "EMERGENCY_FALLBACK":
      return "Best available route";
    case "EDIT_ROUTE_UPDATED":
      return "Route edited";
    case "EDIT_POINT_NOT_ROUTABLE":
    case "EDIT_POINT_OUT_OF_COVERAGE":
      return "Control point blocked";
    case "EDIT_SEGMENT_NO_ROUTE":
      return "Segment blocked";
    case "EDIT_CONTROL_POINTS_TOO_FEW":
      return "More controls needed";
    default:
      return code.replaceAll("_", " ").toLowerCase().replace(/\b\w/g, (match) => match.toUpperCase());
  }
}

export function diagnosticMessage(diagnostic: RouteGenerationDiagnostic): string {
  switch (diagnostic.code) {
    case "OSRM_COVERAGE_MISMATCH":
      return diagnostic.message;
    case "OSRM_COVERAGE_UNAVAILABLE":
      return diagnostic.message;
    case "NO_CANDIDATE":
      return "The sketch could not be matched to routable roads.";
    case "FAILURE_SUMMARY":
      return diagnostic.message.replace("Try simplifying the shape or moving the start point.", "Simplify the sketch, move the start point, or try fewer tight turns.");
    case "ROUTE_TYPE_FALLBACK":
      return "The requested activity style was changed to keep the route practicable.";
    case "START_POINT_SNAPPED":
      return "The start was moved to the closest routable point.";
    case "NON_SHAPE_CANDIDATES_IGNORED":
      return "Existing activities were available, but GPS Art only returns OSRM routes generated from the sketch.";
    case "ENGINE_CACHE_FALLBACK":
      return "OSRM did not produce a better candidate, so a known historical route was returned.";
    case "ENGINE_FALLBACK_LEGACY":
      return "A backup routing strategy was used to keep a proposal available.";
    case "BACKTRACKING_RELAXED":
      return "Some overlap was allowed to preserve the artwork.";
    case "ART_FIT_RETRACE_ALLOWED":
      return "The drawing match won; overlap is shown as rideability context instead of blocking the route.";
    case "DIRECTION_RELAXED":
    case "DIRECTION_BEST_EFFORT":
      return "The internal heading preference was softened to keep the route available.";
    case "SELECTION_RELAXED":
      return "Selection rules were softened to return a usable proposal.";
    case "EMERGENCY_FALLBACK":
      return "The best available generated route was selected despite weak matching.";
    case "EDIT_ROUTE_UPDATED":
      return "The edited route stays snapped to OSRM roads.";
    case "EDIT_POINT_NOT_ROUTABLE":
    case "EDIT_POINT_OUT_OF_COVERAGE":
    case "EDIT_SEGMENT_NO_ROUTE":
    case "EDIT_CONTROL_POINTS_TOO_FEW":
      return diagnostic.message;
    default:
      return diagnostic.message;
  }
}

export function diagnosticTone(code: string): PresentedDiagnostic["tone"] {
  if (
    code === "NO_CANDIDATE"
    || code === "FAILURE_SUMMARY"
    || code.startsWith("OSRM_COVERAGE_")
    || code === "EDIT_POINT_NOT_ROUTABLE"
    || code === "EDIT_POINT_OUT_OF_COVERAGE"
    || code === "EDIT_SEGMENT_NO_ROUTE"
    || code === "EDIT_CONTROL_POINTS_TOO_FEW"
  ) {
    return "error";
  }
  if (nonBlockingGenerationDiagnosticCodes.has(code)) {
    return "warn";
  }
  return "info";
}

export function diagnosticIcon(code: string): string {
  if (
    code === "NO_CANDIDATE"
    || code === "FAILURE_SUMMARY"
    || code.startsWith("OSRM_COVERAGE_")
    || code === "EDIT_POINT_NOT_ROUTABLE"
    || code === "EDIT_POINT_OUT_OF_COVERAGE"
    || code === "EDIT_SEGMENT_NO_ROUTE"
    || code === "EDIT_CONTROL_POINTS_TOO_FEW"
  ) {
    return "fa-solid fa-triangle-exclamation";
  }
  if (code === "EDIT_ROUTE_UPDATED") {
    return "fa-solid fa-magnet";
  }
  if (code === "START_POINT_SNAPPED") {
    return "fa-solid fa-location-dot";
  }
  if (code === "ROUTE_TYPE_FALLBACK") {
    return "fa-solid fa-route";
  }
  if (code === "ART_FIT_RETRACE_ALLOWED") {
    return "fa-solid fa-pen-nib";
  }
  if (code.includes("FALLBACK")) {
    return "fa-solid fa-life-ring";
  }
  return "fa-solid fa-circle-info";
}

export function presentDiagnostic(diagnostic: RouteGenerationDiagnostic): PresentedDiagnostic {
  return {
    code: diagnostic.code,
    title: diagnosticTitle(diagnostic.code),
    message: diagnosticMessage(diagnostic),
    tone: diagnosticTone(diagnostic.code),
    icon: diagnosticIcon(diagnostic.code),
  };
}

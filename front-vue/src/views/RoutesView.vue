<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { useContextStore } from "@/stores/context";
import { useRoutesStore } from "@/stores/routes";
import type { BuiltInShapeTemplateKey } from "@/stores/routes";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import type { GeneratedRoute, RouteGenerationDiagnostic, RouteType } from "@/models/route-recommendation.model";
import { formatTime } from "@/utils/formatters";

const contextStore = useContextStore();
const routesStore = useRoutesStore();
const uiStore = useUiStore();
onMounted(() => contextStore.updateCurrentView("routes"));

const mapContainer = ref<HTMLDivElement | null>(null);
const map = ref<L.Map>();
const startMarker = ref<L.CircleMarker>();
const shapePolylineLayer = ref<L.Polyline>();
const selectedRouteOutlineLayer = ref<L.Polyline>();
const selectedRouteLayer = ref<L.Polyline>();
const traceImageLayer = ref<L.ImageOverlay>();
const gpxFileInput = ref<HTMLInputElement | null>(null);
const traceImageFileInput = ref<HTMLInputElement | null>(null);
const gpxImportMode = ref<"replace" | "append">("replace");
const selectedShapeTemplate = ref<BuiltInShapeTemplateKey>("heart");
const saveShapeName = ref("");
const traceImageName = ref("");
const traceImageUrl = ref("");
const traceImageBounds = ref<L.LatLngBoundsExpression | null>(null);
const isExporting = ref(false);
const isLocating = ref(false);

const selectedRoute = computed(() => routesStore.selectedRoute);
const generationDiagnostics = computed(() => routesStore.generationDiagnostics);
const failureSummaryDiagnostic = computed(() =>
  generationDiagnostics.value.find((diagnostic) => diagnostic.code === "FAILURE_SUMMARY") ?? null,
);
const detailedGenerationDiagnostics = computed(() =>
  generationDiagnostics.value.filter((diagnostic) => diagnostic.code !== "FAILURE_SUMMARY"),
);
const productFailureSummary = computed(() =>
  failureSummaryDiagnostic.value ? presentDiagnostic(failureSummaryDiagnostic.value) : null,
);
const productGenerationDiagnostics = computed(() =>
  detailedGenerationDiagnostics.value.map((diagnostic) => presentDiagnostic(diagnostic)),
);
const canTransformShape = computed(() => routesStore.canTransformShape);
const builtInShapeTemplates: Array<{ key: BuiltInShapeTemplateKey; label: string }> = [
  { key: "heart", label: "Heart" },
  { key: "star", label: "Star" },
  { key: "circle", label: "Circle" },
  { key: "square", label: "Square" },
  { key: "triangle", label: "Triangle" },
  { key: "diamond", label: "Diamond" },
  { key: "rectangle", label: "Rectangle" },
  { key: "hexagon", label: "Hexagon" },
];
interface CorrectionSuggestion {
  id: string;
  title: string;
  message: string;
  icon: string;
  action?: "simplify" | "smooth" | "center" | "scaleDown" | "scaleUp" | "useLocation" | "generate" | "heart" | "circle";
  disabled?: boolean;
}

const correctionSuggestions = computed<CorrectionSuggestion[]>(() => {
  const suggestions: CorrectionSuggestion[] = [];
  const route = selectedRoute.value;
  const pointCount = routesStore.shapePoints.length;
  if (pointCount < 2) {
    suggestions.push({
      id: "start-template",
      title: "Start from a simple shape",
      message: "Use a template or import an image before routing.",
      icon: "fa-solid fa-shapes",
      action: "heart",
    });
    return suggestions;
  }
  if (!routesStore.startPoint) {
    suggestions.push({
      id: "start-point",
      title: "Anchor the sketch",
      message: "Set a start point before snapping to roads.",
      icon: "fa-solid fa-location-crosshairs",
      action: "useLocation",
    });
  }
  if (pointCount > 120) {
    suggestions.push({
      id: "too-many-points",
      title: "Simplify the trace",
      message: "Reduce point count before asking OSRM to snap it.",
      icon: "fa-solid fa-compress",
      action: "simplify",
    });
  }
  if (route && artFitScore(route) < 82) {
    suggestions.push({
      id: "low-art-fit",
      title: "Improve visual match",
      message: "Smooth the sketch or move it around the start point.",
      icon: "fa-solid fa-wand-magic-sparkles",
      action: routesStore.shapePoints.length >= 4 ? "smooth" : "scaleDown",
    });
  }
  if (route && routeQualityScore(route) < 70) {
    suggestions.push({
      id: "route-quality",
      title: "Make it easier to route",
      message: "Try a smaller sketch or center it closer to the start point.",
      icon: "fa-solid fa-route",
      action: routesStore.startPoint ? "center" : "scaleDown",
      disabled: !routesStore.startPoint,
    });
  }
  if (generationDiagnostics.value.some((diagnostic) => diagnostic.code === "NO_CANDIDATE" || diagnostic.code === "FAILURE_SUMMARY")) {
    suggestions.push({
      id: "no-candidate",
      title: "Recover generation",
      message: "Simplify the shape, then generate again.",
      icon: "fa-solid fa-triangle-exclamation",
      action: "simplify",
    });
  }
  if (suggestions.length === 0) {
    suggestions.push({
      id: "ready",
      title: "Ready to export",
      message: "The selected proposal looks usable.",
      icon: "fa-solid fa-circle-check",
      action: "generate",
      disabled: !canGenerate.value,
    });
  }
  return suggestions.slice(0, 3);
});
const routeComparisonSummary = computed(() => {
  const route = selectedRoute.value;
  const sketchDistanceKm = polylineDistanceKm(routesStore.shapePoints);
  if (!route || routesStore.shapePoints.length < 2 || sketchDistanceKm <= 0) {
    return null;
  }
  const routeDistanceKm = Math.max(0, route.distanceKm);
  const deltaKm = routeDistanceKm - sketchDistanceKm;
  const deltaRatio = (deltaKm / sketchDistanceKm) * 100;
  const fitScore = artFitScore(route);
  return {
    sketchDistance: formatDistance(sketchDistanceKm),
    routeDistance: formatDistance(routeDistanceKm),
    deltaLabel: formatSignedDistanceDelta(deltaKm),
    deltaRatioLabel: formatSignedPercent(deltaRatio),
    deltaClass: distanceDeltaClass(deltaRatio),
    fitClass: visualMatchClass(fitScore),
    fitLabel: artFitLabel(route),
    fitScore: `${fitScore}%`,
    fitSummary: visualMatchSummary(fitScore),
    fitMessage: visualMatchMessage(fitScore),
    sketchPoints: routesStore.shapePoints.length,
    routePoints: route.previewLatLng.filter((point) => point.length >= 2).length,
  };
});
const canGenerate = computed(() => routesStore.canGenerateShape);
const routingEngineLabel = computed(() => {
  const engine = routesStore.routingEngineName || "OSRM";
  switch (routesStore.routingHealthStatus) {
    case "up":
      return `${engine} online`;
    case "disabled":
      return `${engine} disabled`;
    case "misconfigured":
      return `${engine} misconfigured`;
    case "down":
      return `${engine} offline`;
    default:
      return `${engine} status unknown`;
  }
});
const routingEngineClass = computed(() => {
  switch (routesStore.routingHealthStatus) {
    case "up":
      return "routes-engine-chip routes-engine-chip--up";
    case "disabled":
      return "routes-engine-chip routes-engine-chip--disabled";
    case "misconfigured":
      return "routes-engine-chip routes-engine-chip--warn";
    case "down":
      return "routes-engine-chip routes-engine-chip--down";
    default:
      return "routes-engine-chip";
  }
});
const generateRouteButtonLabel = computed(() => {
  if (routesStore.isLoading) {
    return "Generating art...";
  }
  return "Snap artwork to roads";
});

const routeTypeOptions: Array<{ value: RouteType; label: string }> = [
  { value: "RIDE", label: "Ride" },
  { value: "MTB", label: "MTB" },
  { value: "GRAVEL", label: "Gravel" },
  { value: "RUN", label: "Run" },
  { value: "TRAIL", label: "Trail" },
  { value: "HIKE", label: "Hike" },
];
const routeTypeOptionsWithAvailability = computed(() =>
  routeTypeOptions.map((option) => ({
    ...option,
    disabled: !routesStore.isRouteTypeSupported(option.value),
  })),
);
const unavailableRouteTypeLabels = computed(() =>
  routeTypeOptionsWithAvailability.value
    .filter((option) => option.disabled)
    .map((option) => option.label),
);
const routingProfileSummary = computed(() => {
  const extractProfile = routesStore.routingExtractProfile;
  const effectiveProfile = routesStore.routingEffectiveProfile;
  if (extractProfile === "/opt/bicycle.lua" || effectiveProfile === "cycling") {
    return "OSRM profile: bicycle (Ride / MTB / Gravel)";
  }
  if (extractProfile === "/opt/foot.lua" || effectiveProfile === "walking") {
    return "OSRM profile: foot (Run / Trail / Hike)";
  }
  if (extractProfile === "/opt/car.lua" || effectiveProfile === "driving") {
    return "OSRM profile: car (limited route mode)";
  }
  return "OSRM profile: unknown (all route types enabled)";
});

const nonBlockingGenerationDiagnosticCodes = new Set([
  "DIRECTION_RELAXED",
  "DIRECTION_BEST_EFFORT",
  "BACKTRACKING_RELAXED",
  "ROUTE_TYPE_FALLBACK",
  "START_POINT_SNAPPED",
  "ENGINE_FALLBACK_LEGACY",
  "SELECTION_RELAXED",
  "EMERGENCY_FALLBACK",
]);
const preferredRouteReasonPrefixes = [
  "Shape",
  "Surface fitness",
  "Path ratio",
  "Road graph",
  "Generation engine",
  "Selection profile",
  "Direction",
  "Target mode",
  "History guidance",
  "Anti-backtracking",
];
const hiddenRouteReasonPrefixes = [
  "Directional alignment",
  "Surface mix",
];

interface PresentedDiagnostic {
  code: string;
  title: string;
  message: string;
  tone: "info" | "warn" | "error";
  icon: string;
}

function formatDistance(value: number): string {
  return `${value.toFixed(1)} km`;
}

function formatElevation(value: number): string {
  return `${Math.round(value)} m`;
}

function formatSignedDistanceDelta(value: number): string {
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(1)} km`;
}

function formatSignedPercent(value: number): string {
  const rounded = Math.round(value);
  const sign = rounded > 0 ? "+" : "";
  return `${sign}${rounded}%`;
}

function clampScore(value: number | undefined): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.min(100, value));
}

function scoreMeterStyle(value: number | undefined) {
  return { width: `${Math.round(clampScore(value))}%` };
}

function distanceDeltaClass(deltaRatio: number): string {
  const absoluteDelta = Math.abs(deltaRatio);
  if (absoluteDelta <= 12) {
    return "routes-comparison-value routes-comparison-value--strong";
  }
  if (absoluteDelta <= 35) {
    return "routes-comparison-value routes-comparison-value--mixed";
  }
  return "routes-comparison-value routes-comparison-value--warn";
}

function coordinateDistanceKm(from: number[], to: number[]): number {
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

function polylineDistanceKm(points: number[][]): number {
  if (points.length < 2) {
    return 0;
  }
  let distance = 0;
  for (let index = 1; index < points.length; index += 1) {
    distance += coordinateDistanceKm(points[index - 1], points[index]);
  }
  return distance;
}

function formatVariantType(value: string): string {
  return value
    .replaceAll("_", " ")
    .toLowerCase()
    .replace(/\b\w/g, (match) => match.toUpperCase());
}

function artFitScore(route: GeneratedRoute): number {
  const global = clampScore(route.score.global);
  const shape = clampScore(route.score.shape);
  return Math.round((shape * 0.90) + (global * 0.10));
}

function artFitLabel(route: GeneratedRoute): string {
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

function visualMatchSummary(score: number): string {
  if (score >= 82) {
    return "Good match";
  }
  if (score >= 68) {
    return "Medium match";
  }
  return "Weak match";
}

function visualMatchMessage(score: number): string {
  if (score >= 82) {
    return "The generated route keeps the sketch readable.";
  }
  if (score >= 68) {
    return "The route follows the idea, but some parts drift from the sketch.";
  }
  return "The route is usable as a fallback, but the drawing is hard to read.";
}

function visualMatchClass(score: number): string {
  if (score >= 82) {
    return "routes-visual-match routes-visual-match--strong";
  }
  if (score >= 68) {
    return "routes-visual-match routes-visual-match--mixed";
  }
  return "routes-visual-match routes-visual-match--weak";
}

function artFitClass(route: GeneratedRoute): string {
  const score = artFitScore(route);
  if (score >= 90) {
    return "route-quality-chip route-quality-chip--strong";
  }
  if (score >= 82) {
    return "route-quality-chip route-quality-chip--ok";
  }
  return "route-quality-chip route-quality-chip--warn";
}

function routeQualityScore(route: GeneratedRoute): number {
  const global = clampScore(route.score.global);
  const roadFitness = clampScore(route.score.roadFitness);
  return Math.round((roadFitness * 0.60) + (global * 0.40));
}

function routeQualityLabel(route: GeneratedRoute): string {
  const score = routeQualityScore(route);
  if (score >= 85) {
    return "Smooth route";
  }
  if (score >= 70) {
    return "Usable route";
  }
  if (score >= 55) {
    return "Check route";
  }
  return "Low confidence";
}

function scoreBandClass(value: number | undefined): string {
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

function routeSourceLabel(route: GeneratedRoute): string {
  if (route.isRoadGraphGenerated) {
    return "OSRM road snap";
  }
  return formatVariantType(route.variantType);
}

function highlightedRouteReasons(route: GeneratedRoute): string[] {
  const cleanedReasons = route.reasons
    .map((reason) => reason.trim())
    .filter((reason) =>
      reason.length > 0
      && !hiddenRouteReasonPrefixes.some((prefix) => reason.toLowerCase().startsWith(prefix.toLowerCase()))
    );
  const preferredReasons = cleanedReasons.filter((reason) =>
    preferredRouteReasonPrefixes.some((prefix) => reason.toLowerCase().startsWith(prefix.toLowerCase()))
  );
  const selectedReasons = preferredReasons.length > 0 ? preferredReasons : cleanedReasons;
  return selectedReasons.slice(0, 3);
}

function routeTitle(route: GeneratedRoute, index: number): string {
  const title = route.title.trim();
  if (title.length > 0 && title !== route.routeId) {
    return title;
  }
  return `Proposal ${index + 1}`;
}

function diagnosticTitle(code: string): string {
  switch (code) {
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
    case "DIRECTION_RELAXED":
    case "DIRECTION_BEST_EFFORT":
      return "Heading softened";
    case "SELECTION_RELAXED":
      return "Selection softened";
    case "EMERGENCY_FALLBACK":
      return "Best available route";
    default:
      return code.replaceAll("_", " ").toLowerCase().replace(/\b\w/g, (match) => match.toUpperCase());
  }
}

function diagnosticMessage(diagnostic: RouteGenerationDiagnostic): string {
  switch (diagnostic.code) {
    case "NO_CANDIDATE":
      return "The sketch could not be matched to routable roads.";
    case "FAILURE_SUMMARY":
      return diagnostic.message.replace("Try simplifying the shape or moving the start point.", "Simplify the sketch, move the start point, or try fewer tight turns.");
    case "ROUTE_TYPE_FALLBACK":
      return "The requested activity style was changed to keep the route practicable.";
    case "START_POINT_SNAPPED":
      return "The start was moved to the closest routable point.";
    case "NON_SHAPE_CANDIDATES_IGNORED":
      return "Existing activities were available, but Strava Art only returns OSRM routes generated from the sketch.";
    case "ENGINE_CACHE_FALLBACK":
      return "OSRM did not produce a better candidate, so a known historical route was returned.";
    case "ENGINE_FALLBACK_LEGACY":
      return "A backup routing strategy was used to keep a proposal available.";
    case "BACKTRACKING_RELAXED":
      return "Some overlap was allowed to preserve the artwork.";
    case "DIRECTION_RELAXED":
    case "DIRECTION_BEST_EFFORT":
      return "The internal heading preference was softened to keep the route available.";
    case "SELECTION_RELAXED":
      return "Selection rules were softened to return a usable proposal.";
    case "EMERGENCY_FALLBACK":
      return "The best available generated route was selected despite weak matching.";
    default:
      return diagnostic.message;
  }
}

function diagnosticTone(code: string): PresentedDiagnostic["tone"] {
  if (code === "NO_CANDIDATE" || code === "FAILURE_SUMMARY") {
    return "error";
  }
  if (nonBlockingGenerationDiagnosticCodes.has(code)) {
    return "warn";
  }
  return "info";
}

function diagnosticIcon(code: string): string {
  if (code === "NO_CANDIDATE" || code === "FAILURE_SUMMARY") {
    return "fa-solid fa-triangle-exclamation";
  }
  if (code === "START_POINT_SNAPPED") {
    return "fa-solid fa-location-dot";
  }
  if (code === "ROUTE_TYPE_FALLBACK") {
    return "fa-solid fa-route";
  }
  if (code.includes("FALLBACK")) {
    return "fa-solid fa-life-ring";
  }
  return "fa-solid fa-circle-info";
}

function presentDiagnostic(diagnostic: RouteGenerationDiagnostic): PresentedDiagnostic {
  return {
    code: diagnostic.code,
    title: diagnosticTitle(diagnostic.code),
    message: diagnosticMessage(diagnostic),
    tone: diagnosticTone(diagnostic.code),
    icon: diagnosticIcon(diagnostic.code),
  };
}

function openGpxFilePicker(mode: "replace" | "append" = "replace") {
  gpxImportMode.value = mode;
  gpxFileInput.value?.click();
}

function openTraceImagePicker() {
  traceImageFileInput.value?.click();
}

async function onGpxFileSelected(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  if (files.length === 0) {
    return;
  }
  let totalImportedPoints = 0;
  let importedFileCount = 0;
  let invalidFileCount = 0;
  let shouldAppend = gpxImportMode.value === "append";

  try {
    for (const file of files) {
      try {
        const content = await file.text();
        const importedPoints = routesStore.importShapeFromGpx(content, { append: shouldAppend });
        if (importedPoints < 2) {
          invalidFileCount += 1;
          continue;
        }
        totalImportedPoints += importedPoints;
        importedFileCount += 1;
        shouldAppend = true;
      } catch {
        invalidFileCount += 1;
      }
    }

    if (totalImportedPoints < 2) {
      showToast("GPX invalide: aucun tracé exploitable trouvé.", ToastTypeEnum.WARN);
      return;
    }
    redrawMapLayers({ fitBounds: true });
    const modeLabel = gpxImportMode.value === "append" ? "ajoutés" : "importés";
    const fileLabel = importedFileCount > 1 ? "fichiers" : "fichier";
    showToast(`GPX ${modeLabel} (${importedFileCount} ${fileLabel}, ${totalImportedPoints} points).`);
    if (invalidFileCount > 0) {
      showToast(`${invalidFileCount} fichier(s) ignoré(s): format GPX invalide.`, ToastTypeEnum.WARN, 4200);
    }
  } finally {
    input.value = "";
    gpxImportMode.value = "replace";
  }
}

function renderTraceImageLayer() {
  if (!map.value) {
    return;
  }
  if (traceImageLayer.value) {
    traceImageLayer.value.remove();
    traceImageLayer.value = undefined;
  }
  if (!traceImageUrl.value || !traceImageBounds.value) {
    return;
  }
  traceImageLayer.value = L.imageOverlay(traceImageUrl.value, traceImageBounds.value, {
    opacity: 0.38,
    interactive: false,
  }).addTo(map.value);
}

async function onTraceImageSelected(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }
  try {
    const dataUrl = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.addEventListener("load", () => resolve(String(reader.result ?? "")));
      reader.addEventListener("error", () => reject(new Error("image read failed")));
      reader.readAsDataURL(file);
    });
    const currentMap = map.value;
    if (!currentMap) {
      return;
    }
    const bounds = routesStore.shapePoints.length >= 2
      ? L.latLngBounds(routesStore.shapePoints.map((point) => L.latLng(point[0], point[1]))).pad(0.35)
      : currentMap.getBounds().pad(-0.18);
    traceImageUrl.value = dataUrl;
    traceImageName.value = file.name;
    traceImageBounds.value = bounds;
    renderTraceImageLayer();
    redrawMapLayers({ fitBounds: false });
    showToast("Trace image loaded");
  } catch {
    showToast("Unable to load trace image.", ToastTypeEnum.ERROR, 4200);
  } finally {
    input.value = "";
  }
}

function clearTraceImage() {
  traceImageUrl.value = "";
  traceImageName.value = "";
  traceImageBounds.value = null;
  if (traceImageLayer.value) {
    traceImageLayer.value.remove();
    traceImageLayer.value = undefined;
  }
}

function showToast(message: string, type: ToastTypeEnum = ToastTypeEnum.NORMAL, timeout = 2800) {
  uiStore.showToast({
    id: `routes-${Date.now()}-${Math.random()}`,
    message,
    type,
    timeout,
  });
}

function initMap() {
  if (!mapContainer.value) {
    return;
  }
  if (map.value) {
    map.value.remove();
  }

  map.value = L.map(mapContainer.value, { zoomControl: true });
  map.value.setView([45.1885, 5.7245], 10);
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
  }).addTo(map.value);

  map.value.on("click", (event: L.LeafletMouseEvent) => {
    if (routesStore.mode === "SHAPE" && routesStore.isDrawingShape) {
      routesStore.addShapePoint(event.latlng.lat, event.latlng.lng);
      redrawMapLayers({ fitBounds: false });
      return;
    }
    routesStore.setStartPoint(event.latlng.lat, event.latlng.lng);
    persistStartPoint(event.latlng.lat, event.latlng.lng);
    redrawMapLayers({ fitBounds: false });
  });
}

function getStoredStartPoint(): { lat: number; lng: number } | null {
  try {
    const raw = localStorage.getItem("routes-last-location");
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as { lat?: number; lng?: number };
    if (typeof parsed.lat !== "number" || typeof parsed.lng !== "number") {
      return null;
    }
    return { lat: parsed.lat, lng: parsed.lng };
  } catch {
    return null;
  }
}

function persistStartPoint(lat: number, lng: number) {
  try {
    localStorage.setItem("routes-last-location", JSON.stringify({ lat, lng }));
  } catch {
    // best effort only
  }
}

function applyStartPoint(lat: number, lng: number, zoom = 12) {
  routesStore.setStartPoint(lat, lng);
  if (map.value) {
    map.value.setView([lat, lng], zoom);
    map.value.invalidateSize();
  }
  redrawMapLayers({ fitBounds: false });
}

function collectAllMapPoints(): L.LatLng[] {
  const points: L.LatLng[] = [];
  if (routesStore.startPoint) {
    points.push(L.latLng(routesStore.startPoint.lat, routesStore.startPoint.lng));
  }
  routesStore.shapePoints.forEach((point) => {
    if (point.length >= 2) {
      points.push(L.latLng(point[0], point[1]));
    }
  });
  selectedRoute.value?.previewLatLng.forEach((point) => {
    if (point.length >= 2) {
      points.push(L.latLng(point[0], point[1]));
    }
  });
  return points;
}

function redrawMapLayers(options: { fitBounds?: boolean } = {}) {
  if (!map.value) {
    return;
  }

  if (startMarker.value) {
    startMarker.value.remove();
    startMarker.value = undefined;
  }
  if (shapePolylineLayer.value) {
    shapePolylineLayer.value.remove();
    shapePolylineLayer.value = undefined;
  }
  if (selectedRouteOutlineLayer.value) {
    selectedRouteOutlineLayer.value.remove();
    selectedRouteOutlineLayer.value = undefined;
  }
  if (selectedRouteLayer.value) {
    selectedRouteLayer.value.remove();
    selectedRouteLayer.value = undefined;
  }

  renderTraceImageLayer();

  if (selectedRoute.value && selectedRoute.value.previewLatLng.length >= 2) {
    const routeLatLngs = selectedRoute.value.previewLatLng
      .filter((point) => point.length >= 2)
      .map((point) => L.latLng(point[0], point[1]));
    if (routeLatLngs.length >= 2) {
      selectedRouteOutlineLayer.value = L.polyline(routeLatLngs, {
        color: "#ffffff",
        weight: 8,
        opacity: 0.88,
      }).addTo(map.value);
      selectedRouteLayer.value = L.polyline(routeLatLngs, {
        color: "#fc4c02",
        weight: 4,
        opacity: 0.95,
      }).addTo(map.value);
      selectedRouteLayer.value.bindTooltip("Generated route", { direction: "top" });
    }
  }

  if (routesStore.shapePoints.length >= 2) {
    const shapeLatLngs = routesStore.shapePoints.map((point) => L.latLng(point[0], point[1]));
    shapePolylineLayer.value = L.polyline(shapeLatLngs, {
      color: "#7b61ff",
      weight: 3,
      dashArray: "8 8",
      opacity: 0.95,
    }).addTo(map.value);
    shapePolylineLayer.value.bindTooltip("Original sketch", { direction: "top" });
  }

  if (routesStore.startPoint) {
    startMarker.value = L.circleMarker([routesStore.startPoint.lat, routesStore.startPoint.lng], {
      radius: 7,
      color: "#0d6efd",
      weight: 3,
      fillColor: "#6ea8fe",
      fillOpacity: 0.85,
    }).addTo(map.value);
    startMarker.value.bindTooltip("Start point", { direction: "top" });
  }

  const allPoints = collectAllMapPoints();
  if (options.fitBounds !== false && allPoints.length > 0) {
    const bounds = L.latLngBounds(allPoints);
    if (bounds.isValid()) {
      map.value.fitBounds(bounds, { padding: [26, 26] });
    }
  }
}

function describeGeolocationError(error: GeolocationPositionError): string {
  switch (error.code) {
    case error.PERMISSION_DENIED:
      return "permission denied";
    case error.POSITION_UNAVAILABLE:
      return "position unavailable";
    case error.TIMEOUT:
      return "timeout";
    default:
      return error.message || "unknown error";
  }
}

async function requestMyLocation(silent = false) {
  if (isLocating.value) {
    return;
  }
  if (!navigator.geolocation) {
    if (!silent) {
      showToast("Geolocation is not available in this browser", ToastTypeEnum.ERROR, 3800);
    }
    return;
  }
  const host = window.location.hostname;
  const isLocalhost = host === "localhost" || host === "127.0.0.1" || host === "::1";
  if (!window.isSecureContext && !isLocalhost) {
    if (!silent) {
      showToast("Geolocation requires HTTPS outside localhost", ToastTypeEnum.ERROR, 4200);
    }
    return;
  }
  isLocating.value = true;
  navigator.geolocation.getCurrentPosition(
    (position) => {
      const lat = position.coords.latitude;
      const lng = position.coords.longitude;
      applyStartPoint(lat, lng, 12);
      persistStartPoint(lat, lng);
      if (!silent) {
        showToast("Start point set from your current location");
      }
      isLocating.value = false;
    },
    (error) => {
      const fallback = getStoredStartPoint();
      if (fallback) {
        applyStartPoint(fallback.lat, fallback.lng, 11);
        if (!silent) {
          showToast("Unable to access live location, using your last known start point", ToastTypeEnum.WARN, 4200);
        }
      } else {
        if (map.value) {
          const center = map.value.getCenter();
          applyStartPoint(center.lat, center.lng, map.value.getZoom());
          persistStartPoint(center.lat, center.lng);
        }
        if (!silent) {
          const reason = describeGeolocationError(error);
          showToast(`Unable to access your location (${reason}). Using current map center as start point.`, ToastTypeEnum.WARN, 4600);
        }
      }
      isLocating.value = false;
    },
    {
      enableHighAccuracy: false,
      timeout: 20000,
      maximumAge: 10 * 60 * 1000,
    },
  );
}

async function useMyLocation() {
  await requestMyLocation(false);
}

function undoShapePoint() {
  routesStore.undoLastShapePoint();
  redrawMapLayers({ fitBounds: false });
}

function resetStartPoint() {
  routesStore.clearStartPoint();
  redrawMapLayers({ fitBounds: false });
  showToast("Start point cleared. Click the map or use your location to set a new start point.");
}

function currentTemplateCenter(): { lat: number; lng: number } {
  if (routesStore.startPoint) {
    return routesStore.startPoint;
  }
  const center = map.value?.getCenter();
  if (center) {
    return { lat: center.lat, lng: center.lng };
  }
  return { lat: 45.1885, lng: 5.7245 };
}

function applyShapeTemplate(template: BuiltInShapeTemplateKey) {
  const loaded = routesStore.applyBuiltInShapeTemplate(template, currentTemplateCenter());
  if (loaded) {
    redrawMapLayers({ fitBounds: true });
    showToast(`${template} sketch loaded`);
  }
}

function saveCurrentShapeTemplate() {
  const saved = routesStore.saveCurrentShapeTemplate(saveShapeName.value);
  if (!saved) {
    showToast("Draw or import a sketch before saving a template.", ToastTypeEnum.WARN, 3600);
    return;
  }
  saveShapeName.value = "";
  showToast(`Sketch template "${saved.name}" saved`);
}

function loadSavedShapeTemplate(templateId: string) {
  if (!routesStore.loadSavedShapeTemplate(templateId)) {
    showToast("Saved sketch not found.", ToastTypeEnum.WARN, 3600);
    return;
  }
  redrawMapLayers({ fitBounds: true });
  showToast("Saved sketch loaded");
}

function deleteSavedShapeTemplate(templateId: string) {
  if (routesStore.deleteSavedShapeTemplate(templateId)) {
    showToast("Saved sketch deleted");
  }
}

function toggleFreestyleMode(event: Event) {
  const input = event.target as HTMLInputElement;
  routesStore.setFreestyleMode(input.checked);
}

function toggleAutoFitBeforeRouting(event: Event) {
  const input = event.target as HTMLInputElement;
  routesStore.setAutoFitBeforeRouting(input.checked);
}

function exportSketchGpx() {
  try {
    routesStore.exportCurrentShapeGpx(saveShapeName.value || "strava-art-sketch");
    showToast("Sketch GPX exported");
  } catch (error) {
    const message = error instanceof Error && error.message === "shape is required"
      ? "Draw or import a sketch before exporting GPX."
      : "Unable to export sketch GPX.";
    showToast(message, ToastTypeEnum.ERROR, 4200);
  }
}

function exportSketchTcx() {
  try {
    routesStore.exportCurrentShapeTcx(saveShapeName.value || "strava-art-sketch");
    showToast("Sketch TCX exported");
  } catch (error) {
    const message = error instanceof Error && error.message === "shape is required"
      ? "Draw or import a sketch before exporting TCX."
      : "Unable to export sketch TCX.";
    showToast(message, ToastTypeEnum.ERROR, 4200);
  }
}

function exportSketchPng() {
  const points = routesStore.shapePoints.filter((point) => point.length >= 2);
  if (points.length < 2) {
    showToast("Draw or import a sketch before exporting PNG.", ToastTypeEnum.WARN, 3600);
    return;
  }
  const canvas = document.createElement("canvas");
  canvas.width = 900;
  canvas.height = 600;
  const context = canvas.getContext("2d");
  if (!context) {
    showToast("Unable to export sketch PNG.", ToastTypeEnum.ERROR, 4200);
    return;
  }

  const padding = 52;
  const latitudes = points.map((point) => point[0]);
  const longitudes = points.map((point) => point[1]);
  const minLat = Math.min(...latitudes);
  const maxLat = Math.max(...latitudes);
  const minLng = Math.min(...longitudes);
  const maxLng = Math.max(...longitudes);
  const latRange = Math.max(0.00001, maxLat - minLat);
  const lngRange = Math.max(0.00001, maxLng - minLng);
  const drawableWidth = canvas.width - (padding * 2);
  const drawableHeight = canvas.height - (padding * 2);
  const scale = Math.min(drawableWidth / lngRange, drawableHeight / latRange);
  const usedWidth = lngRange * scale;
  const usedHeight = latRange * scale;
  const offsetX = (canvas.width - usedWidth) / 2;
  const offsetY = (canvas.height - usedHeight) / 2;
  const project = (point: number[]) => ({
    x: offsetX + ((point[1] - minLng) * scale),
    y: offsetY + ((maxLat - point[0]) * scale),
  });

  context.fillStyle = "#ffffff";
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.strokeStyle = "#dfe6f1";
  context.lineWidth = 2;
  context.strokeRect(16, 16, canvas.width - 32, canvas.height - 32);
  context.setLineDash([12, 10]);
  context.lineCap = "round";
  context.lineJoin = "round";
  context.strokeStyle = "#6f51ff";
  context.lineWidth = 6;
  context.beginPath();
  points.forEach((point, index) => {
    const projected = project(point);
    if (index === 0) {
      context.moveTo(projected.x, projected.y);
      return;
    }
    context.lineTo(projected.x, projected.y);
  });
  context.stroke();
  context.setLineDash([]);
  context.fillStyle = "#242933";
  context.font = "700 22px system-ui, -apple-system, BlinkMacSystemFont, sans-serif";
  context.fillText(saveShapeName.value.trim() || "Strava Art sketch", 30, canvas.height - 28);

  canvas.toBlob((blob) => {
    if (!blob) {
      showToast("Unable to export sketch PNG.", ToastTypeEnum.ERROR, 4200);
      return;
    }
    const objectUrl = URL.createObjectURL(blob);
    const safeName = (saveShapeName.value.trim().toLowerCase().replace(/[^a-z0-9-]+/g, "-").replace(/^-+|-+$/g, ""))
      || "strava-art-sketch";
    try {
      const link = document.createElement("a");
      link.href = objectUrl;
      link.download = `${safeName}.png`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      showToast("Sketch PNG exported");
    } finally {
      URL.revokeObjectURL(objectUrl);
    }
  }, "image/png");
}

function setStartToMapCenter() {
  const center = map.value?.getCenter();
  if (!center) {
    return;
  }
  routesStore.setStartPoint(center.lat, center.lng);
  persistStartPoint(center.lat, center.lng);
  redrawMapLayers({ fitBounds: false });
  showToast("Start point set from map center");
}

function visibleRoadRadiusKmAroundStart(): number | undefined {
  const currentMap = map.value;
  const start = routesStore.startPoint;
  if (!currentMap || !start) {
    return undefined;
  }
  const bounds = currentMap.getBounds();
  const anchor = L.latLng(start.lat, start.lng);
  if (!bounds.contains(anchor)) {
    return undefined;
  }
  const distancesMeters = [
    currentMap.distance(anchor, L.latLng(bounds.getNorth(), start.lng)),
    currentMap.distance(anchor, L.latLng(bounds.getSouth(), start.lng)),
    currentMap.distance(anchor, L.latLng(start.lat, bounds.getEast())),
    currentMap.distance(anchor, L.latLng(start.lat, bounds.getWest())),
  ].filter((distance) => Number.isFinite(distance) && distance > 0);
  if (distancesMeters.length === 0) {
    return undefined;
  }
  return Math.min(...distancesMeters) / 1000;
}

function autoFitSketchBeforeRouting(): boolean {
  if (!routesStore.autoFitBeforeRouting || !routesStore.startPoint || !canTransformShape.value) {
    return false;
  }
  const fitted = routesStore.fitShapeToStart({
    viewportRadiusKm: visibleRoadRadiusKmAroundStart(),
  });
  if (fitted) {
    redrawMapLayers({ fitBounds: false });
  }
  return fitted;
}

function runCorrectionSuggestion(suggestion: CorrectionSuggestion) {
  if (!suggestion.action || suggestion.disabled) {
    return;
  }
  switch (suggestion.action) {
    case "simplify":
      transformShape("simplify");
      break;
    case "smooth":
      transformShape("smooth");
      break;
    case "center":
      transformShape("center");
      break;
    case "scaleDown":
      transformShape("scaleDown");
      break;
    case "scaleUp":
      transformShape("scaleUp");
      break;
    case "useLocation":
      setStartToMapCenter();
      break;
    case "generate":
      void generateRoutes();
      break;
    case "heart":
      applyShapeTemplate("heart");
      break;
    case "circle":
      applyShapeTemplate("circle");
      break;
    default:
      break;
  }
}

function shapeNudgeStep(): { lat: number; lng: number } {
  const currentMap = map.value;
  if (!currentMap) {
    return { lat: 0.002, lng: 0.002 };
  }
  const bounds = currentMap.getBounds();
  const latStep = Math.max(0.0002, Math.abs(bounds.getNorth() - bounds.getSouth()) * 0.025);
  const lngStep = Math.max(0.0002, Math.abs(bounds.getEast() - bounds.getWest()) * 0.025);
  return { lat: latStep, lng: lngStep };
}

function nudgeShape(direction: "north" | "south" | "east" | "west") {
  const step = shapeNudgeStep();
  const moved = routesStore.translateShape(
    direction === "north" ? step.lat : direction === "south" ? -step.lat : 0,
    direction === "east" ? step.lng : direction === "west" ? -step.lng : 0,
  );
  if (moved) {
    redrawMapLayers({ fitBounds: false });
  }
}

function transformShape(action: "scaleDown" | "scaleUp" | "rotateLeft" | "rotateRight" | "center" | "smooth" | "simplify" | "undo" | "redo") {
  let changed = false;
  switch (action) {
    case "scaleDown":
      changed = routesStore.scaleShape(0.9);
      break;
    case "scaleUp":
      changed = routesStore.scaleShape(1.1);
      break;
    case "rotateLeft":
      changed = routesStore.rotateShape(-15);
      break;
    case "rotateRight":
      changed = routesStore.rotateShape(15);
      break;
    case "center":
      changed = routesStore.centerShapeOnStart();
      if (!changed && !routesStore.startPoint) {
        showToast("Set a start point before centering the sketch.", ToastTypeEnum.WARN, 3600);
      }
      break;
    case "smooth":
      changed = routesStore.smoothShape();
      break;
    case "simplify":
      changed = routesStore.simplifyShape();
      break;
    case "undo":
      changed = routesStore.undoShapeTransform();
      break;
    case "redo":
      changed = routesStore.redoShapeTransform();
      break;
    default:
      changed = false;
  }
  if (changed) {
    redrawMapLayers({ fitBounds: action === "center" });
  }
}

async function generateRoutes() {
  try {
    autoFitSketchBeforeRouting();
    await routesStore.generateRoutes();
    redrawMapLayers();
    if (!routesStore.hasRoutes) {
      const message = productFailureSummary.value?.message ?? productGenerationDiagnostics.value[0]?.message;
      const displayMessage = message
        ? `No road-snapped route. ${message}`
        : "No road-snapped route for this artwork.";
      showToast(displayMessage, ToastTypeEnum.ERROR, 5000);
      return;
    }
    if (routesStore.hasRoutes) {
      const nonBlockingDiagnostic = routesStore.generationDiagnostics.find((diagnostic) =>
        nonBlockingGenerationDiagnosticCodes.has(diagnostic.code)
      );
      if (nonBlockingDiagnostic) {
        showToast(nonBlockingDiagnostic.message, ToastTypeEnum.WARN, 4200);
      }
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unable to generate routes";
    showToast(message, ToastTypeEnum.ERROR, 4200);
  }
}

function pickRoute(routeId: string) {
  routesStore.setSelectedRoute(routeId);
  redrawMapLayers({ fitBounds: true });
}

async function exportRoute(route: GeneratedRoute) {
  routesStore.setSelectedRoute(route.routeId);
  redrawMapLayers({ fitBounds: true });
  isExporting.value = true;
  try {
    await routesStore.exportRouteGpx(route.routeId);
    showToast("GPX exported successfully");
  } catch (error) {
    showToast("Unable to export GPX for this route", ToastTypeEnum.ERROR, 4200);
    console.error(error);
  } finally {
    isExporting.value = false;
  }
}

async function exportSelectedRoute() {
  if (!selectedRoute.value) {
    return;
  }
  await exportRoute(selectedRoute.value);
}

watch(
  () => [routesStore.startPoint, routesStore.shapePoints, selectedRoute.value?.routeId],
  () => redrawMapLayers({ fitBounds: false }),
  { deep: true },
);

onMounted(async () => {
  await nextTick();
  routesStore.setMode("SHAPE");
  routesStore.loadSavedShapeTemplates();
  initMap();
  await routesStore.refreshRoutingHealth();
  const storedStartPoint = getStoredStartPoint();
  if (storedStartPoint) {
    applyStartPoint(storedStartPoint.lat, storedStartPoint.lng, 11);
  }
  redrawMapLayers({ fitBounds: false });
  requestMyLocation(true);
});

onBeforeUnmount(() => {
  if (map.value) {
    map.value.remove();
    map.value = undefined;
  }
});
</script>

<template>
  <section class="routes-view">
    <header class="routes-panel routes-head">
      <div class="routes-title-block">
        <div>
          <span class="routes-kicker">GPS drawing studio</span>
          <h1>Strava Art</h1>
        </div>
        <div class="routes-head-actions">
          <span class="routes-mode-chip">
            <i class="fa-solid fa-pen-nib" aria-hidden="true" />
            Draw art
          </span>
          <span :class="routingEngineClass">
            <span class="routes-engine-dot" />
            {{ routingEngineLabel }}
          </span>
        </div>
      </div>
      <div class="routes-art-steps" aria-label="Strava Art workflow">
        <span><i class="fa-solid fa-pencil" aria-hidden="true" /> Sketch</span>
        <span><i class="fa-solid fa-magnet" aria-hidden="true" /> OSRM snap</span>
        <span><i class="fa-solid fa-file-export" aria-hidden="true" /> GPX export</span>
      </div>
    </header>

    <section class="routes-panel routes-layout">
      <aside class="routes-controls">
        <button
          type="button"
          class="btn btn-outline-primary routes-location-btn"
          :disabled="isLocating"
          @click="useMyLocation"
        >
          <i class="fa-solid fa-location-crosshairs" aria-hidden="true" />
          {{ isLocating ? "Locating..." : "Use my location" }}
        </button>

        <label class="routes-field">
          <span>Activity style</span>
          <select
            v-model="routesStore.routeType"
            class="form-select"
          >
            <option
              v-for="option in routeTypeOptionsWithAvailability"
              :key="option.value"
              :value="option.value"
              :disabled="option.disabled"
            >
              {{ option.label }}
            </option>
          </select>
          <small class="routes-hint">{{ routingProfileSummary }}</small>
          <small
            v-if="unavailableRouteTypeLabels.length > 0"
            class="routes-hint"
          >
            Disabled with current profile: {{ unavailableRouteTypeLabels.join(", ") }}
          </small>
        </label>

        <button
          type="button"
          class="btn btn-outline-secondary btn-sm"
          @click="resetStartPoint"
        >
          <i class="fa-solid fa-crosshairs" aria-hidden="true" />
          Reset start point
        </button>

        <label class="routes-form-check routes-form-check--stacked">
          <span class="routes-checkline">
            <input
              type="checkbox"
              :checked="routesStore.autoFitBeforeRouting"
              @change="toggleAutoFitBeforeRouting"
            >
            <span>Auto-fit sketch</span>
          </span>
          <small>Center and scale before routing.</small>
        </label>

        <div class="routes-shape-tools">
          <div class="routes-shape-tools-head">
            <strong>Artwork sketch</strong>
            <span>{{ routesStore.shapePoints.length }} point(s)</span>
          </div>
          <button
            type="button"
            class="btn btn-outline-secondary"
            @click="routesStore.toggleShapeDrawing"
          >
            <i class="fa-solid fa-pen-nib" aria-hidden="true" />
            {{ routesStore.isDrawingShape ? "Stop drawing" : "Draw sketch" }}
          </button>
          <button
            type="button"
            class="btn btn-outline-secondary"
            @click="openGpxFilePicker('replace')"
          >
            <i class="fa-solid fa-file-import" aria-hidden="true" />
            Import GPX (replace)
          </button>
          <button
            type="button"
            class="btn btn-outline-secondary"
            @click="openGpxFilePicker('append')"
          >
            <i class="fa-solid fa-plus" aria-hidden="true" />
            Import GPX (append)
          </button>
          <button
            type="button"
            class="btn btn-outline-secondary"
            :disabled="routesStore.shapePoints.length === 0"
            @click="undoShapePoint"
          >
            <i class="fa-solid fa-rotate-left" aria-hidden="true" />
            Undo last point
          </button>
          <input
            ref="gpxFileInput"
            type="file"
            class="routes-gpx-input"
            accept=".gpx,application/gpx+xml,application/xml,text/xml"
            multiple
            @change="onGpxFileSelected"
          >
          <button
            type="button"
            class="btn btn-outline-danger"
            :disabled="routesStore.shapePoints.length === 0"
            @click="routesStore.clearShape"
          >
            <i class="fa-solid fa-trash" aria-hidden="true" />
            Clear shape
          </button>
          <small class="routes-hint">
            Click on the map while drawing to add points.
          </small>
        </div>

        <div class="routes-library-tools">
          <div class="routes-library-tools-head">
            <strong>Shape library</strong>
            <span>{{ routesStore.savedShapeTemplateCount }} saved</span>
          </div>
          <div class="routes-template-row">
            <label class="routes-field routes-field--compact">
              <span>Simple shape</span>
              <select
                v-model="selectedShapeTemplate"
                class="form-select form-select-sm"
              >
                <option
                  v-for="template in builtInShapeTemplates"
                  :key="template.key"
                  :value="template.key"
                >
                  {{ template.label }}
                </option>
              </select>
            </label>
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              @click="applyShapeTemplate(selectedShapeTemplate)"
            >
              <i class="fa-solid fa-shapes" aria-hidden="true" />
              Load
            </button>
          </div>
          <div class="routes-image-row">
            <input
              ref="traceImageFileInput"
              type="file"
              class="routes-gpx-input"
              accept="image/png,image/jpeg,image/webp,image/svg+xml"
              @change="onTraceImageSelected"
            >
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              @click="openTraceImagePicker"
            >
              <i class="fa-solid fa-image" aria-hidden="true" />
              Import image
            </button>
            <button
              type="button"
              class="btn btn-outline-danger btn-sm"
              :disabled="!traceImageUrl"
              @click="clearTraceImage"
            >
              <i class="fa-solid fa-eye-slash" aria-hidden="true" />
              Clear
            </button>
          </div>
          <small
            v-if="traceImageName"
            class="routes-hint"
          >
            {{ traceImageName }}
          </small>
          <div class="routes-save-template">
            <span>Save sketch template</span>
            <div class="routes-save-row">
              <input
                v-model="saveShapeName"
                type="text"
                maxlength="48"
                class="form-control form-control-sm"
                placeholder="Template name"
                @keydown.enter.prevent="saveCurrentShapeTemplate"
              >
              <button
                type="button"
                class="btn btn-outline-primary btn-sm"
                :disabled="!canTransformShape"
                @click="saveCurrentShapeTemplate"
              >
                <i class="fa-solid fa-floppy-disk" aria-hidden="true" />
                Save template
              </button>
            </div>
          </div>
          <div
            v-if="routesStore.savedShapeTemplates.length > 0"
            class="routes-saved-list"
          >
            <div
              v-for="template in routesStore.savedShapeTemplates"
              :key="template.id"
              class="routes-saved-item"
            >
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                @click="loadSavedShapeTemplate(template.id)"
              >
                <i class="fa-solid fa-folder-open" aria-hidden="true" />
                {{ template.name }}
              </button>
              <button
                type="button"
                class="btn btn-outline-danger btn-sm"
                :aria-label="`Delete ${template.name}`"
                @click="deleteSavedShapeTemplate(template.id)"
              >
                <i class="fa-solid fa-trash" aria-hidden="true" />
              </button>
            </div>
          </div>
          <label class="routes-freestyle-toggle">
            <input
              type="checkbox"
              :checked="routesStore.freestyleMode"
              @change="toggleFreestyleMode"
            >
            <span>Freestyle exports</span>
          </label>
          <div class="routes-export-row">
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="!routesStore.freestyleMode || !canTransformShape"
              @click="exportSketchGpx"
            >
              <i class="fa-solid fa-file-export" aria-hidden="true" />
              GPX
            </button>
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="!routesStore.freestyleMode || !canTransformShape"
              @click="exportSketchTcx"
            >
              <i class="fa-solid fa-file-export" aria-hidden="true" />
              TCX
            </button>
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="!canTransformShape"
              @click="exportSketchPng"
            >
              <i class="fa-solid fa-image" aria-hidden="true" />
              PNG
            </button>
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              disabled
              title="FIT export needs a binary FIT encoder"
            >
              FIT
            </button>
          </div>
        </div>

      </aside>

      <div class="routes-map-panel">
        <div class="routes-map-head">
          <span class="routes-map-title">Art canvas</span>
          <div class="routes-map-actions">
            <button
              type="button"
              class="btn btn-primary btn-sm routes-map-generate-btn"
              :disabled="routesStore.isLoading || !canGenerate"
              @click="generateRoutes"
            >
              <i class="fa-solid fa-route" aria-hidden="true" />
              {{ generateRouteButtonLabel }}
            </button>
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="!selectedRoute || isExporting"
              @click="exportSelectedRoute"
            >
              <i class="fa-solid fa-download" aria-hidden="true" />
              {{ isExporting ? "Exporting..." : "Export GPX" }}
            </button>
          </div>
        </div>
        <div class="routes-canvas-tools">
          <div class="routes-canvas-tools-head">
            <strong>Transform sketch</strong>
            <span>{{ routesStore.canUndoShapeTransform ? "Undo available" : "Ready" }}</span>
          </div>
          <div class="routes-canvas-toolbar">
            <div class="routes-canvas-tool-group" aria-label="Move sketch">
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Move sketch north"
                aria-label="Move sketch north"
                @click="nudgeShape('north')"
              >
                <i class="fa-solid fa-arrow-up" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Move sketch west"
                aria-label="Move sketch west"
                @click="nudgeShape('west')"
              >
                <i class="fa-solid fa-arrow-left" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Move sketch east"
                aria-label="Move sketch east"
                @click="nudgeShape('east')"
              >
                <i class="fa-solid fa-arrow-right" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Move sketch south"
                aria-label="Move sketch south"
                @click="nudgeShape('south')"
              >
                <i class="fa-solid fa-arrow-down" aria-hidden="true" />
              </button>
            </div>
            <div class="routes-canvas-tool-group">
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Rotate left"
                @click="transformShape('rotateLeft')"
              >
                <i class="fa-solid fa-rotate-left" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Scale down"
                @click="transformShape('scaleDown')"
              >
                <i class="fa-solid fa-magnifying-glass-minus" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape || !routesStore.startPoint"
                title="Center sketch on start point"
                @click="transformShape('center')"
              >
                <i class="fa-solid fa-crosshairs" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Scale up"
                @click="transformShape('scaleUp')"
              >
                <i class="fa-solid fa-magnifying-glass-plus" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!canTransformShape"
                title="Rotate right"
                @click="transformShape('rotateRight')"
              >
                <i class="fa-solid fa-rotate-right" aria-hidden="true" />
              </button>
            </div>
            <div class="routes-canvas-tool-group">
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="routesStore.shapePoints.length < 4"
                title="Smooth sketch"
                @click="transformShape('smooth')"
              >
                <i class="fa-solid fa-bezier-curve" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="routesStore.shapePoints.length < 3"
                title="Simplify sketch"
                @click="transformShape('simplify')"
              >
                <i class="fa-solid fa-compress" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!routesStore.canUndoShapeTransform"
                title="Undo transform"
                @click="transformShape('undo')"
              >
                <i class="fa-solid fa-rotate-left" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm"
                :disabled="!routesStore.canRedoShapeTransform"
                title="Redo transform"
                @click="transformShape('redo')"
              >
                <i class="fa-solid fa-rotate-right" aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>
        <div
          ref="mapContainer"
          class="routes-map"
        />
        <div class="routes-map-legend" aria-label="Map layers">
          <span class="routes-layer-key routes-layer-key--sketch">
            <span aria-hidden="true" />
            Sketch
          </span>
          <span
            v-if="selectedRoute"
            class="routes-layer-key routes-layer-key--route"
          >
            <span aria-hidden="true" />
            Generated route
          </span>
        </div>
        <div class="routes-assistant-tools routes-assistant-tools--map">
          <div class="routes-assistant-tools-head">
            <strong>Correction assistant</strong>
            <span>{{ correctionSuggestions.length }} hint(s)</span>
          </div>
          <div class="routes-assistant-list">
            <article
              v-for="suggestion in correctionSuggestions"
              :key="suggestion.id"
              class="routes-assistant-item"
            >
              <i :class="suggestion.icon" aria-hidden="true" />
              <div>
                <strong>{{ suggestion.title }}</strong>
                <p>{{ suggestion.message }}</p>
              </div>
              <button
                v-if="suggestion.action"
                type="button"
                class="btn btn-outline-primary btn-sm"
                :disabled="suggestion.disabled"
                @click="runCorrectionSuggestion(suggestion)"
              >
                Apply
              </button>
            </article>
          </div>
        </div>
        <div
          v-if="routeComparisonSummary"
          class="routes-comparison"
          aria-label="Sketch and route comparison"
        >
          <div :class="routeComparisonSummary.fitClass">
            <span>Route follows sketch</span>
            <strong>{{ routeComparisonSummary.fitSummary }}</strong>
            <small>{{ routeComparisonSummary.fitMessage }}</small>
          </div>
          <div>
            <span>Sketch</span>
            <strong>{{ routeComparisonSummary.sketchDistance }}</strong>
            <small>{{ routeComparisonSummary.sketchPoints }} points</small>
          </div>
          <div>
            <span>Route</span>
            <strong>{{ routeComparisonSummary.routeDistance }}</strong>
            <small>{{ routeComparisonSummary.routePoints }} points</small>
          </div>
          <div>
            <span>Distance gap</span>
            <strong :class="routeComparisonSummary.deltaClass">
              {{ routeComparisonSummary.deltaLabel }}
            </strong>
            <small>{{ routeComparisonSummary.deltaRatioLabel }}</small>
          </div>
          <div>
            <span>Visual match</span>
            <strong>{{ routeComparisonSummary.fitScore }}</strong>
            <small>{{ routeComparisonSummary.fitLabel }}</small>
          </div>
        </div>
      </div>
    </section>

    <section class="routes-panel routes-results">
      <header class="routes-results-head">
        <h2>Art proposals</h2>
        <span>{{ routesStore.routes.length }} GPX route(s)</span>
      </header>
      <p
        v-if="!routesStore.hasRoutes"
        class="routes-empty"
      >
        {{ generationDiagnostics.length > 0
          ? "No road-snapped proposal is available for this artwork."
          : "Draw or import artwork to see OSRM proposals here." }}
      </p>
      <div
        v-if="!routesStore.hasRoutes && productFailureSummary"
        class="routes-diagnostic-card routes-diagnostic-card--error"
      >
        <i :class="productFailureSummary.icon" aria-hidden="true" />
        <div>
          <strong>{{ productFailureSummary.title }}</strong>
          <p>{{ productFailureSummary.message }}</p>
        </div>
      </div>
      <div
        v-if="!routesStore.hasRoutes && productGenerationDiagnostics.length > 0"
        class="routes-diagnostics-list"
      >
        <article
          v-for="diagnostic in productGenerationDiagnostics"
          :key="diagnostic.code"
          class="routes-diagnostic-card"
          :class="`routes-diagnostic-card--${diagnostic.tone}`"
        >
          <i :class="diagnostic.icon" aria-hidden="true" />
          <div>
            <strong>{{ diagnostic.title }}</strong>
            <p>{{ diagnostic.message }}</p>
          </div>
        </article>
      </div>

      <div
        v-else
        class="routes-results-grid"
      >
        <article
          v-for="(route, index) in routesStore.routes"
          :key="route.routeId"
          role="button"
          tabindex="0"
          class="route-card"
          :class="{ 'route-card--active': selectedRoute?.routeId === route.routeId }"
          @click="pickRoute(route.routeId)"
          @keydown.enter.space.prevent="pickRoute(route.routeId)"
        >
          <div class="route-card-head">
            <div>
              <strong>{{ routeTitle(route, index) }}</strong>
              <span>{{ routeSourceLabel(route) }}</span>
            </div>
            <span :class="artFitClass(route)">
              {{ artFitLabel(route) }}
            </span>
          </div>

          <div class="route-score-stack">
            <div :class="[scoreBandClass(artFitScore(route)), 'route-score-row--primary']">
              <span>Art fit</span>
              <div class="route-score-meter" aria-hidden="true">
                <span :style="scoreMeterStyle(artFitScore(route))" />
              </div>
              <strong>{{ artFitScore(route) }}%</strong>
            </div>
            <div :class="[scoreBandClass(routeQualityScore(route)), 'route-score-row--secondary']">
              <span>Route quality</span>
              <div class="route-score-meter" aria-hidden="true">
                <span :style="scoreMeterStyle(routeQualityScore(route))" />
              </div>
              <strong>{{ routeQualityScore(route) }}%</strong>
            </div>
          </div>

          <dl class="route-card-metrics">
            <div>
              <dt>Distance</dt>
              <dd>{{ formatDistance(route.distanceKm) }}</dd>
            </div>
            <div>
              <dt>D+</dt>
              <dd>{{ formatElevation(route.elevationGainM) }}</dd>
            </div>
            <div>
              <dt>Time</dt>
              <dd>{{ formatTime(route.durationSec) }}</dd>
            </div>
          </dl>

          <p class="route-card-meta">{{ routeQualityLabel(route) }}</p>

          <ul
            v-if="highlightedRouteReasons(route).length > 0"
            class="route-card-reasons"
          >
            <li
              v-for="reason in highlightedRouteReasons(route)"
              :key="reason"
            >
              {{ reason }}
            </li>
          </ul>

          <div class="route-card-actions">
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              @click.stop="pickRoute(route.routeId)"
            >
              <i class="fa-solid fa-location-dot" aria-hidden="true" />
              Select
            </button>
            <button
              type="button"
              class="btn btn-primary btn-sm"
              :disabled="isExporting"
              @click.stop="exportRoute(route)"
            >
              <i class="fa-solid fa-download" aria-hidden="true" />
              GPX
            </button>
          </div>
        </article>
      </div>
      <div
        v-if="routesStore.hasRoutes && productGenerationDiagnostics.length > 0"
        class="routes-diagnostics-list routes-diagnostics-list--notes"
      >
        <article
          v-for="diagnostic in productGenerationDiagnostics"
          :key="diagnostic.code"
          class="routes-diagnostic-card"
          :class="`routes-diagnostic-card--${diagnostic.tone}`"
        >
          <i :class="diagnostic.icon" aria-hidden="true" />
          <div>
            <strong>{{ diagnostic.title }}</strong>
            <p>{{ diagnostic.message }}</p>
          </div>
        </article>
      </div>
    </section>
  </section>
</template>

<style scoped>
.routes-view {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-bottom: 20px;
}

.routes-panel {
  background: #ffffff;
  border: 1px solid #dfe4ec;
  border-radius: 8px;
  padding: 14px;
  box-shadow: 0 6px 20px rgba(12, 21, 38, 0.05);
}

.routes-gpx-input {
  display: none;
}

.routes-head {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.routes-title-block {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.routes-kicker {
  display: block;
  color: #7a4634;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.routes-title-block h1 {
  margin: 2px 0 0;
  color: #242933;
  font-size: 1.55rem;
  line-height: 1.15;
}

.routes-head-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.routes-controls .btn,
.routes-map-head .btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
}

.routes-mode-chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  border: 1px solid #0d6efd;
  border-radius: 999px;
  background: #eef5ff;
  color: #0d4fb3;
  font-size: 0.84rem;
  font-weight: 800;
  padding: 7px 10px;
}

.routes-art-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.routes-art-steps span {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  border: 1px solid #e7d7cb;
  border-radius: 999px;
  background: #fffaf6;
  color: #5e6578;
  font-size: 0.84rem;
  font-weight: 700;
  padding: 6px 10px;
}

.routes-engine-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  border-radius: 999px;
  border: 1px solid #cfd8e6;
  color: #4d566a;
  background: #f6f8fc;
  font-size: 0.82rem;
  font-weight: 700;
}

.routes-engine-chip--up {
  border-color: #2e9c57;
  color: #1d7f42;
  background: #ecf9f1;
}

.routes-engine-chip--down {
  border-color: #de5b5b;
  color: #b23737;
  background: #fff1f1;
}

.routes-engine-chip--warn {
  border-color: #cf8b2d;
  color: #8f5f1f;
  background: #fff8ec;
}

.routes-engine-chip--disabled {
  border-color: #a3adb9;
  color: #5e6573;
  background: #f1f4f8;
}

.routes-engine-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.routes-layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 12px;
}

.routes-controls {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.routes-location-btn {
  width: 100%;
}

.routes-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.routes-field span {
  font-size: 0.85rem;
  font-weight: 700;
  color: #4d566a;
}

.routes-checkbox-field {
  gap: 6px;
}

.routes-form-check {
  padding: 8px 10px;
  border: 1px solid #d9e2ef;
  border-radius: 10px;
  background: #f8fbff;
}

.routes-form-check .form-check-label {
  font-size: 0.84rem;
  color: #4d566a;
}

.routes-form-check--stacked {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.routes-checkline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #4d566a;
  font-size: 0.84rem;
  font-weight: 800;
}

.routes-form-check--stacked small {
  color: #6f7687;
  font-size: 0.78rem;
}

.routes-shape-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.routes-shape-tools-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #4d566a;
  font-size: 0.85rem;
}

.routes-shape-tools-head span {
  color: #6f7687;
  font-weight: 700;
}

.routes-library-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid #e5ebf4;
  padding-top: 10px;
}

.routes-library-tools-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #4d566a;
  font-size: 0.85rem;
}

.routes-library-tools-head span {
  color: #6f7687;
  font-weight: 700;
}

.routes-template-row,
.routes-image-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
  align-items: end;
}

.routes-template-row .btn,
.routes-image-row .btn,
.routes-save-row .btn,
.routes-saved-item .btn,
.routes-export-row .btn,
.routes-library-tools > .btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.routes-field--compact {
  gap: 3px;
}

.routes-save-template {
  display: flex;
  flex-direction: column;
  gap: 5px;
  border-top: 1px solid #e5ebf4;
  padding-top: 8px;
}

.routes-save-template > span {
  color: #4d566a;
  font-size: 0.85rem;
  font-weight: 700;
}

.routes-save-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
}

.routes-saved-list {
  display: grid;
  gap: 6px;
  max-height: 126px;
  overflow: auto;
}

.routes-saved-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
}

.routes-saved-item .btn:first-child {
  min-width: 0;
  justify-content: flex-start;
  overflow: hidden;
  text-overflow: ellipsis;
}

.routes-freestyle-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #4d566a;
  font-size: 0.85rem;
  font-weight: 800;
}

.routes-export-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.routes-assistant-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid #e5ebf4;
  padding-top: 10px;
}

.routes-assistant-tools--map {
  border: 1px solid #dfe6f1;
  border-radius: 8px;
  background: #f8fbff;
  padding: 10px;
}

.routes-assistant-tools-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #4d566a;
  font-size: 0.85rem;
}

.routes-assistant-tools-head span {
  color: #6f7687;
  font-weight: 700;
}

.routes-assistant-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 8px;
}

.routes-assistant-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  border: 1px solid #dfe6f1;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px;
}

.routes-assistant-item i {
  color: #4d83d9;
}

.routes-assistant-item strong {
  display: block;
  color: #303746;
  font-size: 0.84rem;
  line-height: 1.15;
}

.routes-assistant-item p {
  margin: 2px 0 0;
  color: #6a7183;
  font-size: 0.78rem;
  line-height: 1.2;
}

.routes-hint {
  color: #6f7687;
}

.routes-diagnostics-list--notes {
  margin-top: 10px;
}

.routes-map-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.routes-map-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.routes-map-title {
  font-weight: 700;
  color: #344056;
}

.routes-map-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

.routes-map-generate-btn {
  min-width: 190px;
}

.routes-canvas-tools {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 8px 12px;
  align-items: center;
  border: 1px solid #dfe6f1;
  border-radius: 8px;
  background: #f8fbff;
  padding: 8px 10px;
}

.routes-canvas-tools-head {
  display: flex;
  flex-direction: column;
  gap: 1px;
  color: #4d566a;
  font-size: 0.84rem;
}

.routes-canvas-tools-head span {
  color: #6f7687;
  font-size: 0.76rem;
  font-weight: 700;
}

.routes-canvas-toolbar {
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}

.routes-canvas-tool-group {
  display: inline-flex;
  flex-wrap: nowrap;
  gap: 4px;
  border-right: 1px solid #d9e2ef;
  padding-right: 6px;
}

.routes-canvas-tool-group:last-child {
  border-right: 0;
  padding-right: 0;
}

.routes-canvas-tool-group .btn {
  width: 34px;
  min-height: 32px;
  padding-inline: 0;
}

.routes-map {
  width: 100%;
  height: 440px;
  border: 1px solid #d7deea;
  border-radius: 8px;
  overflow: hidden;
}

.routes-map-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.routes-layer-key {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #4f5668;
  font-size: 0.82rem;
  font-weight: 800;
}

.routes-layer-key > span {
  width: 28px;
  height: 0;
  border-top: 3px solid currentColor;
}

.routes-layer-key--sketch {
  color: #6f51ff;
}

.routes-layer-key--sketch > span {
  border-top-style: dashed;
}

.routes-layer-key--route {
  color: #fc4c02;
}

.routes-comparison {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.routes-comparison > div {
  min-width: 0;
  border: 1px solid #dfe6f1;
  border-radius: 8px;
  background: #f8fbff;
  padding: 8px 9px;
}

.routes-comparison span,
.routes-comparison small {
  display: block;
  color: #6f7687;
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
}

.routes-comparison strong {
  display: block;
  color: #303746;
  font-size: 0.95rem;
  line-height: 1.25;
  margin: 2px 0;
}

.routes-comparison small {
  text-transform: none;
}

.routes-visual-match {
  grid-column: 1 / -1;
}

.routes-visual-match strong {
  font-size: 1.02rem;
}

.routes-visual-match--strong {
  border-color: #bfe8cc !important;
  background: #f0fbf4 !important;
}

.routes-visual-match--strong strong {
  color: #1d7f42;
}

.routes-visual-match--mixed {
  border-color: #f0d9a6 !important;
  background: #fff8ec !important;
}

.routes-visual-match--mixed strong {
  color: #8f5f1f;
}

.routes-visual-match--weak {
  border-color: #efc3c3 !important;
  background: #fff5f5 !important;
}

.routes-visual-match--weak strong {
  color: #b23737;
}

.routes-comparison .routes-comparison-value--strong {
  color: #1d7f42;
}

.routes-comparison .routes-comparison-value--mixed {
  color: #8f5f1f;
}

.routes-comparison .routes-comparison-value--warn {
  color: #b23737;
}

.routes-results-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.routes-results-head h2 {
  margin: 0;
  font-size: 1.02rem;
}

.routes-results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 10px;
}

.route-card {
  text-align: left;
  border: 1px solid #d7deea;
  border-radius: 8px;
  background: #f8fbff;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  cursor: pointer;
}

.route-card:hover {
  border-color: #8db4ff;
}

.route-card--active {
  border-color: #0d6efd;
  box-shadow: 0 0 0 2px rgba(13, 110, 253, 0.12);
}

.route-card-head {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: flex-start;
}

.route-card-head > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.route-card-head strong {
  color: #242933;
  line-height: 1.2;
}

.route-card-head span {
  color: #687389;
  font-size: 0.78rem;
  font-weight: 700;
}

.route-score-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 2px;
}

.route-score-row {
  display: grid;
  grid-template-columns: 92px minmax(54px, 1fr) 42px;
  gap: 7px;
  align-items: center;
  color: #4f5668;
  font-size: 0.78rem;
  font-weight: 800;
}

.route-score-row--primary {
  grid-template-columns: 92px minmax(64px, 1fr) 48px;
  color: #303746;
  font-size: 0.86rem;
}

.route-score-row--primary strong {
  font-size: 1rem;
}

.route-score-row--primary .route-score-meter {
  height: 9px;
}

.route-score-row--secondary {
  color: #687389;
}

.route-score-meter {
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: #e5ebf4;
}

.route-score-meter span {
  display: block;
  height: 100%;
  min-width: 4px;
  border-radius: inherit;
  background: #8b95a7;
}

.route-score-row--strong .route-score-meter span {
  background: #2e9c57;
}

.route-score-row--ok .route-score-meter span {
  background: #4d83d9;
}

.route-score-row--mixed .route-score-meter span {
  background: #cf8b2d;
}

.route-score-row--warn .route-score-meter span {
  background: #de5b5b;
}

.route-quality-chip {
  flex: 0 0 auto;
  border: 1px solid #cfd8e6;
  border-radius: 999px;
  background: #ffffff;
  color: #4f5668;
  font-size: 0.74rem;
  font-weight: 800;
  line-height: 1.2;
  padding: 4px 7px;
}

.route-quality-chip--strong {
  border-color: #2e9c57;
  background: #ecf9f1;
  color: #1d7f42;
}

.route-quality-chip--ok {
  border-color: #cf8b2d;
  background: #fff8ec;
  color: #8f5f1f;
}

.route-quality-chip--warn {
  border-color: #de5b5b;
  background: #fff1f1;
  color: #b23737;
}

.route-card-metrics {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
}

.route-card-metrics div {
  border: 1px solid #dfe6f1;
  border-radius: 7px;
  background: #ffffff;
  padding: 6px 7px;
}

.route-card-metrics dt {
  color: #6f7687;
  font-size: 0.7rem;
  font-weight: 800;
  text-transform: uppercase;
}

.route-card-metrics dd {
  margin: 1px 0 0;
  color: #303746;
  font-size: 0.86rem;
  font-weight: 800;
}

.route-card-meta {
  margin: 0;
  color: #4f5668;
  font-size: 0.88rem;
}

.route-card-reasons {
  margin: 2px 0 0;
  padding-left: 16px;
  color: #596173;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.82rem;
}

.route-card-actions {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  margin-top: auto;
}

.route-card-actions .btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.routes-empty {
  margin: 0;
  color: #6a7183;
}

.routes-diagnostics-list {
  display: grid;
  gap: 8px;
  margin-top: 10px;
}

.routes-diagnostic-card {
  color: #596173;
  border: 1px solid #dce4ef;
  border-radius: 8px;
  background: #f8fbff;
  padding: 10px 12px;
  display: flex;
  gap: 10px;
  align-items: flex-start;
  font-size: 0.9rem;
}

.routes-diagnostic-card i {
  margin-top: 2px;
  color: #4d83d9;
}

.routes-diagnostic-card strong {
  display: block;
  color: #303746;
  margin-bottom: 2px;
}

.routes-diagnostic-card p {
  margin: 0;
}

.routes-diagnostic-card--warn {
  border-color: #f0d9a6;
  background: #fff8ea;
}

.routes-diagnostic-card--warn i {
  color: #b16d20;
}

.routes-diagnostic-card--error {
  border-color: #efbe84;
  background: #fff3e4;
}

.routes-diagnostic-card--error i {
  color: #b23737;
}

@media (max-width: 1100px) {
  .routes-title-block {
    flex-direction: column;
  }

  .routes-head-actions {
    justify-content: flex-start;
  }

  .routes-layout {
    grid-template-columns: 1fr;
  }

  .routes-map-actions {
    justify-content: flex-start;
  }

  .routes-canvas-tools {
    grid-template-columns: 1fr;
  }

  .routes-canvas-toolbar {
    justify-content: flex-start;
  }

  .routes-map {
    height: 400px;
  }

  .routes-comparison {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { useContextStore } from "@/stores/context";
import { useRoutesStore } from "@/stores/routes";
import { BUILT_IN_SHAPE_TEMPLATE_GROUPS, type BuiltInShapeTemplateKey } from "@/stores/routes";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import type { GeneratedRoute, RouteGenerationDiagnostic, RouteType } from "@/models/route-recommendation.model";
import { formatTime } from "@/utils/formatters";
import { formatDistance, formatElevation, formatSignedDistanceDelta, formatSignedPercent, scoreMeterStyle, distanceDeltaClass, polylineDistanceKm, artFitScore, artFitLabel, visualMatchSummary, visualMatchMessage, visualMatchClass, artFitClass, routeQualityScore, scoreBandClass, routeSourceLabel, routeProductBadges, routeProductSummary, highlightedRouteReasons, routeTitle, presentDiagnostic, nonBlockingGenerationDiagnosticCodes } from "@/services/route-presentation";
import { useRouteGeolocation } from "@/composables/useRouteGeolocation";

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
const routeEditGuideLayer = ref<L.Polyline>();
const routeEditMarkerLayers = ref<L.Marker[]>([]);
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
const { isLocating, locate, getStoredStartPoint, persistStartPoint } = useRouteGeolocation();

const selectedRoute = computed(() => routesStore.selectedRoute);
const routeEditMode = computed(() => routesStore.isRouteEditMode);
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
const builtInShapeTemplateGroups = BUILT_IN_SHAPE_TEMPLATE_GROUPS;
const builtInShapeTemplateLabels = new Map<BuiltInShapeTemplateKey, string>();
const builtInShapeTemplateIcons = new Map<BuiltInShapeTemplateKey, string>();
builtInShapeTemplateGroups.forEach((group) => {
  group.templates.forEach((template) => {
    builtInShapeTemplateLabels.set(template.key, template.label);
    builtInShapeTemplateIcons.set(template.key, template.icon);
  });
});
const selectedShapeTemplateLabel = computed(() => builtInShapeTemplateLabels.get(selectedShapeTemplate.value) ?? "Choose shape");
const selectedShapeTemplateIcon = computed(() => builtInShapeTemplateIcons.get(selectedShapeTemplate.value) ?? "fa-solid fa-shapes");
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
    const coverageMismatch = generationDiagnostics.value.some((diagnostic) => diagnostic.code === "OSRM_COVERAGE_MISMATCH");
    suggestions.push({
      id: "no-candidate",
      title: coverageMismatch ? "Check OSRM coverage" : "Recover generation",
      message: coverageMismatch ? "Use map data covering this area, or move the artwork into the covered region." : "Simplify the shape, then generate again.",
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
const workspaceStage = computed(() => {
  if (routesStore.shapePoints.length < 2) {
    return "Sketch";
  }
  if (!routesStore.startPoint) {
    return "Anchor";
  }
  if (!routesStore.hasRoutes) {
    return "Generate";
  }
  if (!selectedRoute.value) {
    return "Choose";
  }
  return "Export";
});
const canvasStatusLabel = computed(() => {
  const pointLabel = `${routesStore.shapePoints.length} point${routesStore.shapePoints.length === 1 ? "" : "s"}`;
  const routeLabel = routesStore.hasRoutes ? `${routesStore.routes.length} proposal${routesStore.routes.length === 1 ? "" : "s"}` : "no proposal";
  return `${pointLabel} · ${routeLabel}`;
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
      showToast("Invalid GPX: no usable route was found.", ToastTypeEnum.WARN);
      return;
    }
    redrawMapLayers({ fitBounds: true });
    const modeLabel = gpxImportMode.value === "append" ? "appended" : "imported";
    const fileLabel = importedFileCount > 1 ? "files" : "file";
    showToast(`GPX ${modeLabel} (${importedFileCount} ${fileLabel}, ${totalImportedPoints} points).`);
    if (invalidFileCount > 0) {
      showToast(`${invalidFileCount} file(s) ignored: invalid GPX format.`, ToastTypeEnum.WARN, 4200);
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
    map.value.stop();
    map.value.remove();
  }

  map.value = L.map(mapContainer.value, { zoomControl: true });
  map.value.setView([45.1885, 5.7245], 10);
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
  }).addTo(map.value);

  map.value.on("click", (event: L.LeafletMouseEvent) => {
    if (routesStore.isRouteEditMode) {
      return;
    }
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
  routesStore.routeEditControlPoints.forEach((point) => {
    if (point.length >= 2) {
      points.push(L.latLng(point[0], point[1]));
    }
  });
  return points;
}

function clearRouteEditLayers() {
  if (routeEditGuideLayer.value) {
    routeEditGuideLayer.value.remove();
    routeEditGuideLayer.value = undefined;
  }
  routeEditMarkerLayers.value.forEach((marker) => marker.remove());
  routeEditMarkerLayers.value = [];
}

function routeEditMarkerIcon(index: number, isEndpoint: boolean): L.DivIcon {
  return L.divIcon({
    className: `routes-edit-marker${isEndpoint ? " routes-edit-marker--endpoint" : ""}`,
    html: `<span>${index + 1}</span>`,
    iconSize: [26, 26],
    iconAnchor: [13, 13],
  });
}

function distanceToSegment(point: L.Point, start: L.Point, end: L.Point): number {
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  if (dx === 0 && dy === 0) {
    return point.distanceTo(start);
  }
  const ratio = Math.max(0, Math.min(1, ((point.x - start.x) * dx + (point.y - start.y) * dy) / (dx * dx + dy * dy)));
  return point.distanceTo(L.point(start.x + ratio * dx, start.y + ratio * dy));
}

function nearestRouteEditSegmentIndex(latlng: L.LatLng): number | undefined {
  if (!map.value || routesStore.routeEditControlPoints.length < 2) {
    return undefined;
  }
  const clickPoint = map.value.latLngToLayerPoint(latlng);
  let bestIndex = 0;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (let index = 0; index < routesStore.routeEditControlPoints.length - 1; index += 1) {
    const start = routesStore.routeEditControlPoints[index];
    const end = routesStore.routeEditControlPoints[index + 1];
    const segmentDistance = distanceToSegment(
      clickPoint,
      map.value.latLngToLayerPoint(L.latLng(start[0], start[1])),
      map.value.latLngToLayerPoint(L.latLng(end[0], end[1])),
    );
    if (segmentDistance < bestDistance) {
      bestDistance = segmentDistance;
      bestIndex = index;
    }
  }
  return bestIndex;
}

async function moveRouteEditControlPoint(index: number, latlng: L.LatLng) {
  try {
    await routesStore.moveRouteEditControlPoint(index, latlng.lat, latlng.lng);
    redrawMapLayers({ fitBounds: false });
  } catch (error) {
    showToast("Unable to reroute the edited segment", ToastTypeEnum.ERROR, 4200);
    console.error(error);
    redrawMapLayers({ fitBounds: false });
  }
}

async function insertRouteEditControlPoint(latlng: L.LatLng, afterIndex?: number) {
  try {
    await routesStore.insertRouteEditControlPoint(latlng.lat, latlng.lng, afterIndex);
    redrawMapLayers({ fitBounds: false });
  } catch (error) {
    showToast("Unable to insert this control point", ToastTypeEnum.ERROR, 4200);
    console.error(error);
    redrawMapLayers({ fitBounds: false });
  }
}

async function removeRouteEditControlPoint(index: number) {
  try {
    await routesStore.removeRouteEditControlPoint(index);
    redrawMapLayers({ fitBounds: false });
  } catch (error) {
    showToast("Unable to remove this control point", ToastTypeEnum.ERROR, 4200);
    console.error(error);
    redrawMapLayers({ fitBounds: false });
  }
}

function renderRouteEditLayers() {
  if (!map.value || !routesStore.isRouteEditMode || routesStore.routeEditControlPoints.length < 2) {
    return;
  }
  const controlLatLngs = routesStore.routeEditControlPoints.map((point) => L.latLng(point[0], point[1]));
  routeEditGuideLayer.value = L.polyline(controlLatLngs, {
    color: "#00a8a8",
    weight: 3,
    dashArray: "3 8",
    opacity: 0.9,
  }).addTo(map.value);
  routeEditGuideLayer.value.bindTooltip("Edit controls", { direction: "top" });

  routesStore.routeEditControlPoints.forEach((point, index) => {
    const isEndpoint = index === 0 || index === routesStore.routeEditControlPoints.length - 1;
    const marker = L.marker([point[0], point[1]], {
      draggable: true,
      icon: routeEditMarkerIcon(index, isEndpoint),
      keyboard: true,
      title: `Control point ${index + 1}`,
    }).addTo(map.value as L.Map);
    marker.on("dragend", () => {
      void moveRouteEditControlPoint(index, marker.getLatLng());
    });
    marker.on("contextmenu", (event: L.LeafletMouseEvent) => {
      event.originalEvent.preventDefault();
      event.originalEvent.stopPropagation();
      void removeRouteEditControlPoint(index);
    });
    marker.on("dblclick", (event: L.LeafletMouseEvent) => {
      event.originalEvent.preventDefault();
      event.originalEvent.stopPropagation();
      void removeRouteEditControlPoint(index);
    });
    routeEditMarkerLayers.value.push(marker);
  });
}

function redrawMapLayers(options: { fitBounds?: boolean } = {}) {
  if (!map.value) {
    return;
  }

  clearRouteEditLayers();
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
      if (routesStore.isRouteEditMode) {
        selectedRouteLayer.value.on("click", (event: L.LeafletMouseEvent) => {
          event.originalEvent.preventDefault();
          event.originalEvent.stopPropagation();
          void insertRouteEditControlPoint(event.latlng, nearestRouteEditSegmentIndex(event.latlng));
        });
      }
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

  renderRouteEditLayers();

  const allPoints = collectAllMapPoints();
  if (options.fitBounds !== false && allPoints.length > 0) {
    const bounds = L.latLngBounds(allPoints);
    if (bounds.isValid()) {
      map.value.fitBounds(bounds, { padding: [26, 26], animate: false });
    }
  }
}

async function requestMyLocation(silent = false) {
  try {
    const position = await locate();
    applyStartPoint(position.lat, position.lng, 12);
    persistStartPoint(position.lat, position.lng);
    if (!silent) showToast("Start point set from your current location");
  } catch (error) {
    const fallback = getStoredStartPoint();
    if (fallback) {
      applyStartPoint(fallback.lat, fallback.lng, 11);
      if (!silent) showToast("Unable to access live location, using your last known start point", ToastTypeEnum.WARN, 4200);
      return;
    }
    if (map.value) {
      const center = map.value.getCenter();
      applyStartPoint(center.lat, center.lng, map.value.getZoom());
      persistStartPoint(center.lat, center.lng);
    }
    if (!silent) {
      const reason = error instanceof Error ? error.message : "unknown error";
      showToast(`Unable to access your location (${reason}). Using current map center as start point.`, ToastTypeEnum.WARN, 4600);
    }
  }
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
  selectedShapeTemplate.value = template;
  const loaded = routesStore.applyBuiltInShapeTemplate(template, currentTemplateCenter());
  if (loaded) {
    redrawMapLayers({ fitBounds: true });
    showToast(`${builtInShapeTemplateLabels.get(template) ?? template} sketch loaded`);
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
  context.fillText(saveShapeName.value.trim() || "GPS Art sketch", 30, canvas.height - 28);

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

function beginRouteEdit(route: GeneratedRoute) {
  routesStore.setSelectedRoute(route.routeId);
  if (!routesStore.beginRouteEdit()) {
    showToast("This proposal cannot be edited on OSRM roads", ToastTypeEnum.ERROR, 4200);
    return;
  }
  redrawMapLayers({ fitBounds: false });
}

function stopRouteEdit() {
  routesStore.stopRouteEdit();
  redrawMapLayers({ fitBounds: false });
}

async function resetRouteEdit() {
  if (!routesStore.resetRouteEditControls()) {
    return;
  }
  try {
    await routesStore.applyRouteEdit();
    redrawMapLayers({ fitBounds: false });
  } catch (error) {
    showToast("Unable to reset the edited route", ToastTypeEnum.ERROR, 4200);
    console.error(error);
  }
}

async function undoRouteEdit() {
  try {
    await routesStore.undoRouteEdit();
    redrawMapLayers({ fitBounds: false });
  } catch (error) {
    showToast("Unable to undo this edit", ToastTypeEnum.ERROR, 4200);
    console.error(error);
  }
}

async function redoRouteEdit() {
  try {
    await routesStore.redoRouteEdit();
    redrawMapLayers({ fitBounds: false });
  } catch (error) {
    showToast("Unable to redo this edit", ToastTypeEnum.ERROR, 4200);
    console.error(error);
  }
}

watch(
  () => [routesStore.startPoint, routesStore.shapePoints, selectedRoute.value?.routeId, routesStore.isRouteEditMode, routesStore.routeEditControlPoints],
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
    map.value.stop();
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
          <h1>GPS Art</h1>
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
      <div class="routes-art-steps" aria-label="GPS Art workflow">
        <span :class="{ 'routes-art-step--active': workspaceStage === 'Sketch' }">
          <i class="fa-solid fa-pencil" aria-hidden="true" />
          Sketch
        </span>
        <span :class="{ 'routes-art-step--active': workspaceStage === 'Anchor' }">
          <i class="fa-solid fa-location-crosshairs" aria-hidden="true" />
          Anchor
        </span>
        <span :class="{ 'routes-art-step--active': workspaceStage === 'Generate' }">
          <i class="fa-solid fa-magnet" aria-hidden="true" />
          Generate
        </span>
        <span :class="{ 'routes-art-step--active': workspaceStage === 'Choose' || workspaceStage === 'Export' }">
          <i class="fa-solid fa-file-export" aria-hidden="true" />
          Export
        </span>
      </div>
    </header>

    <section class="routes-workspace">
      <aside class="routes-panel routes-controls">
        <div class="routes-sidebar-head">
          <strong>Source</strong>
          <span>{{ workspaceStage }}</span>
        </div>

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

        <div class="routes-shape-tools">
          <div class="routes-shape-tools-head">
            <strong>Artwork sketch</strong>
            <span>{{ routesStore.shapePoints.length }} point(s)</span>
          </div>
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
            {{ routesStore.isDrawingShape ? "Drawing is active" : "Drawing paused" }}
          </small>
        </div>

        <details
          class="routes-library-tools"
        >
          <summary>
            <span class="routes-library-combo-main">
              <span class="routes-library-combo-icon">
                <i :class="selectedShapeTemplateIcon" aria-hidden="true" />
              </span>
              <span>
                <strong>Templates and imports</strong>
                <small>{{ selectedShapeTemplateLabel }} - {{ routesStore.savedShapeTemplateCount }} saved</small>
              </span>
            </span>
            <i class="fa-solid fa-chevron-down routes-library-combo-arrow" aria-hidden="true" />
          </summary>
          <div class="routes-library-combo-panel">
            <div class="routes-template-panel">
              <div
                v-for="group in builtInShapeTemplateGroups"
                :key="group.id"
                class="routes-template-group"
              >
                <span class="routes-template-group-title">{{ group.label }}</span>
                <div class="routes-template-grid">
                  <button
                    v-for="template in group.templates"
                    :key="template.key"
                    type="button"
                    class="routes-template-button"
                    :class="{ 'routes-template-button--active': selectedShapeTemplate === template.key }"
                    :aria-pressed="selectedShapeTemplate === template.key"
                    @click="applyShapeTemplate(template.key)"
                  >
                    <i :class="template.icon" aria-hidden="true" />
                    <span>{{ template.label }}</span>
                  </button>
                </div>
              </div>
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
        </details>

      </aside>

      <div class="routes-panel routes-map-panel">
        <div class="routes-canvas-topbar">
          <div>
            <span class="routes-map-title">Art canvas</span>
            <span class="routes-canvas-status">{{ canvasStatusLabel }}</span>
          </div>
          <div class="routes-map-actions">
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              @click="routesStore.toggleShapeDrawing"
            >
              <i class="fa-solid fa-pen-nib" aria-hidden="true" />
              {{ routesStore.isDrawingShape ? "Stop drawing" : "Draw" }}
            </button>
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="isLocating"
              @click="useMyLocation"
            >
              <i class="fa-solid fa-location-crosshairs" aria-hidden="true" />
              {{ isLocating ? "Locating..." : "Use my location" }}
            </button>
            <button
              type="button"
              class="btn btn-primary btn-sm routes-map-generate-btn"
              :disabled="routesStore.isLoading || !canGenerate"
              @click="generateRoutes"
            >
              <i class="fa-solid fa-rotate" aria-hidden="true" />
              {{ generateRouteButtonLabel }}
            </button>
          </div>
        </div>
        <div class="routes-map-shell">
          <div
            ref="mapContainer"
            class="routes-map"
          />
          <div class="routes-canvas-tools">
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
          <span
            v-if="routeEditMode"
            class="routes-layer-key routes-layer-key--edit"
          >
            <span aria-hidden="true" />
            Edit controls
          </span>
        </div>
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

      <aside class="routes-panel routes-results routes-decision-panel">
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
              <span>Rideability</span>
              <div class="route-score-meter" aria-hidden="true">
                <span :style="scoreMeterStyle(routeQualityScore(route))" />
              </div>
              <strong>{{ routeQualityScore(route) }}%</strong>
            </div>
          </div>

          <div
            v-if="routeProductBadges(route).length > 0"
            class="route-card-badges"
          >
            <span
              v-for="badge in routeProductBadges(route)"
              :key="badge.id"
              class="route-card-badge"
              :class="`route-card-badge--${badge.tone}`"
            >
              <i :class="badge.icon" aria-hidden="true" />
              {{ badge.label }}
            </span>
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

          <p class="route-card-meta">{{ routeProductSummary(route) }}</p>

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
              v-if="routeEditMode && selectedRoute?.routeId === route.routeId"
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="routesStore.isRouteEditLoading"
              @click.stop="stopRouteEdit"
            >
              <i class="fa-solid fa-check" aria-hidden="true" />
              Done
            </button>
            <button
              v-else
              type="button"
              class="btn btn-outline-primary btn-sm"
              :disabled="!route.isRoadGraphGenerated || routesStore.isRouteEditLoading"
              @click.stop="beginRouteEdit(route)"
            >
              <i class="fa-solid fa-magnet" aria-hidden="true" />
              Edit
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
          <div
            v-if="routeEditMode && selectedRoute?.routeId === route.routeId"
            class="route-edit-actions"
          >
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              :disabled="!routesStore.canUndoRouteEdit || routesStore.isRouteEditLoading"
              title="Undo route edit"
              @click.stop="undoRouteEdit"
            >
              <i class="fa-solid fa-rotate-left" aria-hidden="true" />
            </button>
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              :disabled="!routesStore.canRedoRouteEdit || routesStore.isRouteEditLoading"
              title="Redo route edit"
              @click.stop="redoRouteEdit"
            >
              <i class="fa-solid fa-rotate-right" aria-hidden="true" />
            </button>
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              :disabled="routesStore.isRouteEditLoading"
              title="Reset to generated route"
              @click.stop="resetRouteEdit"
            >
              <i class="fa-solid fa-arrow-rotate-left" aria-hidden="true" />
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
      </aside>
    </section>
  </section>
</template>

<style scoped src="../assets/views/routes-view.css"></style>

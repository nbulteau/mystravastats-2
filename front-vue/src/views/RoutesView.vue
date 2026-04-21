<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { useContextStore } from "@/stores/context";
import { useRoutesStore } from "@/stores/routes";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import type { RouteType } from "@/models/route-recommendation.model";
import { formatTime } from "@/utils/formatters";

const contextStore = useContextStore();
const routesStore = useRoutesStore();
const uiStore = useUiStore();
onMounted(() => contextStore.updateCurrentView("routes"));

const mapContainer = ref<HTMLDivElement | null>(null);
const map = ref<L.Map>();
const startMarker = ref<L.CircleMarker>();
const shapePolylineLayer = ref<L.Polyline>();
const customWaypointDraftLayer = ref<L.Polyline>();
const customWaypointMarkers = ref<L.CircleMarker[]>([]);
const selectedRouteLayer = ref<L.Polyline>();
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
const canGenerate = computed(() =>
  routesStore.mode === "TARGET" ? routesStore.canGenerateTarget : routesStore.canGenerateShape,
);
const isShapeMode = computed(() => routesStore.mode === "SHAPE");
const isTargetMode = computed(() => routesStore.mode === "TARGET");
const isTargetCustomMode = computed(
  () => routesStore.mode === "TARGET" && routesStore.targetGenerationMode === "CUSTOM",
);
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
    return "Generating...";
  }
  if (routesStore.mode === "TARGET" && routesStore.lastGeneratedTargetRouteNumber > 0) {
    return `Generate route (last: #${routesStore.lastGeneratedTargetRouteNumber})`;
  }
  return "Generate route";
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

const directionOptions = [
  { value: "UNDEFINED", label: "Undefined" },
  { value: "N", label: "Nord" },
  { value: "S", label: "Sud" },
  { value: "E", label: "Est" },
  { value: "W", label: "Ouest" },
];
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

function formatDistance(value: number): string {
  return `${value.toFixed(1)} km`;
}

function formatElevation(value: number): string {
  return `${Math.round(value)} m`;
}

function modeButtonClass(mode: "TARGET" | "SHAPE"): string {
  return routesStore.mode === mode ? "btn btn-primary" : "btn btn-outline-secondary";
}

function targetModeButtonClass(mode: "AUTOMATIC" | "CUSTOM"): string {
  return routesStore.targetGenerationMode === mode ? "btn btn-primary btn-sm" : "btn btn-outline-secondary btn-sm";
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
    if (routesStore.mode === "TARGET" && routesStore.targetGenerationMode === "CUSTOM" && routesStore.startPoint) {
      routesStore.addCustomWaypoint(event.latlng.lat, event.latlng.lng);
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
  routesStore.customWaypoints.forEach((point) => {
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
  if (customWaypointDraftLayer.value) {
    customWaypointDraftLayer.value.remove();
    customWaypointDraftLayer.value = undefined;
  }
  if (customWaypointMarkers.value.length > 0) {
    customWaypointMarkers.value.forEach((marker) => marker.remove());
    customWaypointMarkers.value = [];
  }
  if (selectedRouteLayer.value) {
    selectedRouteLayer.value.remove();
    selectedRouteLayer.value = undefined;
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

  if (routesStore.shapePoints.length >= 2) {
    const shapeLatLngs = routesStore.shapePoints.map((point) => L.latLng(point[0], point[1]));
    shapePolylineLayer.value = L.polyline(shapeLatLngs, {
      color: "#7b61ff",
      weight: 3,
      dashArray: "8 8",
      opacity: 0.9,
    }).addTo(map.value);
  }

  if (routesStore.customWaypoints.length > 0) {
    const points = routesStore.customWaypoints
      .filter((point) => point.length >= 2)
      .map((point) => L.latLng(point[0], point[1]));

    points.forEach((point, index) => {
      const marker = L.circleMarker(point, {
        radius: 5,
        color: "#198754",
        weight: 2,
        fillColor: "#75d39a",
        fillOpacity: 0.9,
      }).addTo(map.value!);
      marker.bindTooltip(`Waypoint ${index + 1}`, { direction: "top" });
      customWaypointMarkers.value.push(marker);
    });

    if (routesStore.startPoint) {
      const draftLatLngs = [L.latLng(routesStore.startPoint.lat, routesStore.startPoint.lng), ...points];
      customWaypointDraftLayer.value = L.polyline(draftLatLngs, {
        color: "#198754",
        weight: 3,
        dashArray: "6 8",
        opacity: 0.9,
      }).addTo(map.value);
    }
  }

  if (selectedRoute.value && selectedRoute.value.previewLatLng.length >= 2) {
    const routeLatLngs = selectedRoute.value.previewLatLng
      .filter((point) => point.length >= 2)
      .map((point) => L.latLng(point[0], point[1]));
    if (routeLatLngs.length >= 2) {
      selectedRouteLayer.value = L.polyline(routeLatLngs, {
        color: "#fc4c02",
        weight: 4,
        opacity: 0.95,
      }).addTo(map.value);
    }
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

function switchMode(mode: "TARGET" | "SHAPE") {
  routesStore.setMode(mode);
  redrawMapLayers({ fitBounds: false });
}

function switchTargetGenerationMode(mode: "AUTOMATIC" | "CUSTOM") {
  routesStore.setTargetGenerationMode(mode);
  redrawMapLayers({ fitBounds: false });
}

function clearCustomWaypoints() {
  routesStore.clearCustomWaypoints();
  redrawMapLayers({ fitBounds: false });
}

function undoCustomWaypoint() {
  routesStore.removeLastCustomWaypoint();
  redrawMapLayers({ fitBounds: false });
}

function resetStartPoint() {
  routesStore.clearStartPoint();
  routesStore.clearCustomWaypoints();
  redrawMapLayers({ fitBounds: false });
  showToast("Start point cleared. Click the map or use your location to set a new start point.");
}

async function generateRoutes() {
  const previousCount = routesStore.routes.length;
  try {
    await routesStore.generateRoutes();
    redrawMapLayers();
    if (!routesStore.hasRoutes) {
      const message = failureSummaryDiagnostic.value?.message ?? routesStore.generationDiagnostics[0]?.message;
      const displayMessage = message
        ? `No route generated. ${message}`
        : "No route generated with current constraints. Try widening your targets.";
      showToast(displayMessage, ToastTypeEnum.ERROR, 5000);
      return;
    }
    if (routesStore.mode === "TARGET" && routesStore.routes.length === previousCount) {
      const firstDiagnostic = routesStore.generationDiagnostics[0]?.message;
      showToast(firstDiagnostic ?? "No additional unique route found with current constraints.", ToastTypeEnum.WARN, 4200);
      return;
    }
    if (routesStore.mode === "TARGET" && routesStore.lastGeneratedTargetRouteNumber > 0) {
      showToast(`Route #${routesStore.lastGeneratedTargetRouteNumber} generated and shown on the map.`, ToastTypeEnum.NORMAL, 2600);
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

async function exportSelectedRoute() {
  if (!selectedRoute.value) {
    return;
  }
  isExporting.value = true;
  try {
    await routesStore.exportRouteGpx(selectedRoute.value.routeId);
    showToast("GPX exported successfully");
  } catch (error) {
    showToast("Unable to export GPX for this route", ToastTypeEnum.ERROR, 4200);
    console.error(error);
  } finally {
    isExporting.value = false;
  }
}

watch(
  () => [routesStore.startPoint, routesStore.shapePoints, routesStore.customWaypoints, selectedRoute.value?.routeId],
  () => redrawMapLayers({ fitBounds: false }),
  { deep: true },
);

onMounted(async () => {
  await nextTick();
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
      <div class="routes-mode-switch">
        <button
          type="button"
          :class="modeButtonClass('TARGET')"
          @click="switchMode('TARGET')"
        >
          Target loop generator
        </button>
        <button
          type="button"
          :class="modeButtonClass('SHAPE')"
          @click="switchMode('SHAPE')"
        >
          Shape based generator
        </button>
        <span :class="routingEngineClass">
          <span class="routes-engine-dot" />
          {{ routingEngineLabel }}
        </span>
      </div>
      <p class="routes-head-caption">
        Same map for start point, shape input and generated route preview.
      </p>
    </header>

    <section class="routes-panel routes-layout">
      <aside class="routes-controls">
        <button
          type="button"
          class="btn btn-outline-primary routes-location-btn"
          :disabled="isLocating"
          @click="useMyLocation"
        >
          {{ isLocating ? "Locating..." : "Use my location" }}
        </button>

        <label class="routes-field">
          <span>Route type</span>
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

        <div
          v-if="isTargetMode"
          class="routes-target-mode-switch"
        >
          <span>Target mode</span>
          <div class="routes-target-mode-buttons">
            <button
              type="button"
              :class="targetModeButtonClass('AUTOMATIC')"
              @click="switchTargetGenerationMode('AUTOMATIC')"
            >
              Automatic
            </button>
            <button
              type="button"
              :class="targetModeButtonClass('CUSTOM')"
              @click="switchTargetGenerationMode('CUSTOM')"
            >
              Custom
            </button>
          </div>
        </div>

        <label
          v-if="isTargetMode && !isTargetCustomMode"
          class="routes-field"
        >
          <span>Direction</span>
          <select
            v-model="routesStore.startDirection"
            class="form-select"
          >
            <option
              v-for="option in directionOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </option>
          </select>
        </label>

        <div
          v-if="isTargetCustomMode"
          class="routes-custom-tools"
        >
          <div class="routes-custom-tools-head">
            <strong>Custom waypoints</strong>
            <span>{{ routesStore.customWaypoints.length }} point(s)</span>
          </div>
          <small class="routes-hint">
            Click on the map to add passage points after the start point.
          </small>
          <div class="routes-custom-tools-buttons">
            <button
              type="button"
              class="btn btn-outline-secondary btn-sm"
              :disabled="routesStore.customWaypoints.length === 0"
              @click="undoCustomWaypoint"
            >
              Undo last point
            </button>
            <button
              type="button"
              class="btn btn-outline-danger btn-sm"
              :disabled="routesStore.customWaypoints.length === 0"
              @click="clearCustomWaypoints"
            >
              Clear points
            </button>
          </div>
          <button
            type="button"
            class="btn btn-outline-secondary btn-sm"
            @click="resetStartPoint"
          >
            Reset start point
          </button>
        </div>

        <label class="routes-field">
          <span>Distance target (km)</span>
          <input
            v-model="routesStore.distanceTargetKm"
            type="number"
            min="1"
            step="0.5"
            class="form-control"
          >
        </label>

        <label class="routes-field">
          <span>Elevation target (m)</span>
          <input
            v-model="routesStore.elevationTargetM"
            type="number"
            min="0"
            step="10"
            class="form-control"
          >
        </label>

        <div
          v-if="isShapeMode"
          class="routes-shape-tools"
        >
          <button
            type="button"
            class="btn btn-outline-secondary"
            @click="routesStore.toggleShapeDrawing"
          >
            {{ routesStore.isDrawingShape ? "Stop drawing" : "Draw shape" }}
          </button>
          <button
            type="button"
            class="btn btn-outline-danger"
            :disabled="routesStore.shapePoints.length === 0"
            @click="routesStore.clearShape"
          >
            Clear shape
          </button>
          <small class="routes-hint">
            Click on the map while drawing to add points.
          </small>
        </div>

        <button
          type="button"
          class="btn btn-primary routes-generate-btn"
          :disabled="routesStore.isLoading || !canGenerate"
          @click="generateRoutes"
        >
          {{ generateRouteButtonLabel }}
        </button>
      </aside>

      <div class="routes-map-panel">
        <div class="routes-map-head">
          <span class="routes-map-title">Generated route map</span>
          <button
            type="button"
            class="btn btn-outline-primary btn-sm"
            :disabled="!selectedRoute || isExporting"
            @click="exportSelectedRoute"
          >
            {{ isExporting ? "Exporting..." : "Export GPX" }}
          </button>
        </div>
        <div
          ref="mapContainer"
          class="routes-map"
        />
      </div>
    </section>

    <section class="routes-panel routes-results">
      <header class="routes-results-head">
        <h2>Generated routes</h2>
        <span>{{ routesStore.routes.length }} route(s)</span>
      </header>
      <p
        v-if="!routesStore.hasRoutes"
        class="routes-empty"
      >
        {{ generationDiagnostics.length > 0
          ? "No route matched all constraints for this request."
          : "Generate a route to see proposals here." }}
      </p>
      <p
        v-if="!routesStore.hasRoutes && failureSummaryDiagnostic"
        class="routes-failure-summary"
      >
        {{ failureSummaryDiagnostic.message }}
      </p>
      <ul
        v-if="!routesStore.hasRoutes && detailedGenerationDiagnostics.length > 0"
        class="routes-diagnostics"
      >
        <li
          v-for="diagnostic in detailedGenerationDiagnostics"
          :key="diagnostic.code"
        >
          <strong>{{ diagnostic.code }}</strong>: {{ diagnostic.message }}
        </li>
      </ul>

      <div
        v-else
        class="routes-results-grid"
      >
        <article
          v-for="route in routesStore.routes"
          :key="route.routeId"
          role="button"
          tabindex="0"
          class="route-card"
          :class="{ 'route-card--active': selectedRoute?.routeId === route.routeId }"
          @click="pickRoute(route.routeId)"
          @keydown.enter.space.prevent="pickRoute(route.routeId)"
        >
          <div class="route-card-head">
            <strong>{{ route.title }}</strong>
            <span class="route-score">{{ route.score.global.toFixed(1) }}%</span>
          </div>
          <p class="route-card-meta">
            {{ formatDistance(route.distanceKm) }} • {{ formatElevation(route.elevationGainM) }} • {{ formatTime(route.durationSec) }}
          </p>
          <p class="route-card-meta">
            {{ route.variantType.replaceAll('_', ' ') }}
            <span v-if="route.isRoadGraphGenerated">• road-graph</span>
          </p>
        </article>
      </div>
      <ul
        v-if="routesStore.hasRoutes && detailedGenerationDiagnostics.length > 0"
        class="routes-diagnostics routes-diagnostics--notes"
      >
        <li
          v-for="diagnostic in detailedGenerationDiagnostics"
          :key="diagnostic.code"
        >
          <strong>{{ diagnostic.code }}</strong>: {{ diagnostic.message }}
        </li>
      </ul>
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
  border-radius: 16px;
  padding: 14px;
  box-shadow: 0 6px 20px rgba(12, 21, 38, 0.05);
}

.routes-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.routes-mode-switch {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.routes-head-caption {
  margin: 0;
  color: #5e6578;
  font-size: 0.92rem;
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

.routes-target-mode-switch {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.routes-target-mode-switch > span {
  font-size: 0.85rem;
  font-weight: 700;
  color: #4d566a;
}

.routes-target-mode-buttons {
  display: flex;
  gap: 8px;
}

.routes-custom-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid #d9e2ef;
  border-radius: 10px;
  background: #f8fbff;
}

.routes-custom-tools-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.85rem;
  color: #4d566a;
}

.routes-custom-tools-buttons {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.routes-shape-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.routes-hint {
  color: #6f7687;
}

.routes-diagnostics--notes {
  margin-top: 10px;
  border: 1px solid #f0d9a6;
  border-radius: 10px;
  background: #fff8ea;
}

.routes-generate-btn {
  margin-top: 4px;
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
}

.routes-map-title {
  font-weight: 700;
  color: #344056;
}

.routes-map {
  width: 100%;
  height: 440px;
  border: 1px solid #d7deea;
  border-radius: 14px;
  overflow: hidden;
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
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px;
}

.route-card {
  text-align: left;
  border: 1px solid #d7deea;
  border-radius: 12px;
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
}

.route-score {
  color: #0d6efd;
  font-weight: 700;
}

.route-card-meta {
  margin: 0;
  color: #4f5668;
  font-size: 0.88rem;
}

.routes-empty {
  margin: 0;
  color: #6a7183;
}

.routes-failure-summary {
  margin: 8px 0 0;
  padding: 10px 12px;
  border: 1px solid #efbe84;
  border-radius: 10px;
  background: #fff3e4;
  color: #7a4d1f;
  font-size: 0.9rem;
}

.routes-diagnostics {
  margin: 8px 0 0;
  padding-left: 18px;
  color: #596173;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.9rem;
}

@media (max-width: 1100px) {
  .routes-layout {
    grid-template-columns: 1fr;
  }

  .routes-map {
    height: 400px;
  }
}
</style>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { useContextStore } from "@/stores/context";
import { useRoutesStore } from "@/stores/routes";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import { formatTime } from "@/utils/formatters";

const contextStore = useContextStore();
const routesStore = useRoutesStore();
const uiStore = useUiStore();
contextStore.updateCurrentView("routes");

const mapContainer = ref<HTMLDivElement | null>(null);
const map = ref<L.Map>();
const startMarker = ref<L.CircleMarker>();
const shapePolylineLayer = ref<L.Polyline>();
const selectedRouteLayer = ref<L.Polyline>();
const isExporting = ref(false);

const selectedRoute = computed(() => routesStore.selectedRoute);
const canGenerate = computed(() =>
  routesStore.mode === "TARGET" ? routesStore.canGenerateTarget : routesStore.canGenerateShape,
);
const isShapeMode = computed(() => routesStore.mode === "SHAPE");

const routeTypeOptions = [
  { value: "RIDE", label: "Ride" },
  { value: "MTB", label: "VTT" },
  { value: "GRAVEL", label: "Gravel" },
  { value: "RUN", label: "Course à pied" },
  { value: "TRAIL", label: "Trail" },
  { value: "HIKE", label: "Randonnée" },
];

const directionOptions = [
  { value: "N", label: "Nord" },
  { value: "S", label: "Sud" },
  { value: "E", label: "Est" },
  { value: "W", label: "Ouest" },
];

function formatDistance(value: number): string {
  return `${value.toFixed(1)} km`;
}

function formatElevation(value: number): string {
  return `${Math.round(value)} m`;
}

function modeButtonClass(mode: "TARGET" | "SHAPE"): string {
  return routesStore.mode === mode ? "btn btn-primary" : "btn btn-outline-secondary";
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
    redrawMapLayers({ fitBounds: false });
  });
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

async function useMyLocation() {
  if (!navigator.geolocation) {
    showToast("Geolocation is not available in this browser", ToastTypeEnum.ERROR, 3800);
    return;
  }
  navigator.geolocation.getCurrentPosition(
    (position) => {
      const lat = position.coords.latitude;
      const lng = position.coords.longitude;
      routesStore.setStartPoint(lat, lng);
      if (map.value) {
        map.value.setView([lat, lng], 12);
      }
      redrawMapLayers({ fitBounds: false });
      showToast("Start point set from your current location");
    },
    () => {
      showToast("Unable to access your location (permission denied or unavailable)", ToastTypeEnum.ERROR, 4200);
    },
    {
      enableHighAccuracy: true,
      timeout: 10000,
    },
  );
}

function switchMode(mode: "TARGET" | "SHAPE") {
  routesStore.setMode(mode);
  redrawMapLayers({ fitBounds: false });
}

async function generateRoutes() {
  try {
    await routesStore.generateRoutes();
    redrawMapLayers();
    if (!routesStore.hasRoutes) {
      showToast("No route generated with current constraints. Try widening your targets.", ToastTypeEnum.ERROR, 4200);
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
  () => [routesStore.startPoint, routesStore.shapePoints, selectedRoute.value?.routeId],
  () => redrawMapLayers({ fitBounds: false }),
  { deep: true },
);

onMounted(async () => {
  await nextTick();
  initMap();
  redrawMapLayers({ fitBounds: false });
  useMyLocation();
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
          @click="useMyLocation"
        >
          Use my location
        </button>

        <label class="routes-field">
          <span>Type of route</span>
          <select
            v-model="routesStore.routeType"
            class="form-select"
          >
            <option
              v-for="option in routeTypeOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </option>
          </select>
        </label>

        <label
          v-if="!isShapeMode"
          class="routes-field"
        >
          <span>Departure direction</span>
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

        <label class="routes-field">
          <span>Variants</span>
          <input
            v-model.number="routesStore.variantCount"
            type="number"
            min="1"
            max="24"
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
          {{ routesStore.isLoading ? "Generating..." : "Generate route" }}
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
        Generate a route to see proposals here.
      </p>

      <div
        v-else
        class="routes-results-grid"
      >
        <button
          v-for="route in routesStore.routes"
          :key="route.routeId"
          type="button"
          class="route-card"
          :class="{ 'route-card--active': selectedRoute?.routeId === route.routeId }"
          @click="pickRoute(route.routeId)"
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
        </button>
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

.routes-shape-tools {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.routes-hint {
  color: #6f7687;
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

@media (max-width: 1100px) {
  .routes-layout {
    grid-template-columns: 1fr;
  }

  .routes-map {
    height: 400px;
  }
}
</style>

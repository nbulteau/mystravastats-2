<script setup lang="ts">
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from "vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { formatTime } from "@/utils/formatters";
import {
  CLIMB_MAP_ALL_FILTER,
  buildClimbMapSummits,
  clusterClimbMapSummits,
  filterClimbMapSummits,
  type ClimbMapFilters,
  type ClimbMapStatusFilter,
  type ClimbMapSummit,
} from "@/utils/climb-map";

const props = withDefaults(defineProps<{
  climbs: BadgeCheckResult[];
  yearLabel: string;
  yearOptions: string[];
  category: string;
  focusSummitId?: string | null;
}>(), {
  focusSummitId: null,
});

const emit = defineEmits<{
  "update:category": [category: string];
  "update:year": [year: string];
  "open-climb-log": [variantId: string];
}>();

const FAVORITES_STORAGE_KEY = "mystravastats-climb-map-favorites-v1";
const mapContainer = ref<HTMLDivElement | null>(null);
const map = shallowRef<L.Map | null>(null);
const markerLayer = shallowRef<L.LayerGroup | null>(null);
const markerBySummitId = new Map<string, L.Marker>();
const selectedSummitId = ref<string | null>(null);
const favoriteIds = ref<Set<string>>(new Set());
let resizeObserver: ResizeObserver | null = null;

const localFilters = reactive<{
  country: string;
  massif: string;
  status: ClimbMapStatusFilter;
}>({
  country: CLIMB_MAP_ALL_FILTER,
  massif: CLIMB_MAP_ALL_FILTER,
  status: "ALL",
});

const categoryFilter = computed({
  get: () => props.category,
  set: (value: string) => emit("update:category", value),
});

const yearFilter = computed({
  get: () => props.yearLabel,
  set: (value: string) => emit("update:year", value),
});

const allSummits = computed(() => buildClimbMapSummits(props.climbs));
const activeFilters = computed<ClimbMapFilters>(() => ({
  country: localFilters.country,
  massif: localFilters.massif,
  category: categoryFilter.value,
  status: localFilters.status,
}));
const visibleSummits = computed(() => filterClimbMapSummits(
  allSummits.value,
  activeFilters.value,
  favoriteIds.value,
));
const selectedSummit = computed(() => (
  visibleSummits.value.find((summit) => summit.id === selectedSummitId.value) ?? null
));
const visibleVariantCount = computed(() => visibleSummits.value.reduce(
  (total, summit) => total + summit.variants.length,
  0,
));
const visibleClimbedCount = computed(() => visibleSummits.value.filter((summit) => summit.climbed).length);
const visibleFavoriteCount = computed(() => visibleSummits.value.filter((summit) => favoriteIds.value.has(summit.id)).length);

const countryOptions = computed(() => [...new Set(allSummits.value.map((summit) => summit.country))]
  .filter(Boolean)
  .sort((left, right) => left.localeCompare(right)));
const massifOptions = computed(() => [...new Set(allSummits.value
  .filter((summit) => localFilters.country === CLIMB_MAP_ALL_FILTER || summit.country === localFilters.country)
  .map((summit) => summit.massif))]
  .filter(Boolean)
  .sort((left, right) => left.localeCompare(right)));
const categoryOptions = computed(() => {
  const knownOrder = ["HC", "1", "2", "3", "4"];
  const categories = new Set(allSummits.value.flatMap((summit) => summit.variants.map((variant) => variant.category)));
  return [
    ...knownOrder.filter((category) => categories.has(category)),
    ...[...categories].filter((category) => !knownOrder.includes(category) && category !== "—").sort(),
  ];
});

function loadFavorites() {
  try {
    const stored = JSON.parse(localStorage.getItem(FAVORITES_STORAGE_KEY) ?? "[]") as unknown;
    favoriteIds.value = new Set(Array.isArray(stored) ? stored.filter((value): value is string => typeof value === "string") : []);
  } catch {
    favoriteIds.value = new Set();
  }
}

function persistFavorites() {
  try {
    localStorage.setItem(FAVORITES_STORAGE_KEY, JSON.stringify([...favoriteIds.value].sort()));
  } catch {
    // Favorites remain available for this session when storage is unavailable.
  }
}

function toggleFavorite(summitId: string) {
  const next = new Set(favoriteIds.value);
  if (next.has(summitId)) {
    next.delete(summitId);
  } else {
    next.add(summitId);
  }
  favoriteIds.value = next;
  persistFavorites();
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function summitMarkerIcon(summit: ClimbMapSummit): L.DivIcon {
  const favorite = favoriteIds.value.has(summit.id);
  const state = favorite ? "favorite" : summit.climbed ? "climbed" : "unclimbed";
  const selected = selectedSummitId.value === summit.id ? " climb-map-marker--selected" : "";
  const symbol = favorite ? "★" : summit.climbed ? "✓" : "•";
  return L.divIcon({
    className: "climb-map-div-icon",
    html: `<span class="climb-map-marker climb-map-marker--${state}${selected}" aria-hidden="true"><b>${symbol}</b></span>`,
    iconSize: [32, 32],
    iconAnchor: [16, 16],
  });
}

function clusterMarkerIcon(count: number, climbedCount: number): L.DivIcon {
  return L.divIcon({
    className: "climb-map-div-icon",
    html: `<span class="climb-map-cluster"><strong>${count}</strong><small>${climbedCount} ✓</small></span>`,
    iconSize: [48, 48],
    iconAnchor: [24, 24],
  });
}

function fitMapToSummits(summits: ClimbMapSummit[], maxZoom = 9) {
  const currentMap = map.value;
  if (!currentMap || summits.length === 0) {
    return;
  }
  if (summits.length === 1) {
    currentMap.setView([summits[0].latitude, summits[0].longitude], Math.min(11, maxZoom + 2));
    return;
  }
  const bounds = L.latLngBounds(summits.map((summit) => [summit.latitude, summit.longitude] as L.LatLngTuple));
  currentMap.fitBounds(bounds.pad(0.12), { maxZoom });
}

function renderMarkers(fit = false) {
  const currentMap = map.value;
  const currentLayer = markerLayer.value;
  if (!currentMap || !currentLayer) {
    return;
  }

  currentLayer.clearLayers();
  markerBySummitId.clear();
  const summits = visibleSummits.value;
  const clusters = clusterClimbMapSummits(summits, currentMap.getZoom());

  clusters.forEach((cluster) => {
    if (cluster.summits.length > 1) {
      const climbedCount = cluster.summits.filter((summit) => summit.climbed).length;
      const marker = L.marker([cluster.latitude, cluster.longitude], {
        icon: clusterMarkerIcon(cluster.summits.length, climbedCount),
        keyboard: true,
        title: `${cluster.summits.length} cols — zoom to explore`,
      });
      marker.on("click", () => fitMapToSummits(cluster.summits, Math.min(11, currentMap.getZoom() + 2)));
      marker.addTo(currentLayer);
      return;
    }

    const summit = cluster.summits[0];
    const marker = L.marker([summit.latitude, summit.longitude], {
      icon: summitMarkerIcon(summit),
      keyboard: true,
      title: summit.name,
      riseOnHover: true,
    });
    marker.bindTooltip(
      `<strong>${escapeHtml(summit.name)}</strong><br>${summit.summitAltitude.toLocaleString("fr-FR")} m · ${summit.variants.length} versant${summit.variants.length > 1 ? "s" : ""}`,
      { direction: "top", offset: [0, -12] },
    );
    marker.on("click", () => {
      selectedSummitId.value = summit.id;
      renderMarkers(false);
    });
    marker.addTo(currentLayer);
    markerBySummitId.set(summit.id, marker);
  });

  if (fit) {
    fitMapToSummits(summits);
  }
}

async function focusSummit(summitId: string) {
  const summit = allSummits.value.find((candidate) => candidate.id === summitId);
  if (!summit || !map.value) {
    return;
  }
  localFilters.country = CLIMB_MAP_ALL_FILTER;
  localFilters.massif = CLIMB_MAP_ALL_FILTER;
  localFilters.status = "ALL";
  categoryFilter.value = CLIMB_MAP_ALL_FILTER;
  selectedSummitId.value = summitId;
  await nextTick();
  renderMarkers(false);
  map.value.flyTo([summit.latitude, summit.longitude], 11, { duration: 0.65 });
}

function resetFilters() {
  localFilters.country = CLIMB_MAP_ALL_FILTER;
  localFilters.massif = CLIMB_MAP_ALL_FILTER;
  localFilters.status = "ALL";
  categoryFilter.value = CLIMB_MAP_ALL_FILTER;
}

function formatDecimal(value: number): string {
  return Number.isFinite(value) ? value.toLocaleString("fr-FR", { minimumFractionDigits: 1, maximumFractionDigits: 1 }) : "—";
}

function formatDate(value?: string): string {
  if (!value) return "—";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime())
    ? value.substring(0, 10)
    : parsed.toLocaleDateString("fr-FR", { day: "2-digit", month: "short", year: "numeric" });
}

watch(() => localFilters.country, () => {
  if (localFilters.massif !== CLIMB_MAP_ALL_FILTER && !massifOptions.value.includes(localFilters.massif)) {
    localFilters.massif = CLIMB_MAP_ALL_FILTER;
  }
});

watch(visibleSummits, () => {
  if (selectedSummitId.value && !visibleSummits.value.some((summit) => summit.id === selectedSummitId.value)) {
    selectedSummitId.value = null;
  }
  renderMarkers(true);
});

watch(selectedSummitId, () => renderMarkers(false));
watch(() => props.focusSummitId, (summitId) => {
  if (summitId) {
    void focusSummit(summitId);
  }
});

onMounted(async () => {
  loadFavorites();
  if (!mapContainer.value) return;
  const leafletMap = L.map(mapContainer.value, {
    zoomControl: true,
    minZoom: 3,
    maxZoom: 15,
  });
  map.value = leafletMap;
  markerLayer.value = L.layerGroup().addTo(leafletMap);
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
  }).addTo(leafletMap);
  leafletMap.setView([46.3, 7.3], 5);
  leafletMap.on("zoomend", () => renderMarkers(false));
  resizeObserver = new ResizeObserver(() => leafletMap.invalidateSize({ pan: false }));
  resizeObserver.observe(mapContainer.value);
  await nextTick();
  leafletMap.invalidateSize();
  renderMarkers(true);
  if (props.focusSummitId) {
    await focusSummit(props.focusSummitId);
  }
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  map.value?.remove();
  map.value = null;
  markerLayer.value = null;
  markerBySummitId.clear();
});

defineExpose({ focusSummit });
</script>

<template>
  <div class="climb-map-shell">
    <header class="climb-map-header">
      <div>
        <p class="climb-map-kicker">Interactive summit atlas</p>
        <h2>Explore climbed and undiscovered cols</h2>
        <p>One marker per summit; select it to compare all its starting variants.</p>
      </div>
      <div class="climb-map-summary" aria-label="Visible climb map summary">
        <span><strong>{{ visibleSummits.length }}</strong> summits</span>
        <span><strong>{{ visibleVariantCount }}</strong> variants</span>
        <span><strong>{{ visibleClimbedCount }}</strong> climbed</span>
        <span><strong>{{ visibleFavoriteCount }}</strong> favourites</span>
      </div>
    </header>

    <div class="climb-map-filters" aria-label="Climb map filters">
      <label>
        <span>Year</span>
        <select v-model="yearFilter" class="form-select form-select-sm">
          <option v-for="year in yearOptions" :key="year" :value="year">{{ year }}</option>
        </select>
      </label>
      <label>
        <span>Country</span>
        <select v-model="localFilters.country" class="form-select form-select-sm">
          <option :value="CLIMB_MAP_ALL_FILTER">All countries</option>
          <option v-for="country in countryOptions" :key="country" :value="country">{{ country }}</option>
        </select>
      </label>
      <label>
        <span>Massif</span>
        <select v-model="localFilters.massif" class="form-select form-select-sm">
          <option :value="CLIMB_MAP_ALL_FILTER">All massifs</option>
          <option v-for="massif in massifOptions" :key="massif" :value="massif">{{ massif }}</option>
        </select>
      </label>
      <label>
        <span>Category</span>
        <select v-model="categoryFilter" class="form-select form-select-sm">
          <option :value="CLIMB_MAP_ALL_FILTER">All categories</option>
          <option v-for="categoryOption in categoryOptions" :key="categoryOption" :value="categoryOption">
            Cat. {{ categoryOption }}
          </option>
        </select>
      </label>
      <label>
        <span>Status</span>
        <select v-model="localFilters.status" class="form-select form-select-sm">
          <option value="ALL">All statuses</option>
          <option value="CLIMBED">Climbed</option>
          <option value="UNCLIMBED">To discover</option>
          <option value="FAVORITE">Favourites</option>
        </select>
      </label>
      <button type="button" class="btn btn-sm btn-outline-secondary" @click="resetFilters">Reset</button>
      <button type="button" class="btn btn-sm btn-outline-primary" @click="fitMapToSummits(visibleSummits)">
        <i class="fa-solid fa-expand" aria-hidden="true" /> Recenter
      </button>
    </div>

    <div class="climb-map-legend" aria-label="Climb map legend">
      <span><i class="legend-dot legend-dot--climbed">✓</i> Climbed</span>
      <span><i class="legend-dot legend-dot--unclimbed">•</i> To discover</span>
      <span><i class="legend-dot legend-dot--favorite">★</i> Favourite</span>
      <span><i class="legend-dot legend-dot--cluster">12</i> Grouped summits</span>
    </div>

    <div class="climb-map-layout">
      <div class="climb-map-stage">
        <div ref="mapContainer" class="climb-map-canvas" role="region" aria-label="Interactive climb map" />
        <div v-if="visibleSummits.length === 0" class="climb-map-empty">
          <strong>No cols match these filters.</strong>
          <button type="button" class="btn btn-sm btn-outline-primary" @click="resetFilters">Clear filters</button>
        </div>
      </div>

      <aside class="climb-map-details" aria-live="polite">
        <template v-if="selectedSummit">
          <div class="climb-detail-heading">
            <div>
              <p>{{ selectedSummit.country }} · {{ selectedSummit.massif }}</p>
              <h3>{{ selectedSummit.name }}</h3>
              <span>{{ selectedSummit.summitAltitude.toLocaleString("fr-FR") }} m · {{ selectedSummit.variants.length }} variant{{ selectedSummit.variants.length > 1 ? "s" : "" }}</span>
            </div>
            <button
              type="button"
              class="favorite-button"
              :class="{ 'favorite-button--active': favoriteIds.has(selectedSummit.id) }"
              :aria-label="favoriteIds.has(selectedSummit.id) ? `Remove ${selectedSummit.name} from favourites` : `Add ${selectedSummit.name} to favourites`"
              :aria-pressed="favoriteIds.has(selectedSummit.id)"
              @click="toggleFavorite(selectedSummit.id)"
            >
              <i :class="favoriteIds.has(selectedSummit.id) ? 'fa-solid fa-star' : 'fa-regular fa-star'" aria-hidden="true" />
            </button>
          </div>

          <article v-for="variant in selectedSummit.variants" :key="variant.id" class="climb-variant-card">
            <div class="climb-variant-title">
              <h4>{{ variant.label }}</h4>
              <span :class="variant.climbed ? 'variant-status--climbed' : 'variant-status--unclimbed'">
                {{ variant.climbed ? "Climbed" : "To discover" }}
              </span>
            </div>
            <div class="climb-variant-metrics">
              <span><small>Category</small><strong>{{ variant.category }}</strong></span>
              <span><small>Distance</small><strong>{{ formatDecimal(variant.details.lengthKm) }} km</strong></span>
              <span><small>Elevation</small><strong>+{{ variant.details.totalAscent.toLocaleString("fr-FR") }} m</strong></span>
              <span><small>Difficulty</small><strong>{{ variant.details.difficulty.toLocaleString("fr-FR") }} pts</strong></span>
              <span><small>Average</small><strong>{{ formatDecimal(variant.details.averageGradient) }}%</strong></span>
              <span><small>Maximum</small><strong>{{ variant.details.maximumGradient == null ? "—" : `${formatDecimal(variant.details.maximumGradient)}%` }}</strong></span>
            </div>
            <div class="climb-personal-stats">
              <span><strong>{{ variant.details.ascentCount }}</strong> ascent{{ variant.details.ascentCount > 1 ? "s" : "" }}</span>
              <span v-if="variant.details.bestAscent">
                Best {{ formatTime(variant.details.bestAscent.durationSeconds) }} · {{ formatDate(variant.details.bestAscent.date) }}
              </span>
              <span v-else>No personal ascent in {{ yearLabel.toLowerCase() }}</span>
            </div>
            <div class="climb-variant-actions">
              <RouterLink
                class="btn btn-sm btn-primary"
                :to="{ name: 'climb-detail', params: { variantId: variant.id } }"
              >
                Detailed sheet
              </RouterLink>
              <button type="button" class="btn btn-sm btn-primary" @click="emit('open-climb-log', variant.id)">
                Open in climb log
              </button>
              <a v-if="variant.details.sourceUrl" :href="variant.details.sourceUrl" target="_blank" rel="noreferrer" class="btn btn-sm btn-outline-secondary">
                Source <i class="fa-solid fa-arrow-up-right-from-square" aria-hidden="true" />
              </a>
            </div>
          </article>
        </template>
        <div v-else class="climb-map-selection-empty">
          <span class="selection-empty-icon"><i class="fa-solid fa-mountain-sun" aria-hidden="true" /></span>
          <h3>Select a summit</h3>
          <p>Click a marker to inspect every variant, its difficulty and your best ascent for {{ yearLabel }}.</p>
          <dl>
            <div><dt>Climbed</dt><dd>{{ visibleClimbedCount }}</dd></div>
            <div><dt>To discover</dt><dd>{{ visibleSummits.length - visibleClimbedCount }}</dd></div>
          </dl>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.climb-map-shell {
  display: flex;
  width: 100%;
  min-width: 0;
  flex-direction: column;
  gap: 13px;
}

.climb-map-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.climb-map-kicker {
  margin: 0 0 4px;
  color: #167052;
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.climb-map-header h2 {
  margin: 0;
  color: var(--ms-text);
  font-size: 1.3rem;
  font-weight: 800;
}

.climb-map-header p:last-child {
  margin: 5px 0 0;
  color: var(--ms-text-muted);
  font-size: 0.88rem;
}

.climb-map-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(75px, 1fr));
  gap: 7px;
}

.climb-map-summary span {
  display: flex;
  min-height: 54px;
  flex-direction: column;
  justify-content: center;
  padding: 6px 9px;
  border: 1px solid #cce4da;
  border-radius: 11px;
  color: #52635c;
  background: #f4fbf8;
  font-size: 0.68rem;
  text-align: center;
}

.climb-map-summary strong {
  color: #176d50;
  font-size: 1.02rem;
}

.climb-map-filters {
  display: grid;
  grid-template-columns: repeat(5, minmax(125px, 1fr)) auto auto;
  align-items: end;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--ms-border);
  border-radius: 13px;
  background: var(--ms-surface);
}

.climb-map-filters label {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.climb-map-filters label > span {
  color: var(--ms-text-muted);
  font-size: 0.67rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.climb-map-filters select {
  min-width: 0;
  border-color: var(--ms-border);
  border-radius: 9px;
  font-size: 0.78rem;
}

.climb-map-filters .btn {
  white-space: nowrap;
}

.climb-map-legend {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 700;
}

.climb-map-legend span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.legend-dot {
  display: inline-grid;
  width: 20px;
  height: 20px;
  place-items: center;
  border: 2px solid #fff;
  border-radius: 50%;
  color: white;
  box-shadow: 0 0 0 1px rgb(32 47 41 / 22%);
  font-size: 0.62rem;
  font-style: normal;
}

.legend-dot--climbed { background: #198766; }
.legend-dot--unclimbed { background: #77818c; }
.legend-dot--favorite { background: #dc941d; }
.legend-dot--cluster { width: 25px; height: 25px; background: #233d47; font-size: 0.56rem; }

.climb-map-layout {
  display: grid;
  min-height: 640px;
  grid-template-columns: minmax(0, 1fr) minmax(330px, 380px);
  overflow: hidden;
  border: 1px solid var(--ms-border);
  border-radius: 16px;
  background: var(--ms-surface);
}

.climb-map-stage {
  position: relative;
  min-width: 0;
  min-height: 640px;
}

.climb-map-canvas {
  position: absolute;
  inset: 0;
  background: #e7eee9;
}

.climb-map-empty {
  position: absolute;
  z-index: 500;
  top: 50%;
  left: 50%;
  display: flex;
  min-width: 240px;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 18px;
  border: 1px solid var(--ms-border);
  border-radius: 13px;
  background: rgb(255 255 255 / 92%);
  box-shadow: var(--ms-shadow-soft);
  transform: translate(-50%, -50%);
}

.climb-map-details {
  max-height: 680px;
  overflow-y: auto;
  padding: 16px;
  border-left: 1px solid var(--ms-border);
  background: var(--ms-surface-strong);
}

.climb-detail-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 14px;
}

.climb-detail-heading p {
  margin: 0 0 3px;
  color: #9b4218;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.climb-detail-heading h3 {
  margin: 0;
  color: var(--ms-text);
  font-size: 1.18rem;
  font-weight: 800;
}

.climb-detail-heading span {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.favorite-button {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid #d9dee4;
  border-radius: 50%;
  color: #737d88;
  background: white;
}

.favorite-button--active {
  border-color: #ecc26c;
  color: #c47b08;
  background: #fff8df;
}

.climb-variant-card {
  padding: 13px;
  border: 1px solid var(--ms-border);
  border-radius: 13px;
  background: var(--ms-surface);
  box-shadow: 0 6px 18px rgb(26 39 34 / 6%);
}

.climb-variant-card + .climb-variant-card {
  margin-top: 10px;
}

.climb-variant-title {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}

.climb-variant-title h4 {
  margin: 0;
  color: var(--ms-text);
  font-size: 0.85rem;
  font-weight: 800;
  line-height: 1.3;
}

.climb-variant-title span {
  flex: 0 0 auto;
  padding: 2px 7px;
  border-radius: 999px;
  font-size: 0.62rem;
  font-weight: 800;
}

.variant-status--climbed {
  color: #116347;
  background: #e8f7f0;
}

.variant-status--unclimbed {
  color: #68727d;
  background: #edf0f3;
}

.climb-variant-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-top: 11px;
}

.climb-variant-metrics span {
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: 6px;
  border-radius: 8px;
  background: rgb(123 139 132 / 8%);
}

.climb-variant-metrics small {
  color: var(--ms-text-muted);
  font-size: 0.58rem;
  text-transform: uppercase;
}

.climb-variant-metrics strong {
  overflow: hidden;
  color: var(--ms-text);
  font-size: 0.72rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.climb-personal-stats {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 9px;
  color: var(--ms-text-muted);
  font-size: 0.7rem;
}

.climb-variant-actions {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  flex-wrap: wrap;
}

.climb-variant-actions .btn {
  font-size: 0.68rem;
}

.climb-map-selection-empty {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: var(--ms-text-muted);
  text-align: center;
}

.selection-empty-icon {
  display: grid;
  width: 56px;
  height: 56px;
  margin-bottom: 12px;
  place-items: center;
  border-radius: 18px;
  color: #197052;
  background: #ebf8f2;
  font-size: 1.35rem;
}

.climb-map-selection-empty h3 {
  margin: 0;
  color: var(--ms-text);
  font-size: 1rem;
}

.climb-map-selection-empty p {
  max-width: 280px;
  margin: 7px 0 15px;
  font-size: 0.8rem;
}

.climb-map-selection-empty dl {
  display: flex;
  gap: 8px;
  margin: 0;
}

.climb-map-selection-empty dl div {
  min-width: 95px;
  padding: 8px;
  border: 1px solid var(--ms-border);
  border-radius: 10px;
}

.climb-map-selection-empty dt {
  font-size: 0.62rem;
  text-transform: uppercase;
}

.climb-map-selection-empty dd {
  margin: 0;
  color: var(--ms-text);
  font-size: 1.05rem;
  font-weight: 800;
}

:deep(.climb-map-div-icon) {
  border: 0;
  background: transparent;
}

:deep(.climb-map-marker) {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 3px solid white;
  border-radius: 50% 50% 50% 12%;
  color: white;
  box-shadow: 0 3px 10px rgb(17 36 29 / 35%);
  font-size: 0.72rem;
  font-weight: 900;
  transform: rotate(-45deg);
}

:deep(.climb-map-marker > *) {
  font-weight: 900;
  transform: rotate(45deg);
}

:deep(.climb-map-marker--climbed) { background: #168565; }
:deep(.climb-map-marker--unclimbed) { background: #737d88; }
:deep(.climb-map-marker--favorite) { background: #d99018; }
:deep(.climb-map-marker--selected) { box-shadow: 0 0 0 4px rgb(252 76 2 / 35%), 0 4px 12px rgb(17 36 29 / 40%); }

:deep(.climb-map-cluster) {
  display: flex;
  width: 48px;
  height: 48px;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  border: 3px solid white;
  border-radius: 50%;
  color: white;
  background: #25414a;
  box-shadow: 0 4px 13px rgb(22 45 38 / 38%);
  line-height: 1;
}

:deep(.climb-map-cluster strong) { font-size: 0.78rem; }
:deep(.climb-map-cluster small) { margin-top: 3px; color: #bcebd9; font-size: 0.52rem; }

:deep(.leaflet-control-zoom a) {
  color: #24453a;
}

@media (max-width: 1200px) {
  .climb-map-filters {
    grid-template-columns: repeat(3, minmax(130px, 1fr)) auto auto;
  }
}

@media (max-width: 900px) {
  .climb-map-header {
    flex-direction: column;
  }

  .climb-map-summary {
    width: 100%;
  }

  .climb-map-layout {
    grid-template-columns: 1fr;
  }

  .climb-map-stage {
    min-height: 500px;
  }

  .climb-map-details {
    max-height: none;
    border-top: 1px solid var(--ms-border);
    border-left: 0;
  }
}

@media (max-width: 650px) {
  .climb-map-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .climb-map-filters {
    grid-template-columns: 1fr 1fr;
  }

  .climb-map-filters .btn {
    width: 100%;
  }

  .climb-map-stage {
    min-height: 430px;
  }
}
</style>

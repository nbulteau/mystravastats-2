<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import type {
  RouteRecommendation,
  ShapeRemixRecommendation,
} from "@/models/route-recommendation.model";
import { useContextStore } from "@/stores/context";
import { useRoutesStore } from "@/stores/routes";
import { formatTime } from "@/utils/formatters";

const contextStore = useContextStore();
const routesStore = useRoutesStore();
contextStore.updateCurrentView("routes");

const isLoading = ref<boolean>(false);
const filterDistance = ref<string>(routesStore.distanceTargetKm);
const filterElevation = ref<string>(routesStore.elevationTargetM);
const filterDuration = ref<string>(routesStore.durationTargetMin);
const filterStartDirection = ref<"" | "N" | "S" | "E" | "W">(routesStore.startDirection);
const filterRouteType = ref<"" | "RIDE" | "MTB" | "GRAVEL" | "RUN" | "TRAIL" | "HIKE">(routesStore.routeType);
const filterSeason = ref<"" | "WINTER" | "SPRING" | "SUMMER" | "AUTUMN">(routesStore.season);
const filterShape = ref<"" | "LOOP" | "OUT_AND_BACK" | "POINT_TO_POINT" | "FIGURE_EIGHT">(routesStore.shape);
const filterIncludeRemix = ref<boolean>(routesStore.includeRemix);
const filterLimit = ref<number>(routesStore.limit);

const closestLoops = computed(() => routesStore.result.closestLoops);
const smartVariants = computed(() => routesStore.result.variants);
const seasonalRoutes = computed(() => routesStore.result.seasonal);
const shapeMatches = computed(() => routesStore.result.shapeMatches);
const shapeRemixes = computed(() => routesStore.result.shapeRemixes);
const hasAnyResults = computed(() => routesStore.hasAnyResults);

const sectionTitles: Record<string, string> = {
  CLOSE_MATCH: "Closest match",
  SHORTER: "Shorter variant",
  LONGER: "Longer variant",
  HILLIER: "Hillier variant",
  SEASONAL: "Seasonal pick",
  SHAPE_MATCH: "Shape match",
  SHAPE_REMIX: "Shape remix",
};

async function applyFilters(force = false) {
  routesStore.updateFilters({
    distanceTargetKm: filterDistance.value,
    elevationTargetM: filterElevation.value,
    durationTargetMin: filterDuration.value,
    startDirection: filterStartDirection.value,
    routeType: filterRouteType.value,
    season: filterSeason.value,
    shape: filterShape.value,
    includeRemix: filterIncludeRemix.value,
    limit: filterLimit.value,
  });
  isLoading.value = true;
  try {
    await routesStore.ensureLoaded(force);
  } finally {
    isLoading.value = false;
  }
}

async function refreshData() {
  routesStore.invalidateCache();
  await applyFilters(true);
}

async function applyData() {
  await applyFilters(false);
}

async function resetFilters() {
  filterDistance.value = "";
  filterElevation.value = "";
  filterDuration.value = "";
  filterStartDirection.value = "";
  filterRouteType.value = "";
  filterSeason.value = "";
  filterShape.value = "";
  filterIncludeRemix.value = false;
  filterLimit.value = 6;
  await applyFilters();
}

function sectionTitle(recommendation: RouteRecommendation): string {
  return sectionTitles[recommendation.variantType] ?? recommendation.variantType;
}

function seasonLabel(value: string): string {
  if (!value) {
    return "Any season";
  }
  const normalized = value.toUpperCase();
  if (normalized === "AUTUMN") {
    return "Autumn";
  }
  return normalized.slice(0, 1) + normalized.slice(1).toLowerCase();
}

function formatDistance(value: number): string {
  return `${value.toFixed(1)} km`;
}

function formatElevation(value: number): string {
  return `${Math.round(value)} m`;
}

function shapeLabel(value?: string): string {
  if (!value) {
    return "Unknown";
  }
  return value.toLowerCase().replaceAll("_", " ");
}

onMounted(async () => {
  await applyFilters();
});
</script>

<template>
  <section class="routes-view">
    <header class="routes-panel routes-filters">
      <div class="routes-filters-grid">
        <label class="routes-field">
          <span>Distance target (km)</span>
          <input
            v-model="filterDistance"
            type="number"
            min="0"
            step="0.5"
            class="form-control"
            placeholder="Auto"
          >
        </label>

        <label class="routes-field">
          <span>Elevation target (m)</span>
          <input
            v-model="filterElevation"
            type="number"
            min="0"
            step="10"
            class="form-control"
            placeholder="Auto"
          >
        </label>

        <label class="routes-field">
          <span>Duration target (min)</span>
          <input
            v-model="filterDuration"
            type="number"
            min="0"
            step="5"
            class="form-control"
            placeholder="Auto"
          >
        </label>

        <label class="routes-field">
          <span>Departure direction</span>
          <select
            v-model="filterStartDirection"
            class="form-select"
          >
            <option value="">
              Any
            </option>
            <option value="N">
              North
            </option>
            <option value="S">
              South
            </option>
            <option value="E">
              East
            </option>
            <option value="W">
              West
            </option>
          </select>
        </label>

        <label class="routes-field">
          <span>Route type</span>
          <select
            v-model="filterRouteType"
            class="form-select"
          >
            <option value="">
              Any
            </option>
            <option value="RIDE">
              Ride
            </option>
            <option value="MTB">
              MTB
            </option>
            <option value="GRAVEL">
              Gravel
            </option>
            <option value="RUN">
              Run
            </option>
            <option value="TRAIL">
              Trail
            </option>
            <option value="HIKE">
              Hike
            </option>
          </select>
        </label>

        <label class="routes-field">
          <span>Season</span>
          <select
            v-model="filterSeason"
            class="form-select"
          >
            <option value="">
              Any
            </option>
            <option value="WINTER">
              Winter
            </option>
            <option value="SPRING">
              Spring
            </option>
            <option value="SUMMER">
              Summer
            </option>
            <option value="AUTUMN">
              Autumn
            </option>
          </select>
        </label>

        <label class="routes-field">
          <span>Shape</span>
          <select
            v-model="filterShape"
            class="form-select"
          >
            <option value="">
              Any
            </option>
            <option value="LOOP">
              Loop
            </option>
            <option value="OUT_AND_BACK">
              Out and back
            </option>
            <option value="POINT_TO_POINT">
              Point to point
            </option>
            <option value="FIGURE_EIGHT">
              Figure eight
            </option>
          </select>
        </label>

        <label class="routes-field">
          <span>Max recommendations</span>
          <input
            v-model.number="filterLimit"
            type="number"
            min="1"
            max="24"
            class="form-control"
          >
        </label>
      </div>

      <div class="routes-filter-actions">
        <label class="form-check routes-remix-toggle">
          <input
            v-model="filterIncludeRemix"
            class="form-check-input"
            type="checkbox"
          >
          <span class="form-check-label">Include shape remix (experimental)</span>
        </label>

        <div class="routes-buttons">
          <button
            class="btn btn-outline-secondary"
            type="button"
            :disabled="isLoading"
            @click="resetFilters"
          >
            Reset
          </button>
          <button
            class="btn btn-outline-primary"
            type="button"
            :disabled="isLoading"
            @click="refreshData"
          >
            Refresh
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="isLoading"
            @click="applyData"
          >
            Apply
          </button>
        </div>
      </div>
    </header>

    <div
      v-if="isLoading"
      class="routes-panel routes-loading"
    >
      Building route recommendations from your cache...
    </div>

    <div
      v-else-if="!hasAnyResults"
      class="routes-panel routes-empty"
    >
      No route recommendation found for these filters yet. Try broader filters or disable shape filtering.
    </div>

    <template v-else>
      <section class="routes-panel">
        <h2>MVP useful quickly: closest loops</h2>
        <div class="routes-grid">
          <article
            v-for="recommendation in closestLoops"
            :key="`closest-${recommendation.activity.id}`"
            class="route-card"
          >
            <div class="route-card-head">
              <span class="route-tag">
                {{ sectionTitle(recommendation) }}
              </span>
              <span class="route-score">{{ recommendation.matchScore.toFixed(1) }}%</span>
            </div>
            <RouterLink
              class="route-activity-link"
              :to="`/activities/${recommendation.activity.id}`"
            >
              {{ recommendation.activity.name }}
            </RouterLink>
            <p class="route-meta">
              {{ recommendation.activityDate }} • {{ seasonLabel(recommendation.season) }} •
              {{ formatDistance(recommendation.distanceKm) }} • {{ formatElevation(recommendation.elevationGainM) }} •
              {{ formatTime(recommendation.durationSec) }}
            </p>
            <ul class="route-reasons">
              <li
                v-for="reason in recommendation.reasons"
                :key="`${recommendation.activity.id}-${reason}`"
              >
                {{ reason }}
              </li>
            </ul>
          </article>
        </div>
      </section>

      <section class="routes-panel">
        <h2>Smart variants</h2>
        <div class="routes-grid routes-grid--compact">
          <article
            v-for="recommendation in smartVariants"
            :key="`variant-${recommendation.activity.id}-${recommendation.variantType}`"
            class="route-card"
          >
            <div class="route-card-head">
              <span class="route-tag route-tag--accent">
                {{ sectionTitle(recommendation) }}
              </span>
              <span class="route-score">{{ recommendation.matchScore.toFixed(1) }}%</span>
            </div>
            <RouterLink
              class="route-activity-link"
              :to="`/activities/${recommendation.activity.id}`"
            >
              {{ recommendation.activity.name }}
            </RouterLink>
            <p class="route-meta">
              {{ formatDistance(recommendation.distanceKm) }} • {{ formatElevation(recommendation.elevationGainM) }} •
              {{ formatTime(recommendation.durationSec) }}
            </p>
          </article>
        </div>
      </section>

      <section class="routes-panel">
        <h2>Shape match</h2>
        <div class="routes-grid routes-grid--compact">
          <article
            v-for="recommendation in shapeMatches"
            :key="`shape-${recommendation.activity.id}`"
            class="route-card"
          >
            <div class="route-card-head">
              <span class="route-tag route-tag--shape">
                {{ shapeLabel(recommendation.shape) }}
              </span>
              <span class="route-score">{{ recommendation.matchScore.toFixed(1) }}%</span>
            </div>
            <RouterLink
              class="route-activity-link"
              :to="`/activities/${recommendation.activity.id}`"
            >
              {{ recommendation.activity.name }}
            </RouterLink>
            <p class="route-meta">
              {{ recommendation.activityDate }} • shape confidence {{ recommendation.shapeScore?.toFixed(0) ?? "0" }}%
            </p>
          </article>
        </div>
      </section>

      <section
        v-if="shapeRemixes.length > 0"
        class="routes-panel"
      >
        <h2>Shape remix (experimental)</h2>
        <div class="routes-grid routes-grid--compact">
          <article
            v-for="remix in shapeRemixes"
            :key="remix.id"
            class="route-card route-card--experimental"
          >
            <div class="route-card-head">
              <span class="route-tag route-tag--experimental">
                {{ shapeLabel(remix.shape) }}
              </span>
              <span class="route-score">{{ remix.matchScore.toFixed(1) }}%</span>
            </div>
            <p class="route-meta">
              {{ formatDistance(remix.distanceKm) }} • {{ formatElevation(remix.elevationGainM) }} •
              {{ formatTime(remix.durationSec) }}
            </p>
            <ul class="route-reasons">
              <li
                v-for="reason in remix.reasons"
                :key="`${remix.id}-${reason}`"
              >
                {{ reason }}
              </li>
            </ul>
            <div class="route-components">
              <RouterLink
                v-for="component in remix.components"
                :key="`${remix.id}-${component.id}`"
                class="route-component-link"
                :to="`/activities/${component.id}`"
              >
                {{ component.name }}
              </RouterLink>
            </div>
          </article>
        </div>
      </section>
    </template>
  </section>
</template>

<style scoped>
.routes-view {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-bottom: 26px;
}

.routes-panel {
  background: #ffffff;
  border: 1px solid #dfe4ec;
  border-radius: 16px;
  padding: 14px;
  box-shadow: 0 6px 20px rgba(12, 21, 38, 0.05);
}

.routes-panel h2 {
  font-size: 1.04rem;
  margin: 0 0 10px;
  color: #253149;
}

.routes-filters-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(160px, 1fr));
  gap: 10px;
}

.routes-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.routes-field span {
  font-weight: 700;
  font-size: 0.86rem;
  color: #4d566a;
}

.routes-filter-actions {
  margin-top: 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.routes-remix-toggle {
  margin: 0;
  color: #4d566a;
  font-size: 0.9rem;
}

.routes-buttons {
  display: flex;
  gap: 8px;
}

.routes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 10px;
}

.routes-grid--compact {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.route-card {
  border: 1px solid #dfe4ec;
  border-radius: 12px;
  padding: 10px 11px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: #fbfdff;
}

.route-card--experimental {
  border-color: #f6cb9d;
  background: #fffaf5;
}

.route-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.route-tag {
  border-radius: 999px;
  background: #edf1f8;
  color: #44506a;
  font-size: 0.77rem;
  font-weight: 700;
  padding: 2px 8px;
}

.route-tag--accent {
  background: #fff0e7;
  color: #c45f26;
}

.route-tag--shape {
  background: #e9f5ef;
  color: #2f7a4e;
}

.route-tag--experimental {
  background: #ffe7cd;
  color: #bc6b2f;
}

.route-score {
  font-weight: 700;
  color: #516080;
  font-size: 0.84rem;
}

.route-activity-link {
  font-weight: 700;
  color: #2e5dd7;
  text-decoration: none;
}

.route-activity-link:hover {
  text-decoration: underline;
}

.route-meta {
  margin: 0;
  font-size: 0.88rem;
  color: #5d667c;
}

.route-reasons {
  margin: 0;
  padding-left: 1rem;
  color: #465069;
  font-size: 0.85rem;
}

.route-components {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}

.route-component-link {
  font-size: 0.82rem;
  border: 1px solid #dae2f1;
  border-radius: 999px;
  padding: 2px 8px;
  text-decoration: none;
  color: #2d4ea8;
  background: #f7faff;
}

.route-component-link:hover {
  text-decoration: underline;
}

.routes-loading,
.routes-empty {
  color: #5b6478;
  text-align: center;
  font-size: 1rem;
  padding: 20px;
}

@media (max-width: 900px) {
  .routes-filters-grid {
    grid-template-columns: repeat(2, minmax(140px, 1fr));
  }
}

@media (max-width: 650px) {
  .routes-filters-grid {
    grid-template-columns: 1fr;
  }
}
</style>

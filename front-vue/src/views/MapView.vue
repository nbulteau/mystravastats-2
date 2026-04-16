<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { MapTrack } from "@/models/map.model";
import { useContextStore } from "@/stores/context.js";
import AllTracksMap from "@/components/AllTracksMap.vue";
import { type MapViewport, useMapStore } from "@/stores/map";
import { getActivityTypeColor } from "@/utils/mapTrackColors";

const contextStore = useContextStore();
const mapStore = useMapStore();
contextStore.updateCurrentView("map");

const mapTracks = computed(() => mapStore.mapTracks);
const activityTypeFilter = ref("ALL");
const renderMode = ref<"TRACES" | "DENSITY">("TRACES");
const filtersKey = computed(() => mapStore.currentFiltersKey());
const isRefreshing = ref(false);
const recenterToken = ref(0);

type ActivityTypeSummary = {
  type: string;
  count: number;
  color: string;
};

const activityTypeSummaries = computed<ActivityTypeSummary[]>(() => {
  const counters = new Map<string, number>();
  mapTracks.value.forEach((track) => {
    const type = track.activityType || "Unknown";
    counters.set(type, (counters.get(type) ?? 0) + 1);
  });

  return Array.from(counters.entries())
    .map(([type, count]) => ({
      type,
      count,
      color: getActivityTypeColor(type),
    }))
    .sort((left, right) => right.count - left.count || left.type.localeCompare(right.type));
});

const filteredMapTracks = computed<MapTrack[]>(() => {
  if (activityTypeFilter.value === "ALL") {
    return mapTracks.value;
  }
  return mapTracks.value.filter((track) => track.activityType === activityTypeFilter.value);
});

watch(
  mapTracks,
  (tracks) => {
    if (activityTypeFilter.value === "ALL") {
      return;
    }
    const hasType = tracks.some((track) => track.activityType === activityTypeFilter.value);
    if (!hasType) {
      activityTypeFilter.value = "ALL";
    }
  },
  { immediate: true },
);

const totalTracks = computed(() =>
  filteredMapTracks.value.reduce((count, track) => (track.coordinates.length >= 2 ? count + 1 : count), 0),
);

const totalPoints = computed(() =>
  filteredMapTracks.value.reduce((count, track) => count + track.coordinates.length, 0),
);

const hasTracks = computed(() => totalTracks.value > 0);
const initialViewport = computed(() => mapStore.getViewportForCurrentFilters());

function onViewportChanged(viewport: MapViewport) {
  mapStore.setViewportForCurrentFilters(viewport);
}

function selectActivityTypeFilter(type: string) {
  activityTypeFilter.value = type;
  recenterToken.value += 1;
}

function setRenderMode(mode: "TRACES" | "DENSITY") {
  if (renderMode.value === mode) {
    return;
  }
  renderMode.value = mode;
  recenterToken.value += 1;
}

function recenterMap() {
  mapStore.clearViewportForCurrentFilters();
  recenterToken.value += 1;
}

async function refreshMap() {
  if (isRefreshing.value) {
    return;
  }
  isRefreshing.value = true;
  try {
    await mapStore.ensureLoaded(true);
  } finally {
    isRefreshing.value = false;
  }
}
</script>

<template>
  <section class="map-view">
    <header class="map-toolbar">
      <div class="map-toolbar__stats">
        <strong>{{ totalTracks }}</strong> tracks · {{ totalPoints.toLocaleString() }} points
      </div>
      <div class="map-toolbar__actions">
        <div class="btn-group btn-group-sm">
          <button
            type="button"
            class="btn"
            :class="renderMode === 'TRACES' ? 'btn-primary' : 'btn-outline-primary'"
            @click="setRenderMode('TRACES')"
          >
            Traces
          </button>
          <button
            type="button"
            class="btn"
            :class="renderMode === 'DENSITY' ? 'btn-primary' : 'btn-outline-primary'"
            @click="setRenderMode('DENSITY')"
          >
            Density
          </button>
        </div>
        <button
          type="button"
          class="btn btn-outline-secondary btn-sm"
          :disabled="!hasTracks"
          @click="recenterMap"
        >
          Recenter
        </button>
        <button
          type="button"
          class="btn btn-outline-primary btn-sm"
          :disabled="isRefreshing || mapStore.isLoading"
          @click="refreshMap"
        >
          {{ isRefreshing || mapStore.isLoading ? "Refreshing..." : "Refresh" }}
        </button>
      </div>
    </header>

    <div
      v-if="activityTypeSummaries.length > 0"
      class="map-type-filters"
    >
      <button
        type="button"
        class="type-pill"
        :class="{ 'type-pill--active': activityTypeFilter === 'ALL' }"
        @click="selectActivityTypeFilter('ALL')"
      >
        <span class="type-pill__dot" />
        All
        <strong>{{ mapTracks.length }}</strong>
      </button>
      <button
        v-for="summary in activityTypeSummaries"
        :key="summary.type"
        type="button"
        class="type-pill"
        :class="{ 'type-pill--active': activityTypeFilter === summary.type }"
        @click="selectActivityTypeFilter(summary.type)"
      >
        <span
          class="type-pill__dot"
          :style="{ backgroundColor: summary.color }"
        />
        {{ summary.type }}
        <strong>{{ summary.count }}</strong>
      </button>
    </div>

    <AllTracksMap
      :map-tracks="filteredMapTracks"
      :dataset-key="filtersKey"
      :loading="mapStore.isLoading"
      :error="mapStore.error"
      :initial-viewport="initialViewport"
      :recenter-token="recenterToken"
      :render-mode="renderMode"
      @viewport-changed="onViewportChanged"
    />
  </section>
</template>

<style scoped>
.map-view {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.map-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border: 1px solid var(--ms-border);
  border-radius: 12px;
  background: color-mix(in srgb, var(--ms-surface-strong) 88%, white);
}

.map-toolbar__stats {
  font-size: 0.92rem;
  color: var(--ms-text-muted);
}

.map-toolbar__stats strong {
  color: var(--ms-text);
}

.map-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.map-type-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.type-pill {
  border: 1px solid var(--ms-border);
  background: var(--ms-surface-strong);
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 0.82rem;
  color: var(--ms-text-muted);
  display: inline-flex;
  align-items: center;
  gap: 7px;
  transition: all 0.2s ease;
}

.type-pill:hover {
  border-color: color-mix(in srgb, var(--ms-primary) 45%, var(--ms-border));
}

.type-pill--active {
  border-color: color-mix(in srgb, var(--ms-primary) 62%, var(--ms-border));
  box-shadow: 0 0 0 2px rgba(252, 76, 2, 0.12) inset;
  color: var(--ms-text);
}

.type-pill__dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--ms-primary);
}

.type-pill strong {
  color: var(--ms-text);
  font-size: 0.78rem;
}

@media (max-width: 720px) {
  .map-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .map-toolbar__actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>

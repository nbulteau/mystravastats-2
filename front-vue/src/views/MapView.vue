<script setup lang="ts">
import { computed, ref, watch, onMounted } from "vue";
import type { MapPassageSegment, MapPassages, MapTrack } from "@/models/map.model";
import { useContextStore } from "@/stores/context.js";
import AllTracksMap from "@/components/AllTracksMap.vue";
import { type MapViewport, useMapStore } from "@/stores/map";
import { getActivityTypeColor } from "@/utils/mapTrackColors";

const contextStore = useContextStore();
const mapStore = useMapStore();
onMounted(() => contextStore.updateCurrentView("map"));

const mapTracks = computed(() => mapStore.mapTracks);
const mapPassages = computed(() => mapStore.mapPassages);
const activityTypeFilter = ref("ALL");
const renderMode = ref<"TRACES" | "PASSAGES" | "POINT_DENSITY">("TRACES");
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

const filteredMapPassages = computed<MapPassages>(() => {
  if (activityTypeFilter.value === "ALL") {
    return mapPassages.value;
  }

  const matchingActivityCount = activityTypeSummaries.value.find((summary) => summary.type === activityTypeFilter.value)?.count ?? 0;
  const segments = mapPassages.value.segments
    .map((segment): MapPassageSegment | null => {
      const passageCount = segment.activityTypeCounts?.[activityTypeFilter.value] ?? 0;
      if (passageCount <= 0) {
        return null;
      }
      const ratio = segment.passageCount > 0 ? passageCount / segment.passageCount : 0;
      return {
        ...segment,
        passageCount,
        activityCount: passageCount,
        distanceKm: segment.distanceKm * ratio,
      };
    })
    .filter((segment): segment is MapPassageSegment => segment !== null);

  return {
    ...mapPassages.value,
    segments,
    includedActivities: matchingActivityCount,
  };
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

const totalPassageCorridors = computed(() => filteredMapPassages.value.segments.length);
const maxPassageCount = computed(() =>
  filteredMapPassages.value.segments.reduce((max, segment) => Math.max(max, segment.passageCount), 0),
);
const omittedPassageSegments = computed(() => filteredMapPassages.value.omittedSegments ?? 0);
const toolbarStats = computed(() => {
  if (renderMode.value === "PASSAGES") {
    const suffix = omittedPassageSegments.value > 0 ? ` · ${omittedPassageSegments.value.toLocaleString()} hidden` : "";
    return `${totalPassageCorridors.value} corridors · max ${maxPassageCount.value} passes${suffix}`;
  }
  return `${totalTracks.value} tracks · ${totalPoints.value.toLocaleString()} points`;
});

const hasTracks = computed(() => totalTracks.value > 0);
const initialViewport = computed(() => mapStore.getViewportForCurrentFilters());

function onViewportChanged(viewport: MapViewport) {
  mapStore.setViewportForCurrentFilters(viewport);
}

function selectActivityTypeFilter(type: string) {
  activityTypeFilter.value = type;
  recenterToken.value += 1;
}

function setRenderMode(mode: "TRACES" | "PASSAGES" | "POINT_DENSITY") {
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
    if (renderMode.value === "PASSAGES") {
      await mapStore.ensurePassagesLoaded(true);
    }
  } finally {
    isRefreshing.value = false;
  }
}

watch(
  () => [contextStore.currentActivityType, contextStore.currentYear, renderMode.value],
  () => {
    if (renderMode.value === "PASSAGES") {
      void mapStore.ensurePassagesLoaded();
    }
  },
);
</script>

<template>
  <section class="map-view">
    <header class="map-toolbar">
      <div class="map-toolbar__stats">
        {{ toolbarStats }}
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
            :class="renderMode === 'PASSAGES' ? 'btn-primary' : 'btn-outline-primary'"
            @click="setRenderMode('PASSAGES')"
          >
            Frequency
          </button>
          <button
            type="button"
            class="btn"
            :class="renderMode === 'POINT_DENSITY' ? 'btn-primary' : 'btn-outline-primary'"
            @click="setRenderMode('POINT_DENSITY')"
          >
            Point density
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
      <div class="map-type-filters__hint">
        Type filters below refine the current top selection.
      </div>
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
      :map-passages="filteredMapPassages"
      :dataset-key="filtersKey"
      :loading="mapStore.isLoading || (renderMode === 'PASSAGES' && mapStore.isPassagesLoading)"
      :error="renderMode === 'PASSAGES' ? (mapStore.passagesError || mapStore.error) : mapStore.error"
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
  align-items: center;
}

.map-type-filters__hint {
  width: 100%;
  margin-bottom: 2px;
  font-size: 0.78rem;
  color: var(--ms-text-muted);
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

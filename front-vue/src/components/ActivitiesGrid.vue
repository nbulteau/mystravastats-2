<script setup lang="ts">
import VGrid, { type ColumnProp, type ColumnRegular, VGridVueTemplate } from "@revolist/vue3-datagrid";
import type { Activity } from "@/models/activity.model";
import DistanceCellRenderer from "@/components/cell-renderers/DistanceCellRenderer.vue";
import ElapsedTimeCellRenderer from "@/components/cell-renderers/ElapsedTimeCellRenderer.vue";
import ElevationGainCellRenderer from "@/components/cell-renderers/ElevationGainCellRenderer.vue";
import NameCellRenderer from "@/components/cell-renderers/NameCellRenderer.vue";
import DateCellRenderer from "@/components/cell-renderers/DateCellRenderer.vue";
import GradientCellRenderer from "@/components/cell-renderers/GradientCellRenderer.vue";
import ActivityTypeCellRenderer from "@/components/cell-renderers/ActivityTypeCellRenderer.vue";
import AverageSpeedCellRenderer from "@/components/cell-renderers/AverageSpeedCellRenderer.vue";
import BestSpeedFor1000mCellRenderer from "@/components/cell-renderers/BestSpeedFor1000mCellRenderer.vue";
import { computed, ref } from "vue";
import { ErrorService } from "@/services/error.service";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";

const props = defineProps<{
  activities: Activity[];
  currentActivity: string;
  currentYear: string;
  isLoading?: boolean;
  error?: string | null;
}>();

const uiStore = useUiStore();

const searchQuery = ref("");
const activityTypeFilter = ref("ALL");
const distanceMinKm = ref<number | null>(null);
const elevationMinM = ref<number | null>(null);
const durationMinMin = ref<number | null>(null);
const commuteOnly = ref(false);
const withHeartRate = ref(false);
const withPower = ref(false);

const availableActivityTypes = computed(() =>
  Array.from(new Set(props.activities.map((activity) => activity.type).filter((type) => !!type))).sort((a, b) =>
    a.localeCompare(b),
  ),
);

function toNumberOrZero(value: unknown): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function hasCommuteFlag(activity: Activity): boolean {
  const raw = activity as Activity & { isCommute?: boolean; commute?: boolean };
  return Boolean(raw.commute ?? raw.isCommute ?? false);
}

function resolveAverageHeartRate(activity: Activity): number {
  const raw = activity as Activity & { averageHeartRate?: number; average_heartrate?: number };
  return toNumberOrZero(raw.averageHeartrate ?? raw.averageHeartRate ?? raw.average_heartrate);
}

function resolvePower(activity: Activity): number {
  const raw = activity as Activity & {
    average_watts?: number;
    weighted_average_watts?: number;
    bestPowerFor20Minutes?: number;
    bestPowerFor60Minutes?: number;
    best_power_for_20_minutes?: number;
    best_power_for_60_minutes?: number;
  };
  const averageWatts = toNumberOrZero(raw.averageWatts ?? raw.average_watts);
  const weightedAverageWatts = toNumberOrZero(raw.weightedAverageWatts ?? raw.weighted_average_watts);
  const best20 = toNumberOrZero(raw.bestPowerFor20minutes ?? raw.bestPowerFor20Minutes ?? raw.best_power_for_20_minutes);
  const best60 = toNumberOrZero(raw.bestPowerFor60minutes ?? raw.bestPowerFor60Minutes ?? raw.best_power_for_60_minutes);
  return Math.max(averageWatts, weightedAverageWatts, best20, best60);
}

const filteredActivities = computed<Activity[]>(() => {
  const query = searchQuery.value.trim().toLowerCase();
  return props.activities.filter((activity) => {
    if (query && !activity.name.toLowerCase().includes(query)) {
      return false;
    }
    if (activityTypeFilter.value !== "ALL" && activity.type !== activityTypeFilter.value) {
      return false;
    }

    const distanceKm = (Number(activity.distance) || 0) / 1000;
    if (distanceMinKm.value !== null && distanceKm < distanceMinKm.value) {
      return false;
    }

    const elevation = Number(activity.totalElevationGain) || 0;
    if (elevationMinM.value !== null && elevation < elevationMinM.value) {
      return false;
    }

    const durationMinutes = (Number(activity.elapsedTime) || 0) / 60;
    if (durationMinMin.value !== null && durationMinutes < durationMinMin.value) {
      return false;
    }

    if (commuteOnly.value && !hasCommuteFlag(activity)) {
      return false;
    }
    if (withHeartRate.value && resolveAverageHeartRate(activity) <= 0) {
      return false;
    }
    return !(withPower.value && resolvePower(activity) <= 0);

  });
});

const hasActiveFilters = computed(
  () =>
    searchQuery.value.trim().length > 0 ||
    activityTypeFilter.value !== "ALL" ||
    distanceMinKm.value !== null ||
    elevationMinM.value !== null ||
    durationMinMin.value !== null ||
    commuteOnly.value ||
    withHeartRate.value ||
    withPower.value,
);

function resetFilters() {
  searchQuery.value = "";
  activityTypeFilter.value = "ALL";
  distanceMinKm.value = null;
  elevationMinM.value = null;
  durationMinMin.value = null;
  commuteOnly.value = false;
  withHeartRate.value = false;
  withPower.value = false;
}

async function csvExport() {
  let url = `/api/activities/csv?activityType=${encodeURIComponent(props.currentActivity)}`;
  if (props.currentYear !== "All years") {
    url = `${url}&year=${encodeURIComponent(props.currentYear)}`;
  }

  let response: Response;
  try {
    response = await fetch(url);
  } catch {
    uiStore.showToast({
      id: `csv-export-toast-${Date.now()}`,
      type: ToastTypeEnum.ERROR,
      message: "Unable to export CSV right now. Please retry.",
      timeout: 5000,
    });
    return;
  }

  if (!response.ok) {
    await ErrorService.catchError(response);
    return;
  }

  let objectUrl: string | null = null;
  try {
    const blob = await response.blob();
    objectUrl = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = objectUrl;
    const fileName = `activities-${props.currentActivity}-${props.currentYear}.csv`;
    link.setAttribute("download", fileName);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  } catch {
    uiStore.showToast({
      id: `csv-export-toast-${Date.now()}`,
      type: ToastTypeEnum.ERROR,
      message: "CSV export failed unexpectedly. Please retry.",
      timeout: 5000,
    });
  } finally {
    if (objectUrl) {
      window.URL.revokeObjectURL(objectUrl);
    }
  }
}

const numericCompare = (prop: ColumnProp, a: { [x: string]: unknown }, b: { [x: string]: unknown }) => {
  const aValue = Number(a[prop]);
  const bValue = Number(b[prop]);
  if (!Number.isFinite(aValue) && !Number.isFinite(bValue)) {
    return 0;
  }
  if (!Number.isFinite(aValue)) {
    return -1;
  }
  if (!Number.isFinite(bValue)) {
    return 1;
  }
  return aValue - bValue;
};

const columns = ref<ColumnRegular[]>([
  {
    prop: "name",
    name: "Activity",
    size: 420,
    pin: "colPinStart",
    cellTemplate: VGridVueTemplate(NameCellRenderer),
    sortable: true,
    columnType: "string",
  },
  {
    prop: "type",
    name: "Type",
    size: 170,
    cellTemplate: VGridVueTemplate(ActivityTypeCellRenderer),
    sortable: true,
    columnType: "string",
  },
  {
    prop: "distance",
    name: "Distance",
    size: 110,
    cellTemplate: VGridVueTemplate(DistanceCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "elapsedTime",
    name: "Elapsed time",
    size: 150,
    cellTemplate: VGridVueTemplate(ElapsedTimeCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "totalElevationGain",
    name: "Total elevation gain",
    size: 160,
    cellTemplate: VGridVueTemplate(ElevationGainCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "averageSpeed",
    name: "Average speed",
    size: 150,
    cellTemplate: VGridVueTemplate(AverageSpeedCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "bestSpeedForDistanceFor1000m",
    name: "Best speed for 1000m",
    size: 200,
    cellTemplate: VGridVueTemplate(BestSpeedFor1000mCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "bestElevationForDistanceFor500m",
    name: "Best gradient for 500m",
    size: 180,
    cellTemplate: VGridVueTemplate(GradientCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "bestElevationForDistanceFor1000m",
    name: "Best gradient for 1000m",
    size: 180,
    cellTemplate: VGridVueTemplate(GradientCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: "number",
  },
  {
    prop: "date",
    name: "Date",
    size: 200,
    cellTemplate: VGridVueTemplate(DateCellRenderer),
    sortable: true,
    columnType: "date",
  },
]);

const footerData = computed(() => {
  if (!filteredActivities.value.length) {
    return null;
  }

  const totalDistance = filteredActivities.value.reduce((sum, activity) => sum + (Number(activity.distance) || 0), 0);
  const totalElapsedTime = filteredActivities.value.reduce((sum, activity) => sum + (Number(activity.elapsedTime) || 0), 0);
  const totalMovingTime = filteredActivities.value.reduce((sum, activity) => sum + (Number(activity.movingTime) || 0), 0);
  const totalElevationGain = filteredActivities.value.reduce(
    (sum, activity) => sum + (Number(activity.totalElevationGain) || 0),
    0,
  );

  let averageSpeed = 0;
  if (totalMovingTime > 0) {
    averageSpeed = totalDistance / totalMovingTime;
  }

  const bestSpeed1000m = Math.max(...filteredActivities.value.map((activity) => Number(activity.bestSpeedForDistanceFor1000m) || 0));
  const bestGradient500m = Math.max(
    ...filteredActivities.value.map((activity) => Number(activity.bestElevationForDistanceFor500m) || 0),
  );
  const bestGradient1000m = Math.max(
    ...filteredActivities.value.map((activity) => Number(activity.bestElevationForDistanceFor1000m) || 0),
  );

  const footerType = activityTypeFilter.value !== "ALL" ? activityTypeFilter.value : props.currentActivity;

  return {
    name: "Totals",
    type: footerType,
    distance: totalDistance,
    elapsedTime: totalElapsedTime,
    totalElevationGain: totalElevationGain,
    averageSpeed: averageSpeed,
    bestSpeedForDistanceFor1000m: bestSpeed1000m,
    bestElevationForDistanceFor500m: bestGradient500m,
    bestElevationForDistanceFor1000m: bestGradient1000m,
    date: "",
  };
});

const pinnedBottomRows = computed(() => (footerData.value ? [footerData.value] : []));
</script>

<template>
  <section class="grid-shell">
    <header class="activities-toolbar">
      <div class="activities-toolbar__summary">
        <strong>{{ filteredActivities.length }}</strong> / {{ activities.length }} activities
      </div>
      <div class="activities-toolbar__actions">
        <button
          type="button"
          class="btn btn-outline-secondary btn-sm"
          :disabled="!hasActiveFilters"
          @click="resetFilters"
        >
          Reset filters
        </button>
        <button
          type="button"
          class="btn btn-primary btn-sm"
          @click="csvExport"
        >
          Export CSV
        </button>
      </div>
    </header>

    <div class="activities-filters">
      <div class="filter-field filter-field--wide">
        <label
          for="activities-search"
          class="form-label"
        >Search</label>
        <input
          id="activities-search"
          v-model.trim="searchQuery"
          type="text"
          class="form-control form-control-sm"
          placeholder="Search by activity name"
        >
      </div>
      <div class="filter-field">
        <label
          for="activity-type-filter"
          class="form-label"
        >Type</label>
        <select
          id="activity-type-filter"
          v-model="activityTypeFilter"
          class="form-select form-select-sm"
        >
          <option value="ALL">
            All
          </option>
          <option
            v-for="type in availableActivityTypes"
            :key="type"
            :value="type"
          >
            {{ type }}
          </option>
        </select>
      </div>

      <div class="min-filters-row">
        <div class="filter-field">
          <label
            for="distance-min"
            class="form-label"
          >Distance min (km)</label>
          <input
            id="distance-min"
            v-model.number="distanceMinKm"
            type="number"
            min="0"
            step="0.1"
            class="form-control form-control-sm"
          >
        </div>
        <div class="filter-field">
          <label
            for="elevation-min"
            class="form-label"
          >D+ min (m)</label>
          <input
            id="elevation-min"
            v-model.number="elevationMinM"
            type="number"
            min="0"
            step="10"
            class="form-control form-control-sm"
          >
        </div>
        <div class="filter-field">
          <label
            for="duration-min"
            class="form-label"
          >Duration min (min)</label>
          <input
            id="duration-min"
            v-model.number="durationMinMin"
            type="number"
            min="0"
            step="1"
            class="form-control form-control-sm"
          >
        </div>
      </div>
      <div class="filter-toggles">
        <label class="form-check form-check-inline">
          <input
            v-model="commuteOnly"
            class="form-check-input"
            type="checkbox"
          >
          <span class="form-check-label">Commute only</span>
        </label>
        <label class="form-check form-check-inline">
          <input
            v-model="withHeartRate"
            class="form-check-input"
            type="checkbox"
          >
          <span class="form-check-label">With HR</span>
        </label>
        <label class="form-check form-check-inline">
          <input
            v-model="withPower"
            class="form-check-input"
            type="checkbox"
          >
          <span class="form-check-label">With Power</span>
        </label>
      </div>
    </div>

    <div
      v-if="error"
      class="alert alert-danger mb-2"
      role="alert"
    >
      {{ error }}
    </div>

    <div
      v-else-if="isLoading"
      class="activities-state"
    >
      Loading activities...
    </div>

    <div
      v-else-if="filteredActivities.length === 0"
      class="activities-state"
    >
      No activities match the current filters.
    </div>

    <VGrid
      v-else
      name="activitiesGrid"
      theme="material"
      :columns="columns"
      :source="filteredActivities"
      :pinned-bottom-source="pinnedBottomRows"
      :readonly="true"
      style="height: calc(100vh - 300px)"
    />
  </section>
</template>

<style scoped>
.grid-shell {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.activities-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.activities-toolbar__summary {
  color: var(--ms-text-muted);
  font-size: 0.92rem;
}

.activities-toolbar__summary strong {
  color: var(--ms-text);
}

.activities-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.activities-filters {
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  padding: 10px;
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--ms-surface-strong) 92%, white);
}

.filter-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.filter-field--wide {
  grid-column: span 2;
}

.min-filters-row {
  grid-column: 1 / -1;
  display: grid;
  gap: 8px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-label {
  margin: 0;
  font-size: 0.75rem;
  text-transform: uppercase;
  color: var(--ms-text-muted);
  letter-spacing: 0.02em;
  font-weight: 700;
}

.filter-toggles {
  grid-column: 1 / -1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
}

.activities-state {
  border: 1px dashed var(--ms-border);
  border-radius: 10px;
  padding: 14px;
  color: var(--ms-text-muted);
  background: var(--ms-surface-strong);
  font-size: 0.92rem;
}

@media (max-width: 1200px) {
  .activities-filters {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .activities-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .filter-field--wide {
    grid-column: span 2;
  }
}

@media (max-width: 720px) {
  .min-filters-row {
    grid-template-columns: 1fr;
  }
}
</style>

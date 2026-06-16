<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import {
  type EddingtonBasis,
  type EddingtonMetric,
  type EddingtonScope,
  useDashboardStore,
} from "@/stores/dashboard";
import { computed, onMounted } from "vue";
import type { AnnualGoalTargets } from "@/models/annual-goals.model";
import TooltipHint from "@/components/TooltipHint.vue";
import { getMetricTooltip } from "@/utils/metric-tooltips";
import AnnualGoalsPanel from "@/components/AnnualGoalsPanel.vue";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDataPerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
import DistanceElevationDetailsPerYearChart from "@/components/charts/DistanceElevationDetailsPerYearChart.vue";
import HeartRatePerYearChart from "@/components/charts/HeartRatePerYearChart.vue";
import PowerPerYearChart from "@/components/charts/PowerPerYearChart.vue";
import ActivitiesCountPerYearChart from "@/components/charts/ActivitiesCountPerYearChart.vue";
import DistanceElevationPerYearChart from "@/components/charts/DistanceElevationPerYearChart.vue";
import ActiveDaysConsistencyPerYearChart from "@/components/charts/ActiveDaysConsistencyPerYearChart.vue";
import MovingTimePerYearChart from "@/components/charts/MovingTimePerYearChart.vue";
import { formatActivityTypeLabel } from "@/utils/formatters";

const contextStore = useContextStore();
const dashboardStore = useDashboardStore();
onMounted(() => contextStore.updateCurrentView("dashboard"));

const currentActivityType = computed(() => contextStore.currentActivityType);
const currentYear = computed(() => contextStore.currentYear);
const eddingtonScope = computed(() => dashboardStore.eddingtonScope);
const eddingtonMetric = computed(() => dashboardStore.eddingtonMetric);
const eddingtonBasis = computed(() => dashboardStore.eddingtonBasis);
const currentActivityTypeLabel = computed(() =>
  currentActivityType.value
    .split("_")
    .map((activityType) => formatActivityTypeLabel(activityType))
    .join(", ")
);
const isLoading = computed(() => dashboardStore.isLoading);
const error = computed(() => dashboardStore.error);
const annualGoals = computed(() => dashboardStore.annualGoals);
const annualGoalsError = computed(() => dashboardStore.annualGoalsError);
const isSavingAnnualGoals = computed(() => dashboardStore.isSavingAnnualGoals);
const cumulativeDistancePerYear = computed(() => dashboardStore.cumulativeDistancePerYear);
const cumulativeElevationPerYear = computed(
  () => dashboardStore.cumulativeElevationPerYear
);
const eddingtonNumber = computed(() => dashboardStore.eddingtonNumber);
const activitiesCount = computed(() => dashboardStore.dashboardData.nbActivitiesByYear);
const activeDaysByYear = computed(() => dashboardStore.dashboardData.activeDaysByYear);
const consistencyByYear = computed(() => dashboardStore.dashboardData.consistencyByYear);
const movingTimeByYear = computed(() => dashboardStore.dashboardData.movingTimeByYear);
const averageSpeedByYear = computed(() => dashboardStore.dashboardData.averageSpeedByYear);
const maxSpeedByYear = computed(() => dashboardStore.dashboardData.maxSpeedByYear);
const totalDistanceByYear = computed(
  () => dashboardStore.dashboardData.totalDistanceByYear
);
const averageDistanceByYear = computed(
  () => dashboardStore.dashboardData.averageDistanceByYear
);
const maxDistanceByYear = computed(() => dashboardStore.dashboardData.maxDistanceByYear);
const totalElevationByYear = computed(
  () => dashboardStore.dashboardData.totalElevationByYear
);
const averageElevationByYear = computed(
  () => dashboardStore.dashboardData.averageElevationByYear
);
const maxElevationByYear = computed(() => dashboardStore.dashboardData.maxElevationByYear);
const averageHeartRateByYear = computed(
  () => dashboardStore.dashboardData.averageHeartRateByYear
);
const maxHeartRateByYear = computed(() => dashboardStore.dashboardData.maxHeartRateByYear);
const averageWattsByYear = computed(() => dashboardStore.dashboardData.averageWattsByYear);
const maxWattsByYear = computed(() =>
  sortDataByYear(dashboardStore.dashboardData.maxWattsByYear)
);
const deviceAverageWattsByYear = computed(() =>
  sortDataByYear(dashboardStore.dashboardData.deviceAverageWattsByYear ?? {})
);
const deviceMaxWattsByYear = computed(() =>
  sortDataByYear(dashboardStore.dashboardData.deviceMaxWattsByYear ?? {})
);
const eddingtonScopeOptions: Array<{ value: EddingtonScope; label: string }> = [
  { value: "lifetime", label: "Lifetime" },
  { value: "year", label: "Year" },
  { value: "rolling-12-months", label: "12 months" },
];
const eddingtonMetricOptions: Array<{ value: EddingtonMetric; label: string }> = [
  { value: "distance", label: "Distance" },
  { value: "elevation", label: "Elevation" },
];
const eddingtonBasisOptions: Array<{ value: EddingtonBasis; label: string }> = [
  { value: "days", label: "Days" },
  { value: "activities", label: "Activities" },
];
const isYearScopeDisabled = computed(() => currentYear.value === "All years");
const eddingtonTitle = computed(() => {
  const scopeLabel = eddingtonScope.value === "year"
    ? currentYear.value
    : eddingtonScope.value === "rolling-12-months"
      ? "Rolling 12-month"
      : "Lifetime";
  const metricLabel = eddingtonMetric.value === "elevation" ? "Elevation" : "Distance";
  const basisLabel = eddingtonBasis.value === "activities" ? "activities" : "days";
  return `${scopeLabel} ${metricLabel} Eddington by ${basisLabel}: ${eddingtonNumber.value.eddingtonNumber}`;
});

function sortDataByYear(
  averageWattsByYear: Record<string, number>
): Record<string, number> {
  return Object.keys(averageWattsByYear)
    .sort((a, b) => parseInt(a) - parseInt(b))
    .reduce((acc, key) => {
      acc[key] = averageWattsByYear[key] ?? 0;
      return acc;
    }, {} as Record<string, number>);
}

function tooltip(label: string): string {
  return getMetricTooltip(label) ?? "";
}

async function saveAnnualGoals(targets: AnnualGoalTargets) {
  await dashboardStore.saveAnnualGoals(targets);
}

async function setEddingtonScope(scope: EddingtonScope) {
  if (scope === "year" && isYearScopeDisabled.value) {
    return;
  }
  await dashboardStore.setEddingtonScope(scope);
}

async function setEddingtonMetric(metric: EddingtonMetric) {
  await dashboardStore.setEddingtonMetric(metric);
}

async function setEddingtonBasis(basis: EddingtonBasis) {
  await dashboardStore.setEddingtonBasis(basis);
}
</script>

<template>
  <div
    v-if="isLoading"
    class="chart-empty"
  >
    Loading dashboard data...
  </div>
  <div
    v-else-if="error"
    class="chart-empty chart-empty--error"
  >
    {{ error }}
  </div>
  <div
    v-else
    class="chart-stack"
  >
    <AnnualGoalsPanel
      :annual-goals="annualGoals"
      :selected-year="currentYear"
      :activity-type="currentActivityType"
      :saving="isSavingAnnualGoals"
      :error="annualGoalsError"
      @save="saveAnnualGoals"
    />
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Cumulative distance / elevation
        </h3>
        <TooltipHint :text="tooltip('Total distance')" />
      </div>
      <CumulativeDistancePerYearChart
        :activity-type-label="currentActivityTypeLabel"
        :cumulative-distance-per-year="cumulativeDistancePerYear"
        :cumulative-elevation-per-year="cumulativeElevationPerYear"
      />
    </section>

    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Eddington number
        </h3>
        <TooltipHint :text="tooltip('Eddington number')" />
        <div
          class="eddington-controls"
        >
          <div
            class="scope-switch"
            role="group"
            aria-label="Eddington metric"
          >
            <button
              v-for="option in eddingtonMetricOptions"
              :key="option.value"
              type="button"
              class="scope-switch__button"
              :class="{ 'scope-switch__button--active': eddingtonMetric === option.value }"
              @click="setEddingtonMetric(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <div
            class="scope-switch"
            role="group"
            aria-label="Eddington basis"
          >
            <button
              v-for="option in eddingtonBasisOptions"
              :key="option.value"
              type="button"
              class="scope-switch__button"
              :class="{ 'scope-switch__button--active': eddingtonBasis === option.value }"
              @click="setEddingtonBasis(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
          <div
            class="scope-switch"
            role="group"
            aria-label="Eddington scope"
          >
            <button
              v-for="option in eddingtonScopeOptions"
              :key="option.value"
              type="button"
              class="scope-switch__button"
              :class="{ 'scope-switch__button--active': eddingtonScope === option.value }"
              :disabled="option.value === 'year' && isYearScopeDisabled"
              @click="setEddingtonScope(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </div>
      <EddingtonNumberChart
        :title="eddingtonTitle"
        :eddington-number="eddingtonNumber"
      />
    </section>

    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Activities count
        </h3>
        <TooltipHint :text="tooltip('Nb activities')" />
      </div>
      <ActivitiesCountPerYearChart :activities-count="activitiesCount" />
    </section>

    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Distance / elevation by year
        </h3>
        <TooltipHint :text="tooltip('Total elevation')" />
      </div>
      <DistanceElevationPerYearChart
        :distance-by-year="totalDistanceByYear"
        :elevation-by-year="totalElevationByYear"
      />
    </section>
    
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Active days / consistency
        </h3>
        <TooltipHint :text="tooltip('Heatmap Consistency')" />
      </div>
      <ActiveDaysConsistencyPerYearChart
        :active-days-by-year="activeDaysByYear"
        :consistency-by-year="consistencyByYear"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Distance / elevation
        </h3>
        <TooltipHint :text="tooltip('Max distance')" />
      </div>
      <DistanceElevationDetailsPerYearChart
        :average-distance-by-year="averageDistanceByYear"
        :average-elevation-by-year="averageElevationByYear"
        :max-distance-by-year="maxDistanceByYear"
        :max-elevation-by-year="maxElevationByYear"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Moving time
        </h3>
        <TooltipHint :text="tooltip('Moving time by year')" />
      </div>
      <MovingTimePerYearChart :moving-time-by-year="movingTimeByYear" />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Heart rate
        </h3>
        <TooltipHint :text="tooltip('Average Heartrate')" />
      </div>
      <HeartRatePerYearChart
        :average-heart-rate-by-year="averageHeartRateByYear"
        :max-heart-rate-by-year="maxHeartRateByYear"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Speed
        </h3>
        <TooltipHint :text="tooltip('Average Speed')" />
      </div>
      <SpeedPerYearChart
        :activity-type="currentActivityType"
        :average-speed-by-year="averageSpeedByYear"
        :max-speed-by-year="maxSpeedByYear"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Power
        </h3>
        <TooltipHint :text="tooltip('Average Watts')" />
      </div>
      <PowerPerYearChart
        :average-watts-by-year="averageWattsByYear"
        :max-watts-by-year="maxWattsByYear"
        :device-average-watts-by-year="deviceAverageWattsByYear"
        :device-max-watts-by-year="deviceMaxWattsByYear"
      />
    </section>
  </div>
</template>

<style scoped>
.chart-empty--error {
  border-style: solid;
  border-color: #f1b6bf;
  color: #8f2438;
  background: #fff0f3;
}

.chart-panel__header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.chart-panel__title {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 800;
}

.eddington-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-left: auto;
}

.scope-switch {
  display: inline-flex;
  overflow: hidden;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #f8f9fc;
}

.scope-switch__button {
  border: 0;
  border-right: 1px solid var(--ms-border);
  padding: 5px 10px;
  color: var(--ms-text-muted);
  background: transparent;
  font-size: 0.78rem;
  font-weight: 700;
}

.scope-switch__button:last-child {
  border-right: 0;
}

.scope-switch__button:disabled {
  color: #b7bbc6;
  cursor: not-allowed;
}

.scope-switch__button--active {
  color: #ffffff;
  background: var(--ms-primary);
}

@media (max-width: 640px) {
  .chart-panel__header {
    flex-wrap: wrap;
  }

  .eddington-controls {
    width: 100%;
    margin-left: 0;
  }

  .scope-switch {
    flex: 1 1 100%;
  }

  .scope-switch__button {
    flex: 1;
  }
}
</style>

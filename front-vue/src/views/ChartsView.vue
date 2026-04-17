<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useChartsStore } from "@/stores/charts";
import Tooltip from "bootstrap/js/dist/tooltip";
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import ByMonthsChart from "@/components/charts/ByMonthsChart.vue";
import ByWeeksChart from "@/components/charts/ByWeeksChart.vue";
import AverageSpeedByMonthsChart from "@/components/charts/AverageSpeedByMonthsChart.vue";
import ActivitiesCountPerYearChart from "@/components/charts/ActivitiesCountPerYearChart.vue";
import DistanceElevationPerYearChart from "@/components/charts/DistanceElevationPerYearChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
import WeeklyTrainingLoadChart from "@/components/charts/WeeklyTrainingLoadChart.vue";
import DistanceDistributionHistogramChart from "@/components/charts/DistanceDistributionHistogramChart.vue";
import LongRideProgressionChart from "@/components/charts/LongRideProgressionChart.vue";
import EasyHardRatioByMonthChart from "@/components/charts/EasyHardRatioByMonthChart.vue";
import WeeklyConsistencyChart from "@/components/charts/WeeklyConsistencyChart.vue";
import { getMetricTooltip } from "@/utils/metric-tooltips";

const contextStore = useContextStore();
const chartsStore = useChartsStore();

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivityType);
const distanceByMonths = computed(() => chartsStore.distanceByMonths);
const elevationByMonths = computed(() => chartsStore.elevationByMonths);
const averageSpeedByMonths = computed(() => chartsStore.averageSpeedByMonths); 
const distanceByWeeks = computed(() => chartsStore.distanceByWeeks);
const elevationByWeeks = computed(() => chartsStore.elevationByWeeks);
const cadenceByWeeks = computed(() => chartsStore.cadenceByWeeks);
const activitiesCountByYear = computed(() => chartsStore.activitiesCountByYear);
const totalDistanceByYear = computed(() => chartsStore.totalDistanceByYear);
const totalElevationByYear = computed(() => chartsStore.totalElevationByYear);
const averageSpeedByYear = computed(() => chartsStore.averageSpeedByYear);
const maxSpeedByYear = computed(() => chartsStore.maxSpeedByYear);
const activitiesForCharts = computed(() => chartsStore.activitiesForCharts);
const heartRateZoneAnalysis = computed(() => chartsStore.heartRateZoneAnalysis);
const heartRateByMonth = computed(() => heartRateZoneAnalysis.value.byMonth ?? []);
const heartRateByActivity = computed(() => heartRateZoneAnalysis.value.activities ?? []);
const hasHeartRateData = computed(() => heartRateZoneAnalysis.value.hasHeartRateData ?? false);
const isLoading = computed(() => chartsStore.isLoading);
const error = computed(() => chartsStore.error);
const isAllYears = computed(() => currentYear.value === "All years");
const selectedGranularity = ref<"MONTHS" | "WEEKS">("MONTHS");
const isRefreshing = ref(false);
const chartsViewRoot = ref<HTMLElement | null>(null);

let tooltipInstances: Tooltip[] = [];


const cadenceUnit = computed(() => {
  if (currentActivity.value.endsWith("Run") ) return "ppm";
  if (currentActivity.value.endsWith("Ride")) return "rpm";
  return null;
});

const chartSubtitle = computed(() => {
  if (isAllYears.value) {
    return "Yearly overview across your full history (activities, totals, and speed trends).";
  }

  if (selectedGranularity.value === "MONTHS") {
    return `Monthly trends for ${currentYear.value}. Use weekly mode for short-term variations.`;
  }

  return `Weekly trends for ${currentYear.value}. Use monthly mode for a smoother season view.`;
});

async function refreshChartsLocally() {
  if (isRefreshing.value) {
    return;
  }
  isRefreshing.value = true;
  try {
    await chartsStore.refreshCharts();
  } finally {
    isRefreshing.value = false;
  }
}

const ytdTooltip = computed(
  () => getMetricTooltip("YTD average") ?? "Year-To-Date average for the selected period.",
);
const granularityTooltip = computed(
  () => getMetricTooltip("Charts granularity") ?? "Monthly or weekly chart grouping.",
);
const refreshTooltip = computed(
  () => getMetricTooltip("Charts refresh") ?? "Reload chart data for current filters.",
);
const distanceTooltip = computed(
  () => getMetricTooltip("Total distance") ?? "Total distance aggregated by the selected chart period.",
);
const elevationTooltip = computed(
  () => getMetricTooltip("Total elevation") ?? "Total elevation gain aggregated by the selected chart period.",
);
const averageSpeedTooltip = computed(
  () => getMetricTooltip("Average Speed") ?? "Average speed aggregated by period (month or week).",
);
const cadenceTooltip = computed(
  () => getMetricTooltip("Average Cadence") ?? "Average cadence aggregated by period for activities with cadence data.",
);
const activitiesOverviewTooltip = computed(
  () => getMetricTooltip("Nb activities") ?? "Number of activities by year for the selected activity types.",
);
const distanceElevationOverviewTooltip = computed(
  () => "Combined yearly totals for distance and elevation gain.",
);
const speedOverviewTooltip = computed(
  () => "Yearly speed overview with average and maximum speed trends.",
);
const trainingLoadTooltip = computed(
  () => getMetricTooltip("Weekly training load (TRIMP)") ??
    "Simplified TRIMP by week, computed from time spent in each heart-rate zone.",
);
const distanceDistributionTooltip = computed(
  () => getMetricTooltip("Distance distribution") ?? "Histogram of ride distances to show short vs medium vs long tendencies.",
);
const longRideProgressionTooltip = computed(
  () => getMetricTooltip("Long ride progression") ?? "Weekly max long ride distance with a 4-week moving average.",
);
const easyHardByMonthTooltip = computed(
  () => getMetricTooltip("Easy / Hard ratio by month") ?? "Monthly easy vs hard HR-zone split, including easy/hard ratio trend.",
);
const weeklyConsistencyTooltip = computed(
  () => getMetricTooltip("Weekly consistency") ?? "Number of active weeks vs inactive weeks in the selected year.",
);

type YearMetricComparison = {
  current: number | null;
  previous: number | null;
  delta: number | null;
  deltaPct: number | null;
};

function toNullableNumber(value: number | undefined): number | null {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return null;
  }
  return value;
}

function computeYearMetricComparison(currentValue: number | undefined, previousValue: number | undefined): YearMetricComparison {
  const current = toNullableNumber(currentValue);
  const previous = toNullableNumber(previousValue);

  if (current === null || previous === null) {
    return {
      current,
      previous,
      delta: null,
      deltaPct: null,
    };
  }

  const delta = current - previous;
  const deltaPct = previous === 0 ? null : (delta / previous) * 100;

  return {
    current,
    previous,
    delta,
    deltaPct,
  };
}

const yearComparison = computed(() => {
  if (isAllYears.value) {
    return null;
  }

  const currentYearValue = Number.parseInt(currentYear.value, 10);
  if (Number.isNaN(currentYearValue)) {
    return null;
  }

  const currentKey = String(currentYearValue);
  const previousKey = String(currentYearValue - 1);

  const comparison = {
    currentYear: currentKey,
    previousYear: previousKey,
    distance: computeYearMetricComparison(totalDistanceByYear.value[currentKey], totalDistanceByYear.value[previousKey]),
    elevation: computeYearMetricComparison(totalElevationByYear.value[currentKey], totalElevationByYear.value[previousKey]),
    averageSpeed: computeYearMetricComparison(averageSpeedByYear.value[currentKey], averageSpeedByYear.value[previousKey]),
    activities: computeYearMetricComparison(activitiesCountByYear.value[currentKey], activitiesCountByYear.value[previousKey]),
  };

  const hasData = [
    comparison.distance,
    comparison.elevation,
    comparison.averageSpeed,
    comparison.activities,
  ].some((metric) => metric.current !== null || metric.previous !== null);

  return hasData ? comparison : null;
});

const comparisonRows = computed(() => {
  const comparison = yearComparison.value;
  if (!comparison) {
    return [];
  }

  return [
    { label: "Distance", unit: "km", digits: 1, metric: comparison.distance },
    { label: "Elevation", unit: "m", digits: 0, metric: comparison.elevation },
    { label: "Average speed", unit: "km/h", digits: 1, metric: comparison.averageSpeed },
    { label: "Activities", unit: "", digits: 0, metric: comparison.activities },
  ];
});

function formatMetricValue(value: number | null, unit: string, digits: number): string {
  if (value === null) {
    return "N/A";
  }
  const formatted = value.toFixed(digits);
  return unit ? `${formatted} ${unit}` : formatted;
}

function formatDeltaValue(value: number | null, unit: string, digits: number): string {
  if (value === null) {
    return "N/A";
  }
  const sign = value > 0 ? "+" : "";
  const formatted = `${sign}${value.toFixed(digits)}`;
  return unit ? `${formatted} ${unit}` : formatted;
}

function formatDeltaPct(value: number | null): string {
  if (value === null) {
    return "N/A";
  }
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(1)}%`;
}

function deltaClass(value: number | null): string {
  if (value === null || value === 0) {
    return "chart-comparison__delta--neutral";
  }
  return value > 0 ? "chart-comparison__delta--positive" : "chart-comparison__delta--negative";
}

function disposeBootstrapTooltips(): void {
  tooltipInstances.forEach((instance) => instance.dispose());
  tooltipInstances = [];
}

async function initBootstrapTooltips(): Promise<void> {
  disposeBootstrapTooltips();
  await nextTick();

  if (!chartsViewRoot.value) {
    return;
  }

  const elements = chartsViewRoot.value.querySelectorAll<HTMLElement>('[data-bs-toggle="tooltip"]');
  tooltipInstances = Array.from(elements).map((element) => new Tooltip(element));
}

onMounted(() => {
  contextStore.updateCurrentView("charts");
  void initBootstrapTooltips();
});

onBeforeUnmount(() => {
  disposeBootstrapTooltips();
});

watch([selectedGranularity, isAllYears, isLoading, error, isRefreshing], () => {
  void initBootstrapTooltips();
});

watch([ytdTooltip, granularityTooltip, refreshTooltip], () => {
  void initBootstrapTooltips();
});

watch(
  [
    distanceTooltip,
    elevationTooltip,
    averageSpeedTooltip,
    cadenceTooltip,
    activitiesOverviewTooltip,
    distanceElevationOverviewTooltip,
    speedOverviewTooltip,
    trainingLoadTooltip,
    distanceDistributionTooltip,
    longRideProgressionTooltip,
    easyHardByMonthTooltip,
    weeklyConsistencyTooltip,
  ],
  () => {
    void initBootstrapTooltips();
  },
);

</script>

<template>
  <div
    ref="chartsViewRoot"
    class="charts-view"
  >
    <section class="chart-toolbar">
      <div class="chart-toolbar__left">
        <h2 class="chart-toolbar__title">
          Charts
        </h2>
        <p class="chart-toolbar__subtitle">
          {{ chartSubtitle }}
        </p>
        <div
          v-if="!isAllYears"
          class="chart-toolbar__hints"
        >
          <span class="chart-toolbar__hint">
            YTD average
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="ytdTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Granularity
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="granularityTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Distance
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="distanceTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Elevation
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="elevationTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Avg speed
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="averageSpeedTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Cadence
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="cadenceTooltip"
            >?</span>
          </span>
        </div>
        <div
          v-else
          class="chart-toolbar__hints"
        >
          <span class="chart-toolbar__hint">
            Activities
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="activitiesOverviewTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Distance + Elevation
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="distanceElevationOverviewTooltip"
            >?</span>
          </span>
          <span class="chart-toolbar__hint">
            Speed
            <span
              class="chart-toolbar__hint-icon"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="speedOverviewTooltip"
            >?</span>
          </span>
        </div>
      </div>
      <div class="chart-toolbar__actions">
        <div
          v-if="!isAllYears"
          class="btn-group btn-group-sm"
          role="group"
          aria-label="Chart granularity"
        >
          <button
            type="button"
            class="btn"
            :class="selectedGranularity === 'MONTHS' ? 'btn-primary' : 'btn-outline-secondary'"
            @click="selectedGranularity = 'MONTHS'"
          >
            Monthly
          </button>
          <button
            type="button"
            class="btn"
            :class="selectedGranularity === 'WEEKS' ? 'btn-primary' : 'btn-outline-secondary'"
            @click="selectedGranularity = 'WEEKS'"
          >
            Weekly
          </button>
        </div>
        <button
          type="button"
          class="btn btn-outline-secondary btn-sm"
          :disabled="isLoading || isRefreshing"
          data-bs-toggle="tooltip"
          data-bs-placement="left"
          :data-bs-title="refreshTooltip"
          @click="refreshChartsLocally"
        >
          {{ isRefreshing ? "Refreshing..." : "Refresh" }}
        </button>
      </div>
    </section>

    <div
      v-if="isLoading"
      class="chart-empty"
    >
      Loading chart data...
    </div>

    <div
      v-else-if="error"
      class="chart-empty chart-empty--error"
    >
      {{ error }}
    </div>

    <template v-else>
      <section
        v-if="!isAllYears && yearComparison"
        class="chart-comparison"
      >
        <div class="chart-comparison__header">
          <h3 class="chart-comparison__title">
            Year-over-year snapshot
          </h3>
          <span class="chart-comparison__subtitle">
            {{ yearComparison.currentYear }} vs {{ yearComparison.previousYear }}
          </span>
        </div>
        <div class="chart-comparison__grid">
          <article
            v-for="row in comparisonRows"
            :key="row.label"
            class="chart-comparison__card"
          >
            <h4 class="chart-comparison__metric">
              {{ row.label }}
            </h4>
            <p class="chart-comparison__line">
              {{ yearComparison.currentYear }}:
              <strong>{{ formatMetricValue(row.metric.current, row.unit, row.digits) }}</strong>
            </p>
            <p class="chart-comparison__line">
              {{ yearComparison.previousYear }}:
              <strong>{{ formatMetricValue(row.metric.previous, row.unit, row.digits) }}</strong>
            </p>
            <p
              :class="['chart-comparison__delta', deltaClass(row.metric.delta)]"
            >
              Δ {{ formatDeltaValue(row.metric.delta, row.unit, row.digits) }}
              <span class="chart-comparison__delta-pct">({{ formatDeltaPct(row.metric.deltaPct) }})</span>
            </p>
          </article>
        </div>
      </section>

      <div
        v-if="!isAllYears"
        class="chart-stack"
      >
        <template v-if="selectedGranularity === 'MONTHS'">
          <section class="chart-panel">
            <ByMonthsChart
              title="Distance"
              unit="km"
              :data-by-months="distanceByMonths"
              :selected-year="currentYear"
            />
          </section>
          <section class="chart-panel">
            <ByMonthsChart
              title="Elevation"
              unit="m"
              :data-by-months="elevationByMonths"
              :selected-year="currentYear"
            />
          </section>
          <section class="chart-panel">
            <AverageSpeedByMonthsChart
              :activity-type="currentActivity"
              :data-by-months="averageSpeedByMonths"
            />
          </section>
        </template>

        <template v-else>
          <section class="chart-panel">
            <ByWeeksChart
              title="Distances"
              unit="km"
              :items-by-weeks="distanceByWeeks"
              :selected-year="currentYear"
            />
          </section>
          <section class="chart-panel">
            <ByWeeksChart
              title="Elevation"
              unit="m"
              :items-by-weeks="elevationByWeeks"
              :selected-year="currentYear"
            />
          </section>
        </template>
        <section
          v-if="selectedGranularity === 'WEEKS' && cadenceUnit"
          class="chart-panel"
        >
          <ByWeeksChart
            title="Cadence"
            :unit="cadenceUnit"
            :items-by-weeks="cadenceByWeeks"
            :selected-year="currentYear"
          />
        </section>

        <section class="chart-panel">
          <div class="chart-panel__header">
            <h3 class="chart-panel__title">
              Weekly Training Load
            </h3>
            <span
              class="chart-panel__hint"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="trainingLoadTooltip"
            >?</span>
          </div>
          <WeeklyTrainingLoadChart
            v-if="hasHeartRateData"
            :activity-summaries="heartRateByActivity"
            :selected-year="currentYear"
          />
          <div
            v-else
            class="chart-empty"
          >
            No heart-rate stream data available for training load analysis.
          </div>
        </section>

        <section class="chart-panel">
          <div class="chart-panel__header">
            <h3 class="chart-panel__title">
              Distance Distribution
            </h3>
            <span
              class="chart-panel__hint"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="distanceDistributionTooltip"
            >?</span>
          </div>
          <DistanceDistributionHistogramChart :activities="activitiesForCharts" />
        </section>

        <section class="chart-panel">
          <div class="chart-panel__header">
            <h3 class="chart-panel__title">
              Long Ride Progression
            </h3>
            <span
              class="chart-panel__hint"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="longRideProgressionTooltip"
            >?</span>
          </div>
          <LongRideProgressionChart
            :activities="activitiesForCharts"
            :selected-year="currentYear"
          />
        </section>

        <section class="chart-panel">
          <div class="chart-panel__header">
            <h3 class="chart-panel__title">
              Easy / Hard Ratio By Month
            </h3>
            <span
              class="chart-panel__hint"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="easyHardByMonthTooltip"
            >?</span>
          </div>
          <EasyHardRatioByMonthChart
            v-if="hasHeartRateData"
            :by-month="heartRateByMonth"
            :selected-year="currentYear"
          />
          <div
            v-else
            class="chart-empty"
          >
            No heart-rate stream data available for easy/hard monthly ratio.
          </div>
        </section>

        <section class="chart-panel">
          <div class="chart-panel__header">
            <h3 class="chart-panel__title">
              Weekly Consistency
            </h3>
            <span
              class="chart-panel__hint"
              tabindex="0"
              role="button"
              data-bs-toggle="tooltip"
              data-bs-placement="top"
              :data-bs-title="weeklyConsistencyTooltip"
            >?</span>
          </div>
          <WeeklyConsistencyChart
            :distance-by-weeks="distanceByWeeks"
            :selected-year="currentYear"
          />
        </section>
      </div>

      <div
        v-else
        class="chart-stack"
      >
        <section class="chart-panel">
          <ActivitiesCountPerYearChart :activities-count="activitiesCountByYear" />
        </section>
        <section class="chart-panel">
          <DistanceElevationPerYearChart
            :distance-by-year="totalDistanceByYear"
            :elevation-by-year="totalElevationByYear"
          />
        </section>
        <section class="chart-panel">
          <SpeedPerYearChart
            :activity-type="currentActivity"
            :average-speed-by-year="averageSpeedByYear"
            :max-speed-by-year="maxSpeedByYear"
          />
        </section>
      </div>
    </template>
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

.chart-panel__hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1rem;
  height: 1rem;
  border: 1px solid #ffc9b2;
  border-radius: 999px;
  color: #d44700;
  background: #fff6f1;
  font-size: 0.72rem;
  font-weight: 700;
  line-height: 1;
  cursor: help;
  user-select: none;
}

.chart-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 10px 12px;
  margin-bottom: 14px;
}

.chart-toolbar__left {
  min-width: 0;
}

.chart-toolbar__title {
  margin: 0;
  font-size: 1.02rem;
  font-weight: 800;
}

.chart-toolbar__subtitle {
  margin: 2px 0 0;
  font-size: 0.88rem;
  color: var(--ms-text-muted);
}

.chart-toolbar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.chart-toolbar__hints {
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.chart-toolbar__hint {
  display: inline-flex;
  align-items: center;
  font-size: 0.8rem;
  color: var(--ms-text-muted);
}

.chart-toolbar__hint-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1rem;
  height: 1rem;
  margin-left: 0.3rem;
  border: 1px solid #ffc9b2;
  border-radius: 999px;
  color: #d44700;
  background: #fff6f1;
  font-size: 0.72rem;
  font-weight: 700;
  line-height: 1;
  cursor: help;
  user-select: none;
}

.chart-comparison {
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 12px;
  margin-bottom: 14px;
}

.chart-comparison__header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.chart-comparison__title {
  margin: 0;
  font-size: 0.96rem;
  font-weight: 800;
}

.chart-comparison__subtitle {
  font-size: 0.82rem;
  color: var(--ms-text-muted);
}

.chart-comparison__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.chart-comparison__card {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  background: #fcfdff;
  padding: 8px 10px;
}

.chart-comparison__metric {
  margin: 0 0 6px;
  font-size: 0.88rem;
  font-weight: 700;
}

.chart-comparison__line {
  margin: 0;
  font-size: 0.8rem;
  color: var(--ms-text-muted);
}

.chart-comparison__delta {
  margin: 6px 0 0;
  font-size: 0.82rem;
  font-weight: 700;
}

.chart-comparison__delta-pct {
  font-weight: 600;
}

.chart-comparison__delta--positive {
  color: #1f8a43;
}

.chart-comparison__delta--negative {
  color: #b6403a;
}

.chart-comparison__delta--neutral {
  color: var(--ms-text-muted);
}

@media (max-width: 900px) {
  .chart-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .chart-toolbar__actions {
    justify-content: space-between;
  }

  .chart-comparison__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>

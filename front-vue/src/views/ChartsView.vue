<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useChartsStore } from "@/stores/charts";
import { computed, ref } from "vue";
import ByMonthsChart from "@/components/charts/ByMonthsChart.vue";
import ByWeeksChart from "@/components/charts/ByWeeksChart.vue";
import AverageSpeedByMonthsChart from "@/components/charts/AverageSpeedByMonthsChart.vue";
import ActivitiesCountPerYearChart from "@/components/charts/ActivitiesCountPerYearChart.vue";
import DistanceElevationPerYearChart from "@/components/charts/DistanceElevationPerYearChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
import TooltipHint from "@/components/TooltipHint.vue";
import { getMetricTooltip } from "@/utils/metric-tooltips";
import { onMounted } from "vue";

const contextStore = useContextStore();
const chartsStore = useChartsStore();
onMounted(() => contextStore.updateCurrentView("charts"));

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
const isLoading = computed(() => chartsStore.isLoading);
const error = computed(() => chartsStore.error);
const isAllYears = computed(() => currentYear.value === "All years");
const selectedGranularity = ref<"MONTHS" | "WEEKS">("MONTHS");
const isRefreshing = ref(false);


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

</script>

<template>
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
          <TooltipHint :text="ytdTooltip" />
        </span>
        <span class="chart-toolbar__hint">
          Granularity
          <TooltipHint :text="granularityTooltip" />
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
        :title="refreshTooltip"
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

  <div
    v-else-if="!isAllYears"
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

<style scoped>
.chart-empty--error {
  border-style: solid;
  border-color: #f1b6bf;
  color: #8f2438;
  background: #fff0f3;
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

@media (max-width: 900px) {
  .chart-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .chart-toolbar__actions {
    justify-content: space-between;
  }
}
</style>

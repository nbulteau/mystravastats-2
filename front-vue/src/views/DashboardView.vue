<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import TooltipHint from "@/components/TooltipHint.vue";
import { getMetricTooltip } from "@/utils/metric-tooltips";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDataPerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
import DistancePerYearChart from "@/components/charts/DistancePerYearChart.vue";
import ElevationPerYearChart from "@/components/charts/ElevationPerYearChart.vue";
import HeartRatePerYearChart from "@/components/charts/HeartRatePerYearChart.vue";
import PowerPerYearChart from "@/components/charts/PowerPerYearChart.vue";
import ActivitiesCountPerYearChart from "@/components/charts/ActivitiesCountPerYearChart.vue";
import DistanceElevationPerYearChart from "@/components/charts/DistanceElevationPerYearChart.vue";
import ActivityHeatmapChart from "@/components/charts/ActivityHeatmapChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("dashboard");

const currentActivityType = computed(() => contextStore.currentActivityType);
const cumulativeDistancePerYear = computed(() => contextStore.cumulativeDistancePerYear);
const cumulativeElevationPerYear = computed(
  () => contextStore.cumulativeElevationPerYear
);
const eddingtonNumber = computed(() => contextStore.eddingtonNumber);
const activitiesCount = computed(() => contextStore.dashboardData.nbActivitiesByYear);
const averageSpeedByYear = computed(() => contextStore.dashboardData.averageSpeedByYear);
const maxSpeedByYear = computed(() => contextStore.dashboardData.maxSpeedByYear);
const totalDistanceByYear = computed(
  () => contextStore.dashboardData.totalDistanceByYear
);
const averageDistanceByYear = computed(
  () => contextStore.dashboardData.averageDistanceByYear
);
const maxDistanceByYear = computed(() => contextStore.dashboardData.maxDistanceByYear);
const totalElevationByYear = computed(
  () => contextStore.dashboardData.totalElevationByYear
);
const averageElevationByYear = computed(
  () => contextStore.dashboardData.averageElevationByYear
);
const maxElevationByYear = computed(() => contextStore.dashboardData.maxElevationByYear);
const averageHeartRateByYear = computed(
  () => contextStore.dashboardData.averageHeartRateByYear
);
const maxHeartRateByYear = computed(() => contextStore.dashboardData.maxHeartRateByYear);
const averageWattsByYear = computed(() => contextStore.dashboardData.averageWattsByYear);
const maxWattsByYear = computed(() =>
  sortDataByYear(contextStore.dashboardData.maxWattsByYear)
);
const activityHeatmap = computed(() => contextStore.activityHeatmap);

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
</script>

<template>
  <div class="chart-stack">
    <section class="chart-panel">
      <ActivitiesCountPerYearChart :activities-count="activitiesCount" />
    </section>
    <section class="chart-panel">
      <div class="chart-help">
        Eddington number
        <TooltipHint :text="getMetricTooltip('Eddington number') ?? ''" />
      </div>
      <EddingtonNumberChart
        :title="`Eddington number for ${currentActivityType}: ${eddingtonNumber.eddingtonNumber}`"
        :eddington-number="eddingtonNumber"
      />
    </section>
    <section class="chart-panel">
      <CumulativeDistancePerYearChart
        :cumulative-distance-per-year="cumulativeDistancePerYear"
        :cumulative-elevation-per-year="cumulativeElevationPerYear"
      />
    </section>
    <section class="chart-panel">
      <DistanceElevationPerYearChart
        :distance-by-year="totalDistanceByYear"
        :elevation-by-year="totalElevationByYear"
      />
    </section>
    <section class="chart-panel">
      <DistancePerYearChart
        :average-distance-by-year="averageDistanceByYear"
        :max-distance-by-year="maxDistanceByYear"
      />
    </section>
    <section class="chart-panel">
      <ElevationPerYearChart
        :average-elevation-by-year="averageElevationByYear"
        :max-elevation-by-year="maxElevationByYear"
      />
    </section>
    <section class="chart-panel">
      <HeartRatePerYearChart
        :average-heart-rate-by-year="averageHeartRateByYear"
        :max-heart-rate-by-year="maxHeartRateByYear"
      />
    </section>
    <section class="chart-panel">
      <SpeedPerYearChart
        :activity-type="currentActivityType"
        :average-speed-by-year="averageSpeedByYear"
        :max-speed-by-year="maxSpeedByYear"
      />
    </section>
    <section class="chart-panel">
      <PowerPerYearChart
          :average-watts-by-year="averageWattsByYear"
          :max-watts-by-year="maxWattsByYear"
      />
    </section>
    <section class="chart-panel chart-panel--wide">
      <ActivityHeatmapChart :activity-heatmap="activityHeatmap" />
    </section>
  </div>
</template>

<style scoped>
.chart-help {
  display: inline-flex;
  align-items: center;
  color: #4c617b;
  font-size: 0.85rem;
  margin-bottom: 0.35rem;
}

/* Allow the heatmap panel to span the full row */
.chart-panel--wide {
  grid-column: 1 / -1;
}
</style>

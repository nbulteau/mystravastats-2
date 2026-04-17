<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useDashboardStore } from "@/stores/dashboard";
import { computed, onMounted } from "vue";
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
import ActiveDaysConsistencyPerYearChart from "@/components/charts/ActiveDaysConsistencyPerYearChart.vue";
import MovingTimePerYearChart from "@/components/charts/MovingTimePerYearChart.vue";
import ElevationEfficiencyPerYearChart from "@/components/charts/ElevationEfficiencyPerYearChart.vue";

const contextStore = useContextStore();
const dashboardStore = useDashboardStore();
onMounted(() => contextStore.updateCurrentView("dashboard"));

const currentActivityType = computed(() => contextStore.currentActivityType);
const isLoading = computed(() => dashboardStore.isLoading);
const error = computed(() => dashboardStore.error);
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
const elevationEfficiencyByYear = computed(
  () => dashboardStore.dashboardData.elevationEfficiencyByYear
);
const averageHeartRateByYear = computed(
  () => dashboardStore.dashboardData.averageHeartRateByYear
);
const maxHeartRateByYear = computed(() => dashboardStore.dashboardData.maxHeartRateByYear);
const averageWattsByYear = computed(() => dashboardStore.dashboardData.averageWattsByYear);
const maxWattsByYear = computed(() =>
  sortDataByYear(dashboardStore.dashboardData.maxWattsByYear)
);

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
          Eddington number
        </h3>
        <TooltipHint :text="tooltip('Eddington number')" />
      </div>
      <EddingtonNumberChart
        :title="`Eddington number for ${currentActivityType}: ${eddingtonNumber.eddingtonNumber}`"
        :eddington-number="eddingtonNumber"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Cumulative distance / elevation
        </h3>
        <TooltipHint :text="tooltip('Total distance')" />
      </div>
      <CumulativeDistancePerYearChart
        :cumulative-distance-per-year="cumulativeDistancePerYear"
        :cumulative-elevation-per-year="cumulativeElevationPerYear"
      />
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
          Distance
        </h3>
        <TooltipHint :text="tooltip('Max distance')" />
      </div>
      <DistancePerYearChart
        :average-distance-by-year="averageDistanceByYear"
        :max-distance-by-year="maxDistanceByYear"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Elevation
        </h3>
        <TooltipHint :text="tooltip('Max elevation')" />
      </div>
      <ElevationPerYearChart
        :average-elevation-by-year="averageElevationByYear"
        :max-elevation-by-year="maxElevationByYear"
      />
    </section>
    <section class="chart-panel">
      <div class="chart-panel__header">
        <h3 class="chart-panel__title">
          Elevation efficiency
        </h3>
        <TooltipHint :text="tooltip('Elevation efficiency')" />
      </div>
      <ElevationEfficiencyPerYearChart
        :elevation-efficiency-by-year="elevationEfficiencyByYear"
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
</style>

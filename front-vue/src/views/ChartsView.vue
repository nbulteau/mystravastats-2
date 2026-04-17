<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useChartsStore } from "@/stores/charts";
import { computed } from "vue";
import ByMonthsChart from "@/components/charts/ByMonthsChart.vue";
import ByWeeksChart from "@/components/charts/ByWeeksChart.vue";
import AverageSpeedByMonthsChart from "@/components/charts/AverageSpeedByMonthsChart.vue";
import ActivitiesCountPerYearChart from "@/components/charts/ActivitiesCountPerYearChart.vue";
import DistanceElevationPerYearChart from "@/components/charts/DistanceElevationPerYearChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
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


const cadenceUnit = computed(() => {
  if (currentActivity.value.endsWith("Run") ) return "ppm";
  if (currentActivity.value.endsWith("Ride")) return "rpm";
  return null;
});

</script>

<template>
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
        :current-year="currentYear"
        :data-by-months="averageSpeedByMonths"
      />
    </section>
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
    <section
      v-if="cadenceUnit"
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
</style>

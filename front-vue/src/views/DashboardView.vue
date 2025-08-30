<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDataPerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
import DistancePerYearChart from "@/components/charts/DistancePerYearChart.vue";
import ElevationPerYearChart from "@/components/charts/ElevationPerYearChart.vue";
import HeartRatePerYearChart from "@/components/charts/HeartRatePerYearChart.vue";
import PowerPerYearChart from "@/components/charts/PowerPerYearChart.vue";
import ActivitiesCountPerYearChart from "@/components/charts/ActivitiesCountPerYearChart.vue";
import DistanceElevationPerYearChart from "@/components/charts/DistanceElevationPerYearChart.vue";

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
  <ActivitiesCountPerYearChart :activities-count="activitiesCount" />
  <EddingtonNumberChart
    :title="`Eddington number for ${currentActivityType}: ${eddingtonNumber.eddingtonNumber}`"
    :eddington-number="eddingtonNumber"
  />
  <CumulativeDistancePerYearChart
    :cumulative-distance-per-year="cumulativeDistancePerYear"
    :cumulative-elevation-per-year="cumulativeElevationPerYear"
  />
  <DistanceElevationPerYearChart
    :distance-by-year="totalDistanceByYear"
    :elevation-by-year="totalElevationByYear"
  />
  <DistancePerYearChart
    :average-distance-by-year="averageDistanceByYear"
    :max-distance-by-year="maxDistanceByYear"
  />
  <ElevationPerYearChart
    :average-elevation-by-year="averageElevationByYear"
    :max-elevation-by-year="maxElevationByYear"
  />
  <HeartRatePerYearChart
    :average-heart-rate-by-year="averageHeartRateByYear"
    :max-heart-rate-by-year="maxHeartRateByYear"
  />
  <SpeedPerYearChart
    :activity-type="currentActivityType"
    :average-speed-by-year="averageSpeedByYear"
    :max-speed-by-year="maxSpeedByYear"
  />
  <PowerPerYearChart
      :average-watts-by-year="averageWattsByYear"
      :max-watts-by-year="maxWattsByYear"
  />
</template>

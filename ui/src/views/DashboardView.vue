<script setup lang="ts">
import {useContextStore} from "@/stores/context.js";
import {computed} from "vue";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDataPerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";
import DistancePerYearChart from "@/components/charts/DistancePerYearChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("dashboard");

const currentActivity = computed(() => contextStore.currentActivity);
const cumulativeDistancePerYear = computed(() => contextStore.cumulativeDistancePerYear);
const cumulativeElevationPerYear = computed(() => contextStore.cumulativeElevationPerYear);
const eddingtonNumber = computed(() => contextStore.eddingtonNumber);
const averageSpeedByYear = computed(() => contextStore.dashboardData.averageSpeedByYear);
const maxSpeedByYear = computed(() => contextStore.dashboardData.maxSpeedByYear);
const totalDistanceByYear = computed(() => contextStore.dashboardData.totalDistanceByYear);
const averageDistanceByYear = computed(() => contextStore.dashboardData.averageDistanceByYear);
const maxDistanceByYear = computed(() => contextStore.dashboardData.maxDistanceByYear);

</script>

<template>
  <EddingtonNumberChart
      :title="`Eddington number for ${currentActivity}: ${eddingtonNumber.eddingtonNumber}`"
      :eddington-number="eddingtonNumber"
  />
  <CumulativeDistancePerYearChart
      :cumulative-distance-per-year="cumulativeDistancePerYear"
      :cumulative-elevation-per-year="cumulativeElevationPerYear"
  />
  <DistancePerYearChart
      :average-distance-by-year="averageDistanceByYear"
      :max-distance-by-year="maxDistanceByYear"
  />
  <SpeedPerYearChart
      :activity-type="currentActivity"
      :average-speed-by-year="averageSpeedByYear"
      :max-speed-by-year="maxSpeedByYear"
  />

</template>

<script setup lang="ts">
import {useContextStore} from "@/stores/context.js";
import {computed} from "vue";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDataPerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";
import SpeedPerYearChart from "@/components/charts/SpeedPerYearChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("dashboard");

const currentActivity = computed(() => contextStore.currentActivity);
const cumulativeDistancePerYear = computed(() => contextStore.cumulativeDistancePerYear);
const cumulativeElevationPerYear = computed(() => contextStore.cumulativeElevationPerYear);
const eddingtonNumber = computed(() => contextStore.eddingtonNumber);
const averageSpeedByYear = computed(() => contextStore.dashboardData.averageSpeedByYear);
const maxSpeedByYear = computed(() => contextStore.dashboardData.maxSpeedByYear);

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
  <SpeedPerYearChart
      :activity-type="currentActivity"
      :average-speed-by-year="averageSpeedByYear"
      :max-speed-by-year="maxSpeedByYear"
  />

</template>

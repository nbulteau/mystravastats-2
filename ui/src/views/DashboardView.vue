<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDataPerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("dashboard");

const currentActivity = computed(() => contextStore.currentActivity);
const cumulativeDistancePerYear = computed(() => contextStore.cumulativeDistancePerYear);
const cumulativeElevationPerYear = computed(() => contextStore.cumulativeElevationPerYear);
const eddingtonNumber = computed(() => contextStore.eddingtonNumber);
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
</template>

<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import DistanceByMonthsChart from "@/components/charts/DistanceByMonthsChart.vue";
import ElevationByMonthsChart from "@/components/charts/ElevationByMonthsChart.vue";
import DistanceByWeeksChart from "@/components/charts/DistanceByWeeksChart.vue";
import ElevationByWeeksChart from "@/components/charts/ElevationByWeeksChart.vue";
import CumulativeDistancePerYearChart from "@/components/charts/CumulativeDistancePerYearChart.vue";
import EddingtonNumberChart from "@/components/charts/EddingtonNumberChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("charts");

const currentYear = computed(() => contextStore.currentYear);
const distanceByMonths = computed(() => contextStore.distanceByMonths);
const elevationByMonths = computed(() => contextStore.elevationByMonths);
const distanceByWeeks = computed(() => contextStore.distanceByWeeks);
const elevationByWeeks = computed(() => contextStore.elevationByWeeks);
const cumulativeDistancePerYear = computed(() => contextStore.cumulativeDistancePerYear);
const eddingtonNumber = computed(() => contextStore.eddingtonNumber);
</script>

<template>
  <main>
    <EddingtonNumberChart :eddington-number="eddingtonNumber" />
    <CumulativeDistancePerYearChart
      :current-year="currentYear"
      :cumulative-distance-per-year="cumulativeDistancePerYear"
    />
    <div v-if="currentYear !== 'All years'">
      <DistanceByMonthsChart
        :current-year="currentYear"
        :distance-by-months="distanceByMonths"
      />
      <ElevationByMonthsChart
        :current-year="currentYear"
        :elevation-by-months="elevationByMonths"
      />
      <DistanceByWeeksChart
        :current-year="currentYear"
        :distance-by-weeks="distanceByWeeks"
      />
      <ElevationByWeeksChart
        :current-year="currentYear"
        :elevation-by-weeks="elevationByWeeks"
      />
    </div>
  </main>
</template>

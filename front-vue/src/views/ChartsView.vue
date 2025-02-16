<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import ByMonthsChart from "@/components/charts/ByMonthsChart.vue";
import ByWeeksChart from "@/components/charts/ByWeeksChart.vue";
import AverageSpeedByMonthsChart from "@/components/charts/AverageSpeedByMonthsChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("charts");

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivityType);
const distanceByMonths = computed(() => contextStore.distanceByMonths);
const elevationByMonths = computed(() => contextStore.elevationByMonths);
const averageSpeedByMonths = computed(() => contextStore.averageSpeedByMonths); 
const distanceByWeeks = computed(() => contextStore.distanceByWeeks);
const elevationByWeeks = computed(() => contextStore.elevationByWeeks);
</script>

<template>
  <div v-if="currentYear !== 'All years'">
    <ByMonthsChart
      title="Distance"
      unit="km"
      :data-by-months="distanceByMonths"
    />
    <ByMonthsChart
      title="Elevation"
      unit="m"
      :data-by-months="elevationByMonths"
    />
    <AverageSpeedByMonthsChart
      :activity-type="currentActivity"
      :current-year="currentYear"
      :data-by-months="averageSpeedByMonths"
    />
    <ByWeeksChart
      title="Distances"
      unit="km"
      :distance-by-weeks="distanceByWeeks"
    />
    <ByWeeksChart
      title="Elevation"
      unit="m"
      :distance-by-weeks="elevationByWeeks"
    />
  </div>
</template>

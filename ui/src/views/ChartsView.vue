<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import ByMonthsChart from "@/components/charts/ByMonthsChart.vue";
import ByWeeksChart from "@/components/charts/ByWeeksChart.vue";
import AverageSpeedByMonthsChart from "@/components/charts/AverageSpeedByMonthsChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("charts");

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivity);
const distanceByMonths = computed(() => contextStore.distanceByMonths);
const elevationByMonths = computed(() => contextStore.elevationByMonths);
const averageSpeedByMonths = computed(() => contextStore.averageSpeedByMonths); 
const distanceByWeeks = computed(() => contextStore.distanceByWeeks);
const elevationByWeeks = computed(() => contextStore.elevationByWeeks);
</script>

<template>
  <div v-if="currentYear !== 'All years'">
    <ByMonthsChart
      title="Distance by months"
      y-axis-title="Distance (km)"
      unit="km"
      :current-year="currentYear"
      :data-by-months="distanceByMonths"
    />
    <ByMonthsChart
      title="Elevation by months"
      y-axis-title="Elevation (m)"
      unit="m"
      :current-year="currentYear"
      :data-by-months="elevationByMonths"
    />
    <AverageSpeedByMonthsChart
      :activity-type="currentActivity"
      :current-year="currentYear"
      :data-by-months="averageSpeedByMonths"
    />
    <ByWeeksChart
      title="Distance by weeks"
      y-axis-title="Distance (km)"
      unit="km"
      :current-year="currentYear"
      :distance-by-weeks="distanceByWeeks"
    />
    <ByWeeksChart
      title="Elevation by weeks"
      y-axis-title="Elevation (m)"
      unit="m"
      :current-year="currentYear"
      :distance-by-weeks="elevationByWeeks"
    />
  </div>
</template>

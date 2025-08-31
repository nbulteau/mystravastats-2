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
const cadenceByWeeks = computed(() => contextStore.cadenceByWeeks);


const cadenceUnit = computed(() => {
  if (currentActivity.value.endsWith("Run") ) return "ppm";
  if (currentActivity.value.endsWith("Ride")) return "rpm";
  return null;
});

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
    <ByWeeksChart
        title="Cadence"
        :unit="cadenceUnit"
        :distance-by-weeks="cadenceByWeeks"
        v-if="cadenceUnit"
    />

  </div>
</template>

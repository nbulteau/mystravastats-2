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
  <div
    v-if="currentYear !== 'All years'"
    class="chart-stack"
  >
    <section class="chart-panel">
      <ByMonthsChart
        title="Distance"
        unit="km"
        :data-by-months="distanceByMonths"
      />
    </section>
    <section class="chart-panel">
      <ByMonthsChart
        title="Elevation"
        unit="m"
        :data-by-months="elevationByMonths"
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
      />
    </section>
    <section class="chart-panel">
      <ByWeeksChart
        title="Elevation"
        unit="m"
        :items-by-weeks="elevationByWeeks"
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
      />
    </section>
  </div>

  <div
    v-else
    class="chart-empty"
  >
    Select a specific year to display monthly and weekly charts.
  </div>
</template>

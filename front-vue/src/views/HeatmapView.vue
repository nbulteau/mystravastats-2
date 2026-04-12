<script setup lang="ts">
import { computed } from "vue";
import { useContextStore } from "@/stores/context.js";
import ActivityHeatmapChart from "@/components/charts/ActivityHeatmapChart.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("heatmap");

const activityHeatmap = computed(() => contextStore.activityHeatmap);
const currentYear = computed(() => contextStore.currentYear);
</script>

<template>
  <div class="chart-stack">
    <section class="chart-panel chart-panel--wide">
      <ActivityHeatmapChart
        :activity-heatmap="activityHeatmap"
        :selected-year="currentYear"
      />
    </section>
  </div>
</template>

<style scoped>
.chart-panel--wide {
  grid-column: 1 / -1;
}
</style>

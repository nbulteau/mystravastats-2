<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useContextStore } from "@/stores/context.js";
import { useDashboardStore } from "@/stores/dashboard";
import ActivityHeatmapChart from "@/components/charts/ActivityHeatmapChart.vue";

const contextStore = useContextStore();
const dashboardStore = useDashboardStore();
onMounted(() => contextStore.updateCurrentView("heatmap"));

const activityHeatmap = computed(() => dashboardStore.activityHeatmap);
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

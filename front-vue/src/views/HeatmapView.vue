<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useContextStore } from "@/stores/context.js";
import { useDashboardStore } from "@/stores/dashboard";
import type { HeatmapScope } from "@/stores/dashboard";
import ActivityHeatmapChart from "@/components/charts/ActivityHeatmapChart.vue";

const contextStore = useContextStore();
const dashboardStore = useDashboardStore();
onMounted(() => contextStore.updateCurrentView("heatmap"));

const activityHeatmap = computed(() => dashboardStore.activityHeatmap);
const currentYear = computed(() => contextStore.currentYear);
const heatmapScope = computed(() => dashboardStore.heatmapScope);

async function onScopeChange(scope: HeatmapScope) {
  if (dashboardStore.heatmapScope === scope) {
    return;
  }
  dashboardStore.setHeatmapScope(scope);
  await dashboardStore.ensureHeatmapLoaded();
}
</script>

<template>
  <div class="chart-stack heatmap-view">
    <div class="heatmap-scope-toolbar">
      <span class="heatmap-scope-label">Scope</span>
      <div
        class="btn-group btn-group-sm"
        role="group"
        aria-label="Heatmap scope"
      >
        <button
          type="button"
          class="btn"
          :class="heatmapScope === 'selection' ? 'btn-primary' : 'btn-outline-primary'"
          @click="onScopeChange('selection')"
        >
          Current selection
        </button>
        <button
          type="button"
          class="btn"
          :class="heatmapScope === 'all-sports' ? 'btn-primary' : 'btn-outline-primary'"
          @click="onScopeChange('all-sports')"
        >
          All sports
        </button>
      </div>
    </div>
    <p
      v-if="heatmapScope === 'all-sports'"
      class="heatmap-scope-note"
    >
      All sports mode aggregates ride, run, hike, and ski activities. Metrics stay unit-safe
      (distance, elevation, duration) to avoid speed/pace confusion.
    </p>
    <section class="chart-panel chart-panel--wide">
      <ActivityHeatmapChart
        :activity-heatmap="activityHeatmap"
        :selected-year="currentYear"
      />
    </section>
  </div>
</template>

<style scoped>
.heatmap-view {
  gap: 0.75rem;
}

.heatmap-scope-toolbar {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.heatmap-scope-label {
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: #5f6470;
  text-transform: uppercase;
}

.heatmap-scope-note {
  margin: 0;
  font-size: 0.88rem;
  color: #5f6470;
}

.chart-panel--wide {
  grid-column: 1 / -1;
}
</style>

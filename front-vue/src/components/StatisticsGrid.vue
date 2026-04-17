<script setup lang="ts">
import { computed, ref } from "vue";
import VGrid, { VGridVueTemplate, type CellProps } from "@revolist/vue3-datagrid";
import type { Statistics } from "@/models/statistics.model";
import ActivityCellRenderer from "./cell-renderers/ActivityCellRenderer.vue";
import MetricLabelCellRenderer from "./cell-renderers/MetricLabelCellRenderer.vue";

const props = withDefaults(defineProps<{
  year: string;
  statistics: Statistics[];
  height?: string;
}>(), {
  height: "calc(100vh - 150px)",
});

const yearLabel = computed(() => props.year === "All years" ? "All years" : props.year);
const statisticsCount = computed(() => props.statistics.length);

const columns = ref([
  {
    prop: "label", name: "Statistic", size: 250, cellTemplate: VGridVueTemplate(MetricLabelCellRenderer),
    columnProperties: (): CellProps => {
      return {
        style: {
          fontWeight: 'bold',
        },
      }
    },
  },
  { prop: "value", name: "Value", size: 250 },
  { prop: "activity", name: "Activity", size: 500, cellTemplate: VGridVueTemplate(ActivityCellRenderer) },
]);
</script>

<template>
  <section class="grid-shell">
    <header class="statistics-grid-header">
      <div class="statistics-grid-title">
        Statistics · {{ yearLabel }}
      </div>
      <div class="statistics-grid-meta">
        {{ statisticsCount }} metrics
      </div>
    </header>
    <VGrid
      name="statisticsGrid"
      theme="material"
      :columns="columns"
      :source="statistics"
      :readonly="true"
      :style="{ height: props.height }"
    />
  </section>
</template>

<style scoped>
.statistics-grid-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.statistics-grid-title {
  font-weight: 700;
  color: var(--ms-text);
}

.statistics-grid-meta {
  color: var(--ms-text-muted);
  font-size: 0.9rem;
}
</style>

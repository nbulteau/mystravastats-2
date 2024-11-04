<script setup lang="ts">
import { ref } from "vue";
import VGrid, { VGridVueTemplate, type CellProps } from "@revolist/vue3-datagrid";
import type { Statistics } from "@/models/statistics.model";
import ActivityCellRenderer from "./cell-renderers/ActivityCellRenderer.vue";

defineProps<{
  year: string;
  activity: string;
  statistics: Statistics[];
}>();

const columns = ref([
  {
    prop: "label", name: "Statistic", size: 250,
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
  <VGrid
    name="statisticsGrid"
    theme="material"
    :columns="columns"
    :source="statistics"
    :readonly="true"
    style="height: 100%; height: calc(100vh - 150px);"
  />
</template>

<style scoped></style>

<script setup lang="ts">
import { ref } from "vue";
import VGrid, { VGridVueTemplate } from "@revolist/vue3-datagrid";
import type { Statistics } from "@/models/statistics.model";
import ActivityCellRenderer from "./cell-renderers/ActivityCellRenderer.vue";

defineProps<{
  year: string;
  activity: string;
  statistics: Statistics[];
}>();

const columns = ref([
  { prop: "label", name: "Statistic", size: 250 },
  { prop: "value", name: "Value", size: 250 },
  { prop: "activity", name: "Activity", size: 500, cellTemplate: VGridVueTemplate(ActivityCellRenderer) },
]);
</script>

<template>
  <VGrid
    name="statisticsGrid" 
    theme="material" 
    :exporting="true"
    :columns="columns"
    :source="statistics" 
    style="height: 100%; height: calc(100vh - 150px);" 
  />
</template>

<style scoped></style>

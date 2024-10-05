<script setup lang="ts">
import { ref } from "vue";
import VGrid, { VGridVueTemplate, type ColumnRegular, type ColumnProp, type CellProps, } from "@revolist/vue3-datagrid";
import type { Activity } from "@/models/activity.model";
import DistanceCellRenderer from "@/components/cell-renderers/DistanceCellRenderer.vue";
import ElapsedTimeCellRenderer from "@/components/cell-renderers/ElapsedTimeCellRenderer.vue";
import ElevationGainCellRenderer from "@/components/cell-renderers/ElevationGainCellRenderer.vue";
import SpeedCellRenderer from "@/components/cell-renderers/SpeedCellRenderer.vue";
import NameCellRenderer from "./cell-renderers/NameCellRenderer.vue";
import DateCellRenderer from "./cell-renderers/DateCellRenderer.vue";
import GradientCellRenderer from "./cell-renderers/GradientCellRenderer.vue";

const props = defineProps<{
  activities: Activity[];
  currentActivity: string;
  currentYear: string;
}>();

async function csvExport() {
  let url = `http://localhost:8080/api/activities/csv?activityType=${props.currentActivity}`;
  if (props.currentYear != "All years") {
    url = `${url}&year=${props.currentYear}`;
  }

  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error("Network response was not ok");
    }
    const blob = await response.blob();
    const objectUrl = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = objectUrl;
    const fileName = "activities-" + props.currentActivity + "-" + props.currentYear + ".csv"
    link.setAttribute("download", fileName);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  } catch (error) {
    console.error("There was an error exporting the CSV:", error);
  }
}

const columns = ref<ColumnRegular[]>([
  {
    prop: "name", name: "Activity", size: 500, pin: "colPinStart", cellTemplate: VGridVueTemplate(NameCellRenderer), sortable: false, columnType: 'string',
    // use this to return custom html per column
    columnTemplate: (createElement, column) => {
      return createElement('div', {
        style: {
          display: 'flex',
          alignItems: 'center',
        }
      }, [createElement('span', {
        style: {
          flex: '1',
        }
      }, column.name), createElement('button', {
        class: 'btn btn-sm btn-outline-secondary ms-2',
        onClick: async () => {
          await csvExport();
        }
      }, createElement('img', {
        src: "/icons/share-outline.svg",
        alt: "CSV export",
        style: {
          width: '25px',
          height: '25px'
        }
      }))]);
    },
    columnProperties: (): CellProps => {
      return {
        style: {
          fontWeight: 'bold',
        },
      }
    },
  },
  {
    prop: "distance", name: "Distance", size: 100, cellTemplate: VGridVueTemplate(DistanceCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "elapsedTime", name: "Elapsed time", size: 150, cellTemplate: VGridVueTemplate(ElapsedTimeCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "totalElevationGain", name: "Total elevation gain", size: 160, cellTemplate: VGridVueTemplate(ElevationGainCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "averageSpeed", name: "Average speed", size: 140, cellTemplate: VGridVueTemplate(SpeedCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "bestTimeForDistanceFor1000m", name: "Best speed for 1000m", size: 200, cellTemplate: VGridVueTemplate(SpeedCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "bestElevationForDistanceFor500m", name: "Best gradient for 500m", size: 180, cellTemplate: VGridVueTemplate(GradientCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "bestElevationForDistanceFor1000m", name: "Best gradient for 1000m", size: 180, cellTemplate: VGridVueTemplate(GradientCellRenderer),
    sortable: true,
    cellCompare: (prop: ColumnProp, a: { [x: string]: { toString: () => string; }; }, b: { [x: string]: { toString: () => string; }; }) => {
      const aValue = a[prop]?.toString();
      const bValue = b[prop]?.toString();
      return parseFloat(aValue) - parseFloat(bValue);
    },
    columnType: 'number'
  },
  {
    prop: "date", name: "Date", size: 200, cellTemplate: VGridVueTemplate(DateCellRenderer),
    sortable: true,
    columnType: 'date'
  },
]);
</script>

<template>
  <VGrid
    name="activitiesGrid"
    theme="material"
    :columns="columns"
    :source="activities"
    style="height: 100%; height: calc(100vh - 150px)"
  />
</template>

<style scoped></style>

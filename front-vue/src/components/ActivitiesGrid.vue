<script setup lang="ts">
import {eventBus} from "@/main";
import VGrid, {type ColumnProp, type ColumnRegular, VGridVueTemplate,} from "@revolist/vue3-datagrid";
import type {Activity} from "@/models/activity.model";
import DistanceCellRenderer from "@/components/cell-renderers/DistanceCellRenderer.vue";
import ElapsedTimeCellRenderer from "@/components/cell-renderers/ElapsedTimeCellRenderer.vue";
import ElevationGainCellRenderer from "@/components/cell-renderers/ElevationGainCellRenderer.vue";
import NameCellRenderer from "./cell-renderers/NameCellRenderer.vue";
import DateCellRenderer from "./cell-renderers/DateCellRenderer.vue";
import GradientCellRenderer from "./cell-renderers/GradientCellRenderer.vue";
import {useRouter} from 'vue-router';
import {computed, onMounted, ref} from "vue";
import shareIcon from "@/assets/share-outline.svg";
import AverageSpeedCellRenderer from "@/components/cell-renderers/AverageSpeedCellRenderer.vue";
import BestSpeedFor1000mCellRenderer from "@/components/cell-renderers/BestSpeedFor1000mCellRenderer.vue";


const props = defineProps<{
  activities: Activity[];
  currentActivity: string;
  currentYear: string;
}>();

const router = useRouter();

function showDetailedActivity(activityId: string) {

  // Navigate to the detailed activity view
  router.push(`/activities/${activityId}`)
    .then(() => {
      console.log("Navigated to detailed activity view");
    })
    .catch((error) => {
      console.error("Failed to navigate:", error);
    });
}

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

const numericCompare = (prop: ColumnProp, a: { [x: string]: any }, b: { [x: string]: any }) => {
  if (a[prop] === undefined || a[prop] === null) return -1;
  if (b[prop] === undefined || b[prop] === null) return 1;
  
  const aValue = parseFloat(a[prop].toString());
  const bValue = parseFloat(b[prop].toString());
  
  if (isNaN(aValue) && isNaN(bValue)) return 0;
  if (isNaN(aValue)) return -1;
  if (isNaN(bValue)) return 1;
  
  return aValue - bValue;
};

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
        async onClick() {
          await csvExport();
        }
      }, createElement('img', {
        src: shareIcon,
        alt: "CSV export",
        style: {
          width: '25px',
          height: '25px'
        }
      }))]);
    },
  },
  {
    prop: "distance",
    name: "Distance",
    size: 100,
    cellTemplate: VGridVueTemplate(DistanceCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "elapsedTime",
    name: "Elapsed time",
    size: 150,
    cellTemplate: VGridVueTemplate(ElapsedTimeCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "totalElevationGain",
    name: "Total elevation gain",
    size: 160,
    cellTemplate: VGridVueTemplate(ElevationGainCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "averageSpeed",
    name: "Average speed",
    size: 140,
    cellTemplate: VGridVueTemplate(AverageSpeedCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "bestSpeedForDistanceFor1000m",
    name: "Best speed for 1000m",
    size: 200,
    cellTemplate: VGridVueTemplate(BestSpeedFor1000mCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "bestElevationForDistanceFor500m",
    name: "Best gradient for 500m",
    size: 180,
    cellTemplate: VGridVueTemplate(GradientCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "bestElevationForDistanceFor1000m",
    name: "Best gradient for 1000m",
    size: 180,
    cellTemplate: VGridVueTemplate(GradientCellRenderer),
    sortable: true,
    cellCompare: numericCompare,
    columnType: 'number'
  },
  {
    prop: "date", name: "Date", size: 200, cellTemplate: VGridVueTemplate(DateCellRenderer),
    sortable: true,
    columnType: 'date'
  },
]);

const footerData = computed(() => {
  if (!props.activities.length) return {};

  const totalDistance = props.activities.reduce((sum, activity) => sum + (Number(activity.distance) || 0), 0);
  const totalElapsedTime = props.activities.reduce((sum, activity) => sum + (Number(activity.elapsedTime) || 0), 0);
  const totalMovingTime = props.activities.reduce((sum, activity) => sum + (Number(activity.movingTime) || 0), 0);
  const totalElevationGain = props.activities.reduce((sum, activity) => sum + (Number(activity.totalElevationGain) || 0), 0);

  let avgSpeed = 0.0;
  if (totalMovingTime > 0) {
     // speed in m/s
    avgSpeed = totalDistance / totalMovingTime
  }

  const bestSpeed1000m = Math.max(...props.activities.map(a => Number(a.bestSpeedForDistanceFor1000m) || 0));
  const bestGradient500m = Math.max(...props.activities.map(a => Number(a.bestElevationForDistanceFor500m) || 0));
  const bestGradient1000m = Math.max(...props.activities.map(a => Number(a.bestElevationForDistanceFor1000m) || 0));

  return {
    name: "",
    type: props.currentActivity,
    distance: totalDistance,
    elapsedTime: totalElapsedTime,
    totalElevationGain: totalElevationGain,
    averageSpeed: avgSpeed,
    bestSpeedForDistanceFor1000m: bestSpeed1000m,
    bestElevationForDistanceFor500m: bestGradient500m,
    bestElevationForDistanceFor1000m: bestGradient1000m,
    date: ""
  };
});

onMounted(() => {
  eventBus.on("detailledActivityClick", (event: any) => showDetailedActivity(event as string));
});
</script>

<template>
  <VGrid
      name="activitiesGrid"
      theme="material"
      :columns="columns"
      :source="activities"
      :pinnedBottomSource="[footerData]"
      :readonly="true"
      style="height: calc(100vh - 150px)"
  />
</template>

<style scoped></style>

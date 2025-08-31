<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, XAxisOptions } from "highcharts";

const props = defineProps<{
  title: string
  elevationByWeeks: Map<string, number>[]
}>();

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: props.title },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        fontSize: "13px",
        fontFamily: "Verdana, sans-serif",
      },
    },
  },
  yAxis: {
    min: 0,
    title: {
      text: "Elevation (m)",
    },
  },
  legend: {
    enabled: false,
  },
  tooltip: {
    formatter: function (this: any): string {
      return this.points.reduce(function (
        s: string,
        point: {x: string; y: number}
      ) { return `${s}: <span>${Math.round(point.y)} m</span>`; },
        "<b>" + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Elevation",
      type: "column",
      colorByPoint: false,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        //color: "#FFFFFF",
        inside: true,
        verticalAlign: "top",
        format: "{point.y:.1f}", // one decimal
        y: 10, // 10 pixels down from the top
        style: {
          fontSize: "13px",
          fontFamily: "Verdana, sans-serif",
        },
      },
      data: [], // Initialize with an empty array
    },
  ],
});

// Function to convert the array of objects to an array of numbers
function convertToNumberArray(data: Map<string, number>[]): number[] {
  return data.map((item) => Object.values(item)[0]);
}

// Watch for changes in ElevationByWeeks and update the chart data
watch(
  () => props.elevationByWeeks,
  (newData) => {
    if (chartOptions.series && chartOptions.series.length > 0) {
      (chartOptions.series[0] as SeriesColumnOptions).data = convertToNumberArray(newData);
    }
    (chartOptions.xAxis as XAxisOptions).categories = Array.from(newData.keys()).map(String);
  },
  { immediate: true } // Immediate to handle initial data
); 
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped></style>

<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, XAxisOptions } from "highcharts";
import { calculateAverageLine } from "@/utils/charts";

const props = defineProps<{
  title: string;
  unit: string;
  distanceByWeeks: Map<string, number>[];
}>();

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: props.title },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        //fontSize: "13px",
        //fontFamily: "Verdana, sans-serif",
      },
    },
  },
  yAxis: {
    min: 0,
    title: {
      text: props.title + ' (' + props.unit + ')',
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
        ) { return `${s}: <span>${Math.round(point.y)} ${props.unit}</span>`; },
        "<b>week: " + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Distance",
      type: "column",
      colorByPoint: false,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        color: "#000000",
        inside: true,
        verticalAlign: "top",
        format: "{point.y:.0f}", // No decimal
        y: 10, // 10 pixels down from the top
        style: {
          fontSize: "13px",
          //fontFamily: "Verdana, sans-serif",
        },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Average distance",
      type: "line",
      color: "red",
      dashStyle: "ShortDash",
      marker: {
        enabled: false,
      },
      enableMouseTracking: false,
      dataLabels: {
        enabled: true,
        formatter: function (this: Highcharts.Point) {
          // Display the label only for the first point
          if (this.index === 0) {
            return 'average ' + props.title.toLowerCase() + ` by weeks : ${this.y ? this.y.toFixed(1) : 0} ` + props.unit;
          }
          return null;
        },
        style: {
          //fontSize: "13px",
          //fontFamily: "Verdana, sans-serif",
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

// Watch for changes in distanceByWeeks and update the chart data
watch(() => props.distanceByWeeks, (newData) => {
  if (chartOptions.series && chartOptions.series.length > 0) {
    const data = convertToNumberArray(newData);

    (chartOptions.series[0] as SeriesColumnOptions).data = data;

    (chartOptions.series[1] as SeriesColumnOptions).data = calculateAverageLine(data);
  }
  (chartOptions.xAxis as XAxisOptions).categories = Array.from(newData.keys()).map(String);
}, { immediate: true }); // Immediate to handle initial data
</script>


<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>



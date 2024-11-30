<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions } from "highcharts";
import { calculateAverageLine } from "@/utils/charts";

const props = defineProps<{
  title: string;
  unit: string;
  dataByMonths: Map<string, number>[];
}>();

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: props.title + " by months" },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        fontSize: "13px",
        fontFamily: "Verdana, sans-serif",
      },
    },
    categories: [
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December",
    ],
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
      return this.points.reduce(function (s: string, point: { x: string; y: number }) {
        return `${s}: <span>${point.y.toFixed(2)} ${props.unit}</span>`;
      }, "<b>" + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Distance",
      type: "column",
      colors: [
        "#9b20d9",
        "#9215ac",
        "#861ec9",
        "#7a17e6",
        "#7010f9",
        "#691af3",
        "#6225ed",
        "#5b30e7",
        "#533be1",
        "#4c46db",
        "#4551d5",
        "#3e5ccf",
      ],
      colorByPoint: true,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        color: "#FFFFFF",
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
            return 'Average ' + props.title.toLowerCase() + ` by months : ${this.y ? this.y.toFixed(1) : 0} ` + props.unit;
          }
          return null;
        },
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

// Watch for changes in distanceByMonths and update the chart data
watch(
  () => props.dataByMonths,
  (newData) => {
    if (chartOptions.series && chartOptions.series.length > 0) {
      const data = convertToNumberArray(newData);
      (chartOptions.series[0] as SeriesColumnOptions).data = data;

      (chartOptions.series[1] as SeriesColumnOptions).data = calculateAverageLine(data);
    }
  },
  { immediate: true }
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped></style>

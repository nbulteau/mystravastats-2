<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, YAxisOptions } from "highcharts";
import {formatSpeed, formatSpeedWithUnit} from "@/utils/formatters";

const props = defineProps<{
  activityType: string;
  dataByMonths: Map<string, number>[];
}>();

const unit = computed(() => ((props.activityType === "Run" || props.activityType === "TrailRun") ? "min/km" : "km/h"));
const MONTH_COLORS = [
  "#ffe3d5",
  "#ffd7c5",
  "#ffc9b2",
  "#ffb998",
  "#ffa983",
  "#ff9a6e",
  "#ff8a5b",
  "#ff7a48",
  "#ff6a36",
  "#fc5a1b",
  "#ef4b06",
  "#dd3f00",
];

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: "Average speed by month" },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        fontSize: "12px",
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
      text: `Average speed ${unit.value}`,
    },
    labels: {
      formatter: function (this: any): string {
        if(this.value === 0) {
            return '';
          }
        return formatSpeedWithUnit(this.value, props.activityType);
      },
    },
  },
  legend: {
    enabled: false,
  },
  tooltip: {
    formatter: function (this: any): string {
      if(this.y === 0) {
            return 'Not available';
          }
      const speed = formatSpeedWithUnit(this.y, props.activityType);

      return this.points.reduce(function (s: string) {
        return `${s}: <span>${speed}</span>`;
      }, `<b>${(chartOptions.xAxis as Highcharts.XAxisOptions).categories?.[this.point.index]}</b>`);
    },
    shared: true,
  },
  series: [
    {
      name: "Average speed",
      type: "column",
      colors: MONTH_COLORS,
      colorByPoint: true,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        color: "#2f323a",
        inside: true,
        verticalAlign: "top",
        formatter: function (this: any): string {

          return formatSpeed(this.y, props.activityType);
        },
        y: 10, // 10 pixels down from the top
        style: {
          fontSize: "13px",
          //fontFamily: "Verdana, sans-serif",
        },
      },
      data: [], // Initialize with an empty array
    },
  ],
});

// Function to convert the array of objects to an array of numbers
function convertToNumberArray(data: Map<string, number>[]): number[] {
  return data.map((item) => Object.values(item)[0], props.activityType);
}

// Watch for changes in distanceByMonths and update the chart data
watch(
  () => props.dataByMonths,
  (newData) => {
    if (chartOptions.series && chartOptions.series.length > 0) {
      (chartOptions.series[0] as SeriesColumnOptions).data = convertToNumberArray(
        newData
      );
    }
    if (chartOptions.yAxis && (chartOptions.yAxis as YAxisOptions).title) {
      (chartOptions.yAxis as YAxisOptions).title!.text = `Average speed ${unit.value}`;
    }
  },
  { immediate: true }
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped></style>

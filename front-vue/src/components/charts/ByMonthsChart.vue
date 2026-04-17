<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, Point} from "highcharts"; // Import Point here
import {calculateYtdAverageLine, extractPeriodEntries} from "@/utils/charts";
import type { ChartPeriodPoint } from "@/models/chart-period-point.model";

const props = defineProps<{
  title: string;
  unit: string;
  dataByMonths: ChartPeriodPoint[];
  selectedYear: string;
}>();

const chartOptions: Highcharts.Options = reactive({
  chart: {type: "column"},
  title: {text: props.title + " by months"},
  subtitle: {
    text: "",
  },
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
      return this.points.reduce(function (s: string, point: { x: string; y: number; point: { activityCount?: number } }) {
        const activityCount = Number(point.point.activityCount ?? 0);
        const activityLabel = activityCount === 1 ? "activity" : "activities";
        return `${s}<br/><span>${point.y.toFixed(2)} ${props.unit} · ${activityCount} ${activityLabel}</span>`;
      }, `<b>${(chartOptions.xAxis as Highcharts.XAxisOptions).categories?.[this.point.index]}</b>`);
    },
    shared: true,
  },
  series: [
    {
      name: "Distance",
      type: "column",
      colors: [
        `#FF5733`,
        `#33C4FF`,
        `#9B59B6`,
        `#E67E22`,
        `#28B463`,
        `#F39C12`,
        `#8E44AD`,
        `#1ABC9C`,
        `#2ECC71`,
        `#3498DB`,
        `#D35400`,
        `#34495E`,

      ],
      colorByPoint: true,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        color: "#000000",
        inside: true,
        verticalAlign: "top",
        format: "{point.y:.0f}",  // No decimal
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
        formatter: function (this: Point) { // Use Point here
          // Display the label only for the first point
          if (this.index === 0) {
            const isCurrentYear = props.selectedYear === String(new Date().getFullYear());
            const labelPrefix = isCurrentYear ? "YTD average" : "Average";
            return `${labelPrefix} ${props.title.toLowerCase()} by months : ${this.y ? this.y.toFixed(1) : 0} ${props.unit}`;
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

// Watch for changes in distanceByMonths and update the chart data
watch(
    () => props.dataByMonths,
    (newData) => {
      if (chartOptions.series && chartOptions.series.length > 0) {
        const entries = extractPeriodEntries(newData);
        const data = entries.map((entry) => ({
          y: entry.value,
          activityCount: entry.activityCount,
        }));
        (chartOptions.series[0] as SeriesColumnOptions).data = data;

        (chartOptions.series[1] as SeriesColumnOptions).data = calculateYtdAverageLine(
          entries.map((entry) => entry.value),
          props.selectedYear,
          "MONTHS",
        );

        const activeMonths = entries.filter((entry) => entry.activityCount > 0).length;
        const totalActivities = entries.reduce((sum, entry) => sum + entry.activityCount, 0);
        chartOptions.subtitle = {
          text: `${totalActivities} activities across ${activeMonths} active month(s)`,
        };
      }
    },
    {immediate: true}
);
</script>

<template>
  <Chart :options="chartOptions"/>
</template>

<style scoped></style>

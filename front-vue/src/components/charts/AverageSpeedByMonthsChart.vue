<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, YAxisOptions } from "highcharts";
import {formatSpeed, formatSpeedWithUnit} from "@/utils/formatters";
import type { ChartPeriodPoint } from "@/models/chart-period-point.model";
import { extractPeriodEntries } from "@/utils/charts";

const props = defineProps<{
  activityType: string;
  dataByMonths: ChartPeriodPoint[];
}>();

const unit = computed(() => ((props.activityType === "Run" || props.activityType === "TrailRun") ? "min/km" : "km/h"));
const MONTH_COLORS = [
  "#FF5733",
  "#33C4FF",
  "#9B59B6",
  "#E67E22",
  "#28B463",
  "#F39C12",
  "#8E44AD",
  "#1ABC9C",
  "#2ECC71",
  "#3498DB",
  "#D35400",
  "#34495E",
];

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: "Average speed by month" },
  subtitle: {
    text: "",
  },
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
        return "Not available";
      }
      const speed = formatSpeedWithUnit(this.y, props.activityType);
      const activityCount = Number(this.point?.activityCount ?? 0);
      const activityLabel = activityCount === 1 ? "activity" : "activities";

      return this.points.reduce(function (s: string) {
        return `${s}<br/><span>${speed} · ${activityCount} ${activityLabel}</span>`;
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

// Watch for changes in distanceByMonths and update the chart data
watch(
  () => props.dataByMonths,
  (newData) => {
    const entries = extractPeriodEntries(newData);
    if (chartOptions.series && chartOptions.series.length > 0) {
      (chartOptions.series[0] as SeriesColumnOptions).data = entries.map((entry) => ({
        y: entry.value,
        activityCount: entry.activityCount,
      }));
    }
    if (chartOptions.yAxis && (chartOptions.yAxis as YAxisOptions).title) {
      (chartOptions.yAxis as YAxisOptions).title!.text = `Average speed ${unit.value}`;
    }
    const activeMonths = entries.filter((entry) => entry.activityCount > 0).length;
    const totalActivities = entries.reduce((sum, entry) => sum + entry.activityCount, 0);
    chartOptions.subtitle = {
      text: `${totalActivities} activities across ${activeMonths} active month(s)`,
    };
  },
  { immediate: true }
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped></style>

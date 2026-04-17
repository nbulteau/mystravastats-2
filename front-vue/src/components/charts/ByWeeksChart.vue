<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, XAxisOptions } from "highcharts";
import { calculateYtdAverageLine, extractPeriodEntries, weekLabel } from "@/utils/charts";
import type { ChartPeriodPoint } from "@/models/chart-period-point.model";

const props = defineProps<{
  title: string;
  unit: string;
  itemsByWeeks: ChartPeriodPoint[];
  selectedYear: string;
}>();

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: props.title },
  subtitle: {
    text: "",
  },
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
        point: {x: string; y: number; point: { activityCount?: number }}
      ) {
        const activityCount = Number(point.point.activityCount ?? 0);
        const activityLabel = activityCount === 1 ? "activity" : "activities";
        return `${s}<br/><span>${Math.round(point.y)} ${props.unit} · ${activityCount} ${activityLabel}</span>`;
      },
        `<b>${this.x}</b>`);
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
            const isCurrentYear = props.selectedYear === String(new Date().getFullYear());
            const labelPrefix = isCurrentYear ? "YTD average" : "Average";
            return `${labelPrefix} ${props.title.toLowerCase()} by weeks : ${this.y ? this.y.toFixed(1) : 0} ${props.unit}`;
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

// Watch for changes in itemsByWeeks and update the chart data
watch(() => props.itemsByWeeks, (newData) => {
  if (chartOptions.series && chartOptions.series.length > 0) {
    const entries = extractPeriodEntries(newData);
    const data = entries.map((entry) => ({
      y: entry.value,
      activityCount: entry.activityCount,
    }));
    const categories = entries.map((entry) => weekLabel(entry.key));

    (chartOptions.series[0] as SeriesColumnOptions).data = data;

    (chartOptions.series[1] as SeriesColumnOptions).data = calculateYtdAverageLine(
      entries.map((entry) => entry.value),
      props.selectedYear,
      "WEEKS",
    );
    (chartOptions.xAxis as XAxisOptions).categories = categories;

    const activeWeeks = entries.filter((entry) => entry.activityCount > 0).length;
    const totalActivities = entries.reduce((sum, entry) => sum + entry.activityCount, 0);
    chartOptions.subtitle = {
      text: `${totalActivities} activities across ${activeWeeks} active week(s)`,
    };
  }
}, { immediate: true }); // Immediate to handle initial data
</script>


<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>

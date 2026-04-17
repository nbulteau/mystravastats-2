<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, SeriesLineOptions, XAxisOptions } from "highcharts";
import type { HeartRateZoneActivitySummary } from "@/models/heart-rate-zone.model";
import {
  isoWeekNumber,
  parseActivityDate,
  rollingAverage,
  weekLabel,
  weeksInIsoYear,
} from "@/utils/charts";

const props = defineProps<{
  activitySummaries: HeartRateZoneActivitySummary[];
  selectedYear: string;
}>();

type WeeklyTrainingLoadPoint = {
  y: number;
  trackedHours: number;
};

function zoneWeight(zoneCode: string): number {
  switch (zoneCode) {
    case "Z1":
      return 1;
    case "Z2":
      return 2;
    case "Z3":
      return 3;
    case "Z4":
      return 4;
    case "Z5":
      return 5;
    default:
      return 0;
  }
}

function computeWeeklyTrainingLoad() {
  const selectedYear = Number.parseInt(props.selectedYear, 10);
  const hasYearFilter = !Number.isNaN(selectedYear);
  const totalWeeks = hasYearFilter ? weeksInIsoYear(selectedYear) : 52;

  const trimpByWeek = Array<number>(totalWeeks).fill(0);
  const trackedHoursByWeek = Array<number>(totalWeeks).fill(0);

  for (const summary of props.activitySummaries) {
    const date = parseActivityDate(summary.activityDate);
    if (!date) {
      continue;
    }
    if (hasYearFilter && date.getFullYear() !== selectedYear) {
      continue;
    }

    const week = isoWeekNumber(date);
    if (week < 1 || week > totalWeeks) {
      continue;
    }

    const trimp = summary.zones.reduce((sum, zone) => {
      return sum + zoneWeight(zone.zone) * (zone.seconds / 60);
    }, 0);

    const index = week - 1;
    trimpByWeek[index] += trimp;
    trackedHoursByWeek[index] += summary.totalTrackedSeconds / 3600;
  }

  const categories = Array.from({ length: totalWeeks }, (_, index) => {
    return weekLabel(String(index + 1).padStart(2, "0"));
  });

  const trimpPoints: WeeklyTrainingLoadPoint[] = trimpByWeek.map((value, index) => ({
    y: Math.round(value * 10) / 10,
    trackedHours: Math.round(trackedHoursByWeek[index] * 100) / 100,
  }));

  const movingAverage = rollingAverage(
    trimpPoints.map((point) => (point.trackedHours > 0 ? point.y : null)),
    4,
  ).map((value) => (value === null ? null : Math.round(value * 10) / 10));

  const totalTrackedHours = trackedHoursByWeek.reduce((sum, value) => sum + value, 0);
  const activeWeeks = trackedHoursByWeek.filter((value) => value > 0).length;

  return {
    categories,
    trimpPoints,
    movingAverage,
    activeWeeks,
    totalTrackedHours,
  };
}

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: "Weekly training load (TRIMP)" },
  subtitle: { text: "" },
  xAxis: {
    categories: [],
    crosshair: true,
  },
  yAxis: {
    min: 0,
    title: {
      text: "Training load (TRIMP points)",
    },
  },
  legend: {
    enabled: true,
  },
  tooltip: {
    shared: true,
    formatter: function (this: any): string {
      const details = this.points.map((point: any) => {
        if (point.series.name === "TRIMP") {
          const trackedHours = Number(point.point.trackedHours ?? 0).toFixed(2);
          return `<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y.toFixed(1)} pts · ${trackedHours} h tracked`;
        }
        return `<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y?.toFixed?.(1) ?? point.y} pts`;
      });
      return `<b>${this.x}</b>${details.join("")}`;
    },
  },
  series: [
    {
      name: "TRIMP",
      type: "column",
      color: "#fc4c02",
      data: [],
    },
    {
      name: "4-week moving average",
      type: "line",
      color: "#2f6df6",
      marker: { enabled: false },
      data: [],
    },
  ],
});

function updateChartData() {
  const computed = computeWeeklyTrainingLoad();
  (chartOptions.xAxis as XAxisOptions).categories = computed.categories;
  (chartOptions.series?.[0] as SeriesColumnOptions).data = computed.trimpPoints;
  (chartOptions.series?.[1] as SeriesLineOptions).data = computed.movingAverage;

  chartOptions.subtitle = {
    text:
      computed.totalTrackedHours > 0
        ? `${computed.totalTrackedHours.toFixed(1)} tracked HR hour(s) · ${computed.activeWeeks} active week(s)`
        : "No heart-rate stream data available for weekly training load.",
  };
}

watch(
  () => [props.activitySummaries, props.selectedYear],
  updateChartData,
  { immediate: true, deep: true },
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>

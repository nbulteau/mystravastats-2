<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { Activity } from "@/models/activity.model";
import type { SeriesLineOptions, XAxisOptions } from "highcharts";
import {
  isoWeekNumber,
  parseActivityDate,
  rollingAverage,
  weekLabel,
  weeksInIsoYear,
} from "@/utils/charts";

const props = defineProps<{
  activities: Activity[];
  selectedYear: string;
}>();

type WeeklyLongRideData = {
  categories: string[];
  weeklyMaxDistance: Array<number | null>;
  movingAverage: Array<number | null>;
  bestDistance: number;
  bestDate: string | null;
};

function formatDateLabel(value: string | null): string {
  if (!value) {
    return "N/A";
  }
  const parsed = parseActivityDate(value);
  if (!parsed) {
    return value;
  }
  return parsed.toLocaleDateString("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  });
}

function computeWeeklyLongRideData(): WeeklyLongRideData {
  const selectedYear = Number.parseInt(props.selectedYear, 10);
  const hasYearFilter = !Number.isNaN(selectedYear);
  const totalWeeks = hasYearFilter ? weeksInIsoYear(selectedYear) : 52;

  const weeklyMaxDistance = Array<number | null>(totalWeeks).fill(null);
  let bestDistance = 0;
  let bestDate: string | null = null;

  for (const activity of props.activities) {
    const date = parseActivityDate(activity.date);
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

    const distanceKm = activity.distance / 1000;
    const index = week - 1;
    const current = weeklyMaxDistance[index];
    if (current === null || distanceKm > current) {
      weeklyMaxDistance[index] = distanceKm;
    }

    if (distanceKm > bestDistance) {
      bestDistance = distanceKm;
      bestDate = activity.date;
    }
  }

  const categories = Array.from({ length: totalWeeks }, (_, index) => {
    return weekLabel(String(index + 1).padStart(2, "0"));
  });
  const movingAverage = rollingAverage(weeklyMaxDistance, 4).map((value) =>
    value === null ? null : Math.round(value * 10) / 10
  );

  return {
    categories,
    weeklyMaxDistance: weeklyMaxDistance.map((value) =>
      value === null ? null : Math.round(value * 10) / 10
    ),
    movingAverage,
    bestDistance: Math.round(bestDistance * 10) / 10,
    bestDate,
  };
}

const chartOptions: Highcharts.Options = reactive({
  chart: {
    type: "line",
  },
  title: {
    text: "Long ride progression",
  },
  subtitle: {
    text: "",
  },
  xAxis: {
    categories: [],
    crosshair: true,
  },
  yAxis: {
    min: 0,
    title: {
      text: "Distance (km)",
    },
  },
  tooltip: {
    shared: true,
    formatter: function (this: any): string {
      const details = this.points.map((point: any) => {
        const value = point.y === null ? "N/A" : `${point.y.toFixed(1)} km`;
        return `<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${value}`;
      });
      return `<b>${this.x}</b>${details.join("")}`;
    },
  },
  series: [
    {
      name: "Weekly max distance",
      type: "line",
      color: "#fc4c02",
      marker: {
        enabled: true,
        radius: 3,
      },
      data: [],
    },
    {
      name: "4-week moving average",
      type: "line",
      color: "#2f6df6",
      marker: {
        enabled: false,
      },
      dashStyle: "ShortDash",
      data: [],
    },
  ],
});

function updateChart() {
  const data = computeWeeklyLongRideData();

  (chartOptions.xAxis as XAxisOptions).categories = data.categories;
  (chartOptions.series?.[0] as SeriesLineOptions).data = data.weeklyMaxDistance;
  (chartOptions.series?.[1] as SeriesLineOptions).data = data.movingAverage;
  chartOptions.subtitle = {
    text:
      data.bestDistance > 0
        ? `Best long ride: ${data.bestDistance.toFixed(1)} km on ${formatDateLabel(data.bestDate)}`
        : "No rides available to compute long-ride progression.",
  };
}

watch(
  () => [props.activities, props.selectedYear],
  updateChart,
  { immediate: true, deep: true },
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>

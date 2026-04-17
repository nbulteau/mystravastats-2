<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { Activity } from "@/models/activity.model";
import type { SeriesColumnOptions, XAxisOptions } from "highcharts";

const props = defineProps<{
  activities: Activity[];
}>();

type HistogramData = {
  categories: string[];
  counts: number[];
  totalActivities: number;
  medianDistanceKm: number;
  maxDistanceKm: number;
};

function buildHistogram(distancesKm: number[]): { categories: string[]; counts: number[] } {
  if (distancesKm.length === 0) {
    return { categories: [], counts: [] };
  }

  const maxDistance = Math.max(...distancesKm);
  let binSize = 10;
  if (maxDistance <= 40) {
    binSize = 5;
  } else if (maxDistance > 120 && maxDistance <= 220) {
    binSize = 20;
  } else if (maxDistance > 220) {
    binSize = 25;
  }

  let binCount = Math.max(1, Math.ceil(maxDistance / binSize));
  if (binCount > 16) {
    binSize = Math.ceil(maxDistance / 16 / 5) * 5;
    binCount = Math.max(1, Math.ceil(maxDistance / binSize));
  }

  const counts = Array<number>(binCount).fill(0);
  for (const distance of distancesKm) {
    const rawIndex = Math.floor(distance / binSize);
    const index = Math.min(binCount - 1, rawIndex);
    counts[index] += 1;
  }

  const categories = Array.from({ length: binCount }, (_, index) => {
    const start = index * binSize;
    const end = (index + 1) * binSize;
    if (index === binCount - 1) {
      return `${start}-${end}+ km`;
    }
    return `${start}-${end} km`;
  });

  return { categories, counts };
}

function computeMedian(values: number[]): number {
  if (values.length === 0) {
    return 0;
  }
  const sorted = [...values].sort((left, right) => left - right);
  const middle = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) {
    return (sorted[middle - 1] + sorted[middle]) / 2;
  }
  return sorted[middle];
}

function computeChartData(): HistogramData {
  const distancesKm = props.activities
    .map((activity) => activity.distance / 1000)
    .filter((distance) => Number.isFinite(distance) && distance > 0);

  const { categories, counts } = buildHistogram(distancesKm);
  const medianDistanceKm = computeMedian(distancesKm);
  const maxDistanceKm = distancesKm.length > 0 ? Math.max(...distancesKm) : 0;

  return {
    categories,
    counts,
    totalActivities: distancesKm.length,
    medianDistanceKm,
    maxDistanceKm,
  };
}

const chartOptions: Highcharts.Options = reactive({
  chart: {
    type: "column",
  },
  title: {
    text: "Distance distribution",
  },
  subtitle: {
    text: "",
  },
  xAxis: {
    categories: [],
    crosshair: true,
    labels: {
      rotation: -35,
    },
  },
  yAxis: {
    min: 0,
    title: {
      text: "Number of activities",
    },
    allowDecimals: false,
  },
  legend: {
    enabled: false,
  },
  tooltip: {
    formatter: function (this: any): string {
      return `<b>${this.x}</b><br/>${this.y} activit${this.y === 1 ? "y" : "ies"}`;
    },
  },
  series: [
    {
      name: "Activities",
      type: "column",
      color: "#2f6df6",
      data: [],
    },
  ],
});

function updateChart() {
  const data = computeChartData();
  (chartOptions.xAxis as XAxisOptions).categories = data.categories;
  (chartOptions.series?.[0] as SeriesColumnOptions).data = data.counts;
  chartOptions.subtitle = {
    text:
      data.totalActivities > 0
        ? `${data.totalActivities} activities · median ${data.medianDistanceKm.toFixed(1)} km · max ${data.maxDistanceKm.toFixed(1)} km`
        : "No activities available for distance distribution.",
  };
}

watch(
  () => props.activities,
  updateChart,
  { immediate: true, deep: true },
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>

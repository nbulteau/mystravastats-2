<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesLineOptions, SeriesOptionsType } from "highcharts";

const props = defineProps<{
  movingTimeByYear: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: "line",
  },
  title: {
    text: "Moving time",
  },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        fontSize: "13px",
        fontFamily: "Verdana, sans-serif",
      },
    },
    categories: [] as string[],
    crosshair: true,
  },
  yAxis: {
    min: 0,
    title: {
      text: "Hours",
    },
    labels: {
      format: "{value} h",
    },
  },
  legend: {
    enabled: true,
  },
  tooltip: {
    formatter: function (this: any): string {
      return this.points.reduce(
        (summary: string, point: { color: string; series: { name: string }; y: number }) => {
          return `${summary}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y.toFixed(1)} h`;
        },
        `<b>${this.key}</b>`,
      );
    },
    shared: true,
  },
  series: [
    {
      name: "Moving time",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return `${this.y.toFixed(1)}h`;
        },
      },
      data: [],
    },
    {
      name: "Trend line",
      type: "line",
      dashStyle: "ShortDash",
      marker: {
        enabled: false,
      },
      enableMouseTracking: false,
      data: [],
    },
  ] as SeriesOptionsType[],
});

function toHours(seconds: number): number {
  if (!Number.isFinite(seconds)) {
    return 0;
  }
  return seconds / 3600;
}

function calculateTrendLine(data: number[]): number[] {
  const n = data.length;
  if (n === 0) {
    return [];
  }
  if (n === 1) {
    return [data[0]];
  }
  const xSum = data.reduce((sum, _, index) => sum + index, 0);
  const ySum = data.reduce((sum, value) => sum + value, 0);
  const xySum = data.reduce((sum, value, index) => sum + index * value, 0);
  const xSquaredSum = data.reduce((sum, _, index) => sum + index * index, 0);
  const denominator = (n * xSquaredSum - xSum * xSum);
  if (denominator === 0) {
    return [...data];
  }
  const slope = (n * xySum - xSum * ySum) / denominator;
  const intercept = (ySum - slope * xSum) / n;
  return data.map((_, index) => {
    const value = slope * index + intercept;
    return Number.isFinite(value) ? value : 0;
  });
}

function updateChartData() {
  const years = Object.keys(props.movingTimeByYear ?? {}).sort();
  const values = years.map((year) => toHours(props.movingTimeByYear?.[year] ?? 0));

  chartOptions.xAxis.categories = years;
  (chartOptions.series[0] as SeriesLineOptions).data = values;
  (chartOptions.series[1] as SeriesLineOptions).data = calculateTrendLine(values);
}

watch(
  () => props.movingTimeByYear,
  updateChartData,
  { immediate: true },
);
</script>

<template>
  <div class="chart-container">
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped></style>

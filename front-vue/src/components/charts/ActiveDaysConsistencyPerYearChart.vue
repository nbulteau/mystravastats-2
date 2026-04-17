<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesLineOptions, SeriesOptionsType } from "highcharts";

const props = defineProps<{
  activeDaysByYear: Record<string, number>;
  consistencyByYear: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: "line",
  },
  title: {
    text: "Active days / Consistency",
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
  yAxis: [
    {
      min: 0,
      title: {
        text: "Active days",
      },
    },
    {
      min: 0,
      max: 100,
      title: {
        text: "Consistency (%)",
      },
      labels: {
        format: "{value}%",
      },
      opposite: true,
    },
  ],
  legend: {
    enabled: true,
  },
  tooltip: {
    formatter: function (this: any): string {
      return this.points.reduce(
        (summary: string, point: { color: string; series: { name: string }; y: number }) => {
          const unit = point.series.name === "Consistency" ? "%" : " day(s)";
          const value = point.series.name === "Consistency" ? point.y.toFixed(1) : point.y.toFixed(0);
          return `${summary}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${value}${unit}`;
        },
        `<b>${this.key}</b>`,
      );
    },
    shared: true,
  },
  series: [
    {
      name: "Active days",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return `${this.y.toFixed(0)}`;
        },
      },
      data: [],
    },
    {
      name: "Consistency",
      type: "line",
      yAxis: 1,
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return `${this.y.toFixed(1)}%`;
        },
      },
      data: [],
    },
  ] as SeriesOptionsType[],
});

function updateChartData() {
  const years = Array.from(
    new Set([
      ...Object.keys(props.activeDaysByYear ?? {}),
      ...Object.keys(props.consistencyByYear ?? {}),
    ]),
  ).sort();

  chartOptions.xAxis.categories = years;

  (chartOptions.series[0] as SeriesLineOptions).data = years.map((year) => {
    const value = props.activeDaysByYear?.[year];
    return Number.isFinite(value) ? value : 0;
  });

  (chartOptions.series[1] as SeriesLineOptions).data = years.map((year) => {
    const value = props.consistencyByYear?.[year];
    return Number.isFinite(value) ? value : 0;
  });
}

watch(
  () => [props.activeDaysByYear, props.consistencyByYear],
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

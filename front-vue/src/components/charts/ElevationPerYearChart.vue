<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, SeriesLineOptions, SeriesOptionsType} from "highcharts";

const props = defineProps<{
  averageElevationByYear: Record<string, number>;
  maxElevationByYear: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Elevation",
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
    crosshair: true
  },
  yAxis: {
    min: 0,
    title: {
      text: `Elevation (m)`,
    },
  },
  legend: {
    enabled: true,
  },
  tooltip: {
    formatter: function (this: any): string {
      return this.points.reduce(function (
              s: any,
              point: {
                color: any; series: { name: string }; y: string
              }
          ) {
            return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y} m`;
          },
          "<b>" + this.key + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average elevation",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
        return `${this.y.toFixed(0)} m`;
      },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum elevation",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
        return `${this.y.toFixed(0)} m`;
      },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Trend line",
      type: "line",
      dashStyle: "ShortDash",
      marker: {
        enabled: false
      },
      enableMouseTracking: false,
      data: [], // Initialize with an empty array
    }
  ] as SeriesOptionsType[],
});

function updateChartData() {

  if (!props.averageElevationByYear || !props.maxElevationByYear ) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const averageElevationByYear = Object.values(props.averageElevationByYear);
    const maxElevationByYear = Object.values(props.maxElevationByYear);

    chartOptions.xAxis.categories = Object.keys(props.averageElevationByYear);

    const maxAverageElevation = Math.max(...averageElevationByYear);
    const maxAverageElevationIndex = averageElevationByYear.indexOf(maxAverageElevation);

    const maxMaxElevation = Math.max(...maxElevationByYear);
    const maxMaxElevationIndex = maxElevationByYear.indexOf(maxMaxElevation);

    (chartOptions.series[0] as SeriesColumnOptions).data = averageElevationByYear.map((value, index) => ({
      y: value,
      marker: index === maxAverageElevationIndex ? {enabled: true, radius: 6, fillColor: 'red'} : undefined
    }));

    (chartOptions.series[1] as SeriesColumnOptions).data = maxElevationByYear.map((value, index) => ({
      y: value,
      marker: index === maxMaxElevationIndex ? {enabled: true, radius: 6, fillColor: 'red'} : undefined
    }));

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageElevationByYear);
  }
}

function calculateTrendLine(data: number[]): number[] {
  const n = data.length;
  const xSum = data.reduce((sum, _, index) => sum + index, 0);
  const ySum = data.reduce((sum, value) => sum + value, 0);
  const xySum = data.reduce((sum, value, index) => sum + index * value, 0);
  const xSquaredSum = data.reduce((sum, _, index) => sum + index * index, 0);

  const slope = (n * xySum - xSum * ySum) / (n * xSquaredSum - xSum * xSum);
  const intercept = (ySum - slope * xSum) / n;

  return data.map((_, index) => slope * index + intercept);
}


watch(
    () => props.averageElevationByYear || props.maxElevationByYear,
    updateChartData,
    {immediate: true}
);

</script>

<template>
  <div class="chart-container">
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped>

</style>
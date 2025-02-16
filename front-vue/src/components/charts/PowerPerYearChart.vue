<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, SeriesLineOptions, SeriesOptionsType} from "highcharts";

const props = defineProps<{
  averageWattsByYear: Record<string, number>;
  maxWattsByYear: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Power",
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
      text: `Power (watts)`,
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
          "<b>" + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average watts",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
        return `${this.y.toFixed(0)} watts`;
      },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum watts",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
        return `${this.y.toFixed(0)} watts`;
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

  if (!props.averageWattsByYear || !props.maxWattsByYear ) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const averageWattsByYear = Object.values(props.averageWattsByYear);
    const maxWattsByYear = Object.values(props.maxWattsByYear);

    chartOptions.xAxis.categories = Object.keys(props.averageWattsByYear);

    const maxAverageWatts = Math.max(...averageWattsByYear);
    const maxAverageWattsIndex = averageWattsByYear.indexOf(maxAverageWatts);

    const maxMaxWatts = Math.max(...maxWattsByYear);
    const maxMaxWattsIndex = maxWattsByYear.indexOf(maxMaxWatts);

    (chartOptions.series[0] as SeriesColumnOptions).data = averageWattsByYear.map((value, index) => ({
      y: value,
      marker: index === maxAverageWattsIndex ? {enabled: true, radius: 6, fillColor: 'red'} : undefined
    }));

    (chartOptions.series[1] as SeriesColumnOptions).data = maxWattsByYear.map((value, index) => ({
      y: value,
      marker: index === maxMaxWattsIndex ? {enabled: true, radius: 6, fillColor: 'red'} : undefined
    }));

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageWattsByYear);
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
    () => props.averageWattsByYear || props.maxWattsByYear,
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
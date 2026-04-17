<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, SeriesLineOptions, SeriesOptionsType} from "highcharts";

const props = defineProps<{
  averageDistanceByYear: Record<string, number>;
  maxDistanceByYear: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Distance",
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
      text: `Distance (km)`,
    },
    labels: {
      format: "{value} km",
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
                color: any; series: { name: string }; y: number
              }
          ) {
            return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y.toFixed(0)} km`;
          }, "<b>" + this.key + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average distance",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return `${this.y.toFixed(0)} km`;
        },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum distance",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return `${this.y.toFixed(0)} km`;
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

  if (!props.averageDistanceByYear || !props.maxDistanceByYear ) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const years = Array.from(
      new Set([...Object.keys(props.averageDistanceByYear), ...Object.keys(props.maxDistanceByYear)]),
    ).sort();
    const averageDistanceByYear = years.map((year) => {
      const value = props.averageDistanceByYear[year];
      return Number.isFinite(value) ? value : 0;
    });
    const maxDistanceByYear = years.map((year) => {
      const value = props.maxDistanceByYear[year];
      return Number.isFinite(value) ? value : 0;
    });

    chartOptions.xAxis.categories = years;
    if (years.length === 0) {
      (chartOptions.series[0] as SeriesColumnOptions).data = [];
      (chartOptions.series[1] as SeriesColumnOptions).data = [];
      (chartOptions.series[2] as SeriesLineOptions).data = [];
      return;
    }

    const maxAverageDistance = Math.max(...averageDistanceByYear);
    const maxAverageDistanceIndex = averageDistanceByYear.indexOf(maxAverageDistance);

    const maxMaxDistance = Math.max(...maxDistanceByYear);
    const maxMaxDistanceIndex = maxDistanceByYear.indexOf(maxMaxDistance);

    (chartOptions.series[0] as SeriesColumnOptions).data = averageDistanceByYear.map((value, index) => {
      if (index === maxAverageDistanceIndex) {
        return {
          y: value,
          marker: { enabled: true, radius: 6, fillColor: 'red' }
        };
      } else {
        return value;
      }
    });

    (chartOptions.series[1] as SeriesColumnOptions).data = maxDistanceByYear.map((value, index) => {
      if (index === maxMaxDistanceIndex) {
        return {
          y: value,
          marker: { enabled: true, radius: 6, fillColor: 'red' }
        };
      } else {
        return { y: value };
      }
    });

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageDistanceByYear);
  }
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


watch(
    () => props.averageDistanceByYear || props.maxDistanceByYear,
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

<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type {
  SeriesColumnOptions,
  SeriesLineOptions,
  SeriesOptionsType,
} from "highcharts";

const props = defineProps<{
  elevationByYear: Record<string, number>;
  distanceByYear: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: "line",
  },
  title: {
    text: "Distance / Elevation",
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
  yAxis: [
    {
      title: {
        text: `Elevation (m)`,
      },
      labels: {
        format: "{value} m",
      },
    },
    {
      title: {
        text: "Distance (km)",
      },
      labels: {
        format: "{value} km",
      },
      opposite: true,
    },
  ],
  legend: {
    enabled: true,
  },
tooltip: {
  formatter: function (this: any): string {

    return this.points.reduce(function (
      s: any,
      point: {
        color: any;
        series: { name: string };
        y: number;
      }
    ) {
      let unit = "";
      if (point.series.name === "Elevation") {
        unit = "m";
      } else if (point.series.name === "Distance") {
        unit = "km";
      }

      return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y.toFixed(0)} ${unit}`;
    }, "<b>" + this.key + "</b>");
  },
  shared: true,
},
  series: [
    {
      name: "Elevation",
      type: "line",
      yAxis: 1,
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
      name: "Distance",
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
        enabled: false,
      },
      enableMouseTracking: false,
      data: [], // Initialize with an empty array
    },
  ] as SeriesOptionsType[],
});

function updateChartData() {
  if (!props.elevationByYear || !props.distanceByYear) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const alevationByYear = Object.values(props.elevationByYear);
    const distanceByYear = Object.values(props.distanceByYear);

    chartOptions.xAxis.categories = Object.keys(props.elevationByYear);

    const maxElevation = Math.max(...alevationByYear);
    const maxElevationIndex = alevationByYear.indexOf(maxElevation);

    const maxDistance = Math.max(...distanceByYear);
    const maxDistanceIndex = distanceByYear.indexOf(maxDistance);

    (chartOptions.series[0] as SeriesColumnOptions).data = alevationByYear.map(
      (value, index) => ({
        y: value,
        marker:
          index === maxElevationIndex
            ? { enabled: true, radius: 6, fillColor: "red" }
            : undefined,
      })
    );

    (chartOptions.series[1] as SeriesColumnOptions).data = distanceByYear.map(
      (value, index) => ({
        y: value,
        marker:
          index === maxDistanceIndex
            ? { enabled: true, radius: 6, fillColor: "red" }
            : undefined,
      })
    );

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(
      distanceByYear
    );
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

watch(() => props.elevationByYear || props.distanceByYear, updateChartData, {
  immediate: true,
});
</script>

<template>
  <div class="chart-container">
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped></style>

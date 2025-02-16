<script setup lang="ts">
import {computed, reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, SeriesLineOptions, SeriesOptionsType, YAxisOptions} from "highcharts";
import {formatSpeedWithUnit} from "@/utils/formatters";

const props = defineProps<{
  activityType: string;
  averageSpeedByYear: Record<string, number>;
  maxSpeedByYear: Record<string, number>;
}>();

const unit = computed(() => (props.activityType === "Run" ? "min/km" : "km/h"));

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Speed",
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
      text: `Speed ${unit.value}`,
    },
    labels: {
      formatter: function (this: any): string {
        if (this.isFirst) {
            return "";
          }
        return formatSpeedWithUnit(this.value, props.activityType);
      },
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
        const speed = formatSpeedWithUnit(parseFloat(point.y), props.activityType);
        return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${speed}`; },
          "<b>" + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average speed",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return formatSpeedWithUnit(this.y, props.activityType);
        },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum speed",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return formatSpeedWithUnit(this.y, props.activityType);
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
  const averageSpeedByYear = props.averageSpeedByYear;
  const maxSpeedByYear = props.maxSpeedByYear;

  if (!averageSpeedByYear || !maxSpeedByYear) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const averageSpeedByYear = Object.values(props.averageSpeedByYear);
    const maxSpeedByYear = Object.values(props.maxSpeedByYear);

    chartOptions.xAxis.categories = Object.keys(props.averageSpeedByYear);

    const maxAverageSpeed = Math.max(...averageSpeedByYear);
    const maxAverageSpeedIndex = averageSpeedByYear.indexOf(maxAverageSpeed);

    const maxMaxSpeed = Math.max(...maxSpeedByYear);
    const maxMaxSpeedIndex = maxSpeedByYear.indexOf(maxMaxSpeed);

    (chartOptions.series[0] as SeriesColumnOptions).data = averageSpeedByYear.map((value, index) => ({
      y: value,
      marker: index === maxAverageSpeedIndex ? { enabled: true, radius: 6, fillColor: 'red' } : undefined
    }));

    (chartOptions.series[1] as SeriesColumnOptions).data = maxSpeedByYear.map((value, index) => ({
      y: value,
      marker: index === maxMaxSpeedIndex ? { enabled: true, radius: 6, fillColor: 'red' } : undefined
    }));

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageSpeedByYear);
  }

  if (chartOptions.yAxis && (chartOptions.yAxis as YAxisOptions).title) {
    (chartOptions.yAxis as YAxisOptions).title!.text = `Speed ${unit.value}`;
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
    () => props.averageSpeedByYear || props.maxSpeedByYear || props.activityType,
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
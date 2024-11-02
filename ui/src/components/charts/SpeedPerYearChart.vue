<script setup lang="ts">
import {computed, reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, SeriesLineOptions, SeriesOptionsType, YAxisOptions} from "highcharts";

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
    text: "Speed per year",
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
  },
  yAxis: {
    min: 0,
    title: {
      text: `Speed ${unit.value}`,
    },
    labels: {
      formatter: function (this: any): string {
        return formatSpeedWithUnit(this.value, props.activityType);
      },
    },
  },
  legend: {
    enabled: true,
  },
  tooltip: {
    formatter: function (this: any): string {
      const speed = formatSpeedWithUnit(this.y, props.activityType);

      return this.points.reduce(function (s: string) {
        return `${s}: <span>${speed}</span>`;
      }, "<b>" + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average speed",
      type: "line",
      dataLabels: {
        enabled: true,
        formatter: function (this: any): string {
          return formatSpeed(this.y, props.activityType);
        },
        y: -10,
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum speed",
      type: "line",
      dataLabels: {
        enabled: true,
        formatter: function (this: any): string {
          return formatSpeed(this.y, props.activityType);
        },
        y: -10,
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

  const averageSpeedData = Object.values(averageSpeedByYear);
  const maxSpeedData = Object.values(maxSpeedByYear);
  chartOptions.xAxis.categories = Object.keys(averageSpeedByYear);

  if (chartOptions.series && chartOptions.series.length > 0) {

    const maxAverageSpeed = Math.max(...averageSpeedData);
    const maxAverageSpeedIndex = averageSpeedData.indexOf(maxAverageSpeed);

    const maxMaxSpeed = Math.max(...maxSpeedData);
    const maxMaxSpeedIndex = maxSpeedData.indexOf(maxMaxSpeed);

    (chartOptions.series[0] as SeriesColumnOptions).data = averageSpeedData.map((value, index) => ({
      y: value,
      marker: index === maxAverageSpeedIndex ? { enabled: true, radius: 6, fillColor: 'red' } : undefined
    }));

    (chartOptions.series[1] as SeriesColumnOptions).data = maxSpeedData.map((value, index) => ({
      y: value,
      marker: index === maxMaxSpeedIndex ? { enabled: true, radius: 6, fillColor: 'red' } : undefined
    }));

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageSpeedData);
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

/**
 * Format speed (m/s)
 */
function formatSpeed(speed: number, activityType: string): string {
  if (activityType === "Run") {
    return `${formatSeconds(1000 / speed)}`;
  } else {
    return `${(speed * 3.6).toFixed(2)}`;
  }
}

function formatSpeedWithUnit(speed: number, activityType: string): string {
  if (activityType === "Run") {
    return `${formatSeconds(1000 / speed)}/km`;
  } else {
    return `${(speed * 3.6).toFixed(2)} km/h`;
  }
}

/**
 * Format seconds to minutes and seconds
 */
function formatSeconds(seconds: number): string {
  let min = Math.floor((seconds % 3600) / 60);
  let sec = Math.floor(seconds % 60);
  const hnd = Math.floor((seconds - min * 60 - sec) * 100 + 0.5);

  if (hnd === 100) {
    sec++;
    if (sec === 60) {
      sec = 0;
      min++;
    }
  }

  return `${min}'${sec < 10 ? "0" : ""}${sec}`;
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
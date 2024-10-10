<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, YAxisOptions } from "highcharts";

const props = defineProps<{
  activityType: string;
  dataByMonths: Map<string, number>[];
}>();

const unit = computed(() => (props.activityType === "Run" ? "min/km" : "km/h"));

const chartOptions: Highcharts.Options = reactive({
  chart: { type: "column" },
  title: { text: "Average speed by months" },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        fontSize: "13px",
        fontFamily: "Verdana, sans-serif",
      },
    },
    categories: [
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December",
    ],
  },
  yAxis: {
    min: 0,
    title: {
      text: `Average speed ${unit.value}`,
    },
    labels: {
      formatter: function (this: any): string {
        return formatSpeedWithUnit(this.value, props.activityType);
      },
      
    },
  },
  legend: {
    enabled: false,
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
      name: "Distance",
      type: "column",
      colors: [
        "#9b20d9",
        "#9215ac",
        "#861ec9",
        "#7a17e6",
        "#7010f9",
        "#691af3",
        "#6225ed",
        "#5b30e7",
        "#533be1",
        "#4c46db",
        "#4551d5",
        "#3e5ccf",
      ],
      colorByPoint: true,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        color: "#FFFFFF",
        inside: true,
        verticalAlign: "top",
        formatter: function (this: any): string {
          return formatSpeed(this.y, props.activityType);
        },
        y: 10, // 10 pixels down from the top
        style: {
          fontSize: "13px",
          fontFamily: "Verdana, sans-serif",
        },
      },
      data: [], // Initialize with an empty array
    },
  ],
});

// Function to convert the array of objects to an array of numbers
function convertToNumberArray(data: Map<string, number>[]): number[] {
  return data.map((item) => Object.values(item)[0], props.activityType);
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

// Watch for changes in distanceByMonths and update the chart data
watch(
  () => props.dataByMonths,
  (newData) => {
    if (chartOptions.series && chartOptions.series.length > 0) {
      (chartOptions.series[0] as SeriesColumnOptions).data = convertToNumberArray(
        newData
      );
    }
    if (chartOptions.yAxis && (chartOptions.yAxis as YAxisOptions).title) {
      (chartOptions.yAxis as YAxisOptions).title!.text = `Average speed ${unit.value}`;
    }
  },
  { immediate: true }
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped></style>

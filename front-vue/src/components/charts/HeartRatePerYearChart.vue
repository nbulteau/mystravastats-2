<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesLineOptions, SeriesOptionsType} from "highcharts";
import { calculateTrendLineIgnoringMissing, positiveValuesForYears } from "@/utils/charts";

type HeartRatePoint = {
  y: number;
  custom?: { day: string };
  marker?: { enabled: boolean; radius: number; fillColor: string };
};

const props = withDefaults(defineProps<{
  averageHeartRateByYear: Record<string, number>;
  maxHeartRateByYear: Record<string, number>;
  maxHeartRateDateByYear?: Record<string, string>;
}>(), {
  maxHeartRateDateByYear: () => ({}),
});

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Heart rate",
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
      text: `Heart rate (bpm)`,
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
                color: any; options?: { custom?: { day?: string } }; series: { name: string }; y: string
              }
          ) {
            return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y} bpm${formatTooltipDay(point.options?.custom?.day)}`;
          },
          "<b>" + this.key + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average heart rate",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
        return `${this.y.toFixed(0)} bpm`;
      },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum heart rate",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
        return `${this.y.toFixed(0)} bpm`;
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

  if (!props.averageHeartRateByYear || !props.maxHeartRateByYear ) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const years = Array.from(
      new Set([...Object.keys(props.averageHeartRateByYear), ...Object.keys(props.maxHeartRateByYear)]),
    ).sort((left, right) => Number.parseInt(left, 10) - Number.parseInt(right, 10));
    const averageHeartRateByYear = positiveValuesForYears(props.averageHeartRateByYear, years);
    const maxHeartRateByYear = positiveValuesForYears(props.maxHeartRateByYear, years);

    chartOptions.xAxis.categories = years;

    const maxAverageHeartRateIndex = indexOfMaxValue(averageHeartRateByYear);
    const maxMaxHeartRateIndex = indexOfMaxValue(maxHeartRateByYear);

    (chartOptions.series[0] as SeriesLineOptions).data = averageHeartRateByYear.map((value, index) => {
      if (value === null) return null;
      if (index === maxAverageHeartRateIndex) {
        return {
          y: value,
          marker: { enabled: true, radius: 6, fillColor: 'red' }
        };
      } else {
        return {
          y: value
        };
      }
    });

    (chartOptions.series[1] as SeriesLineOptions).data = maxHeartRateByYear.map((value, index) => {
      if (value === null) return null;
      const year = years[index] ?? "";
      const day = props.maxHeartRateDateByYear?.[year];
      const point: HeartRatePoint = day ? { y: value, custom: { day } } : { y: value };
      if (index === maxMaxHeartRateIndex) {
        return {
          ...point,
          marker: { enabled: true, radius: 6, fillColor: 'red' }
        };
      } else {
        return point;
      }
    });

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLineIgnoringMissing(averageHeartRateByYear);
  }
}

function indexOfMaxValue(values: Array<number | null>): number {
  let bestIndex = -1;
  values.forEach((value, index) => {
    if (value !== null && (bestIndex < 0 || value > (values[bestIndex] ?? -Infinity))) {
      bestIndex = index;
    }
  });
  return bestIndex;
}

function formatTooltipDay(day: string | undefined): string {
  return day ? ` - Day: ${day}` : "";
}

watch(
    () => [
      props.averageHeartRateByYear,
      props.maxHeartRateByYear,
      props.maxHeartRateDateByYear,
    ],
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

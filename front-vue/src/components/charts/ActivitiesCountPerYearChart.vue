<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, SeriesLineOptions, SeriesOptionsType} from "highcharts";

const props = defineProps<{
  activitiesCount: Record<string, number>;
}>();

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Activities count",
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
      text: `Count`,
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
            return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y}`;
          },
          "<b>" + this.x + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Activities count",
      type: "line",
      dataLabels: {
        enabled: true,
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

  if (!props.activitiesCount) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const entries = Object.entries(props.activitiesCount).sort(([left], [right]) => left.localeCompare(right));
    const activitiesCount = entries.map(([, value]) => Number.isFinite(value) ? value : 0);
    chartOptions.xAxis.categories = entries.map(([year]) => year);

    if (activitiesCount.length === 0) {
      (chartOptions.series[0] as SeriesColumnOptions).data = [];
      (chartOptions.series[1] as SeriesLineOptions).data = [];
      return;
    }

    const maxActivitiesCount = Math.max(...activitiesCount);
    const maxActivitiesCountIndex = activitiesCount.indexOf(maxActivitiesCount);

    (chartOptions.series[0] as SeriesColumnOptions).data = activitiesCount.map((value, index) => {
      if (index === maxActivitiesCountIndex) {
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

    (chartOptions.series[1] as SeriesLineOptions).data = calculateTrendLine(activitiesCount);
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
  const denominator = n * xSquaredSum - xSum * xSum;
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
    () => props.activitiesCount,
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

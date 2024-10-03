<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesOptionsType } from "highcharts";

const props = defineProps<{
  currentYear: string;
  cumulativeDistancePerYear: Map<string, Map<string, number>>;
}>();

const chartOptions = reactive({
  chart: {
    type: 'line',
    height: '50%', // Make the chart responsive to the container's height
  },
  title: {
    text: "Cumulative distance per year",
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
      text: "Distance (km)",
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
      ) { return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${parseInt(point.y)} km`; },
        "<b>" + this.x + "</b>");
    },
    shared: true,
  },

  series: [] as SeriesOptionsType[],
});

function convertToNumberArray(data: Map<string, number>): number[] {
  const numberArray: number[] = [];
  data.forEach((value) => {
    numberArray.push(value);
  });

  return numberArray;
}

// Watch for changes in cumulativeDistancePerYear and update the chart data
watch(
  () => props.cumulativeDistancePerYear,
  (newData) => {
    const actual = new Date().getFullYear();
    let year = 2010;

    // reset all series
    chartOptions.series = [];
    do {
      const yearStr = year.toString();
      const yearData = newData.get(yearStr);
      if (yearData !== undefined) {
        const serie: SeriesOptionsType = {
          type: "line",
          name: yearStr,
          data: convertToNumberArray(yearData),
        };
        chartOptions.series.push(serie);

        // Put all the keys as category
        const categories = Array.from(yearData.keys());
        chartOptions.xAxis.categories = categories;
      }
    } while (year++ < actual);
  },
  { immediate: true }
); // Immediate to handle initial data
</script>

<template>
  <div class="chart-container">
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped>
</style>

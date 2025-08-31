<script setup lang="ts">
import {reactive, watch} from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesColumnOptions, Point} from "highcharts"; // Import Point here
import {calculateAverageLine} from "@/utils/charts";

const props = defineProps<{
  title: string;
  unit: string;
  dataByMonths: Map<string, number>[];
}>();

const chartOptions: Highcharts.Options = reactive({
  chart: {type: "column"},
  title: {text: props.title + " by months"},
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
      text: props.title + ' (' + props.unit + ')',
    },
  },
  legend: {
    enabled: false,
  },
  tooltip: {
    formatter: function (this: any): string {
      return this.points.reduce(function (s: string, point: { x: string; y: number }) {
        return `${s}: <span>${point.y.toFixed(2)} ${props.unit}</span>`;
      }, `<b>${(chartOptions.xAxis as Highcharts.XAxisOptions).categories?.[this.point.index]}</b>`);
    },
    shared: true,
  },
  series: [
    {
      name: "Distance",
      type: "column",
      colors: [
        `#FF5733`,
        `#33C4FF`,
        `#9B59B6`,
        `#E67E22`,
        `#28B463`,
        `#F39C12`,
        `#8E44AD`,
        `#1ABC9C`,
        `#2ECC71`,
        `#3498DB`,
        `#D35400`,
        `#34495E`,

      ],
      colorByPoint: true,
      groupPadding: 0,
      dataLabels: {
        enabled: true,
        rotation: -90,
        color: "#000000",
        inside: true,
        verticalAlign: "top",
        format: "{point.y:.0f}",  // No decimal
        y: 10, // 10 pixels down from the top
        style: {
          fontSize: "13px",
          //fontFamily: "Verdana, sans-serif",
        },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Average distance",
      type: "line",
      color: "red",
      dashStyle: "ShortDash",
      marker: {
        enabled: false,
      },
      enableMouseTracking: false,
      dataLabels: {
        enabled: true,
        formatter: function (this: Point) { // Use Point here
          // Display the label only for the first point
          if (this.index === 0) {
            return 'Average ' + props.title.toLowerCase() + ` by months : ${this.y ? this.y.toFixed(1) : 0} ` + props.unit;
          }
          return null;
        },
        style: {
          //fontSize: "13px",
          //fontFamily: "Verdana, sans-serif",
        },
      },
      data: [], // Initialize with an empty array
    },
  ],
});

// Function to convert the array of objects to an array of numbers
function convertToNumberArray(data: Map<string, number>[]): number[] {
  return data.map((item) => Object.values(item)[0]);
}

// Watch for changes in distanceByMonths and update the chart data
watch(
    () => props.dataByMonths,
    (newData) => {
      if (chartOptions.series && chartOptions.series.length > 0) {
        const data = convertToNumberArray(newData);
        (chartOptions.series[0] as SeriesColumnOptions).data = data;

        (chartOptions.series[1] as SeriesColumnOptions).data = calculateAverageLine(data);
      }
    },
    {immediate: true}
);
</script>

<template>
  <Chart :options="chartOptions"/>
</template>

<style scoped></style>

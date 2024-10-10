<script setup lang="ts">
import { Chart } from "highcharts-vue";
import { reactive, watch } from "vue";
import Highcharts, { type Options, type SeriesColumnOptions } from "highcharts";
import { EddingtonNumber } from "@/models/eddington-number.model";

const props = defineProps<{
  title: string;
  eddingtonNumber: EddingtonNumber;
}>();

const chartOptions: Options = reactive({
  chart: {},
  title: {
    text: "",
  },
  credits: {
    text:
      "Source: <a href='https://en.wikipedia.org/wiki/Eddington_number' target='_blank'>Eddington number</a>",
  },
  xAxis: [
    {
      categories: [],
      crosshair: true,
      labels: {
        format: "{value} km",
      },
    },
  ],
  yAxis: [
    {
      labels: {
        style: {
          color: Highcharts.getOptions().colors?.[1] as string || 'black',
        },
      },
      title: {
        text: "Times completed",
        style: {
          color: Highcharts.getOptions().colors?.[1] as string,
        },
      },
    },
  ],
  tooltip: {
    formatter: function (this: any): string {
      return this.points.reduce(function (s: string, point: { x: number; y: number }) {
        const color = (point.x === props.eddingtonNumber.eddingtonNumber && point.y >= props.eddingtonNumber.eddingtonNumber) ? "red" : "black";
        const text = (point.x > props.eddingtonNumber.eddingtonNumber) ? "<br>You need " + (point.x - point.y) + " more ride of at least " + point.x +" km to reach " + point.x : "";

        return `<span style="color: ${color}">You covered at least ${point.y} times ${point.x} kms</span>${text ? text : ""}`;
      }, "");
    },
    shared: true,
  },
  legend: {
    enabled: false,
  },
  series: [
    {
      name: "Occurence",
      type: "column",
      data: [],
    },
  ],
});

watch(
  () => props.eddingtonNumber,
  (newData) => {
    if (newData.eddingtonList) {
      // Modify the data to include color for eddington number bar
      const modifiedData = newData.eddingtonList.map((value, index) => {
        if (index + 1 === newData.eddingtonNumber && value >= newData.eddingtonNumber) {
          return { x: index + 1, y: value, color: "red" };
        }
        return { x: index + 1, y: value, color: "black" };
      });
      if (chartOptions.series && chartOptions.series[0]) {
        (chartOptions.series[0] as SeriesColumnOptions).data = modifiedData;
      }

      if (chartOptions.title) {
        chartOptions.title.text = props.title
      }
    }
  },
  { immediate: true }
); // Immediate to handle initial data
</script>

<template>
  <div class="chart-container">
    <Chart
      type="line"
      :options="chartOptions"
    />
  </div>
</template>

<style scoped></style>

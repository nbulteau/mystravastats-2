<script setup lang="ts">
import { Chart } from "highcharts-vue";
import { reactive, watch } from "vue";
import Highcharts, { type Options, type SeriesColumnOptions } from "highcharts";
import { EddingtonNumber } from "@/models/eddington-number.model";

const props = defineProps<{
  eddingtonNumber: EddingtonNumber;
}>();

const chartOptions: Options = reactive({
  chart: {},
  title: {
    text: "Eddington number",
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
        return `<span style="color: ${
          point.x >= props.eddingtonNumber.eddingtonNumber &&
          point.y >= props.eddingtonNumber.eddingtonNumber
            ? "red"
            : "black"
        }">You covered at least ${point.y} times ${point.x} kms</span>`;
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
        if (index + 1 >= newData.eddingtonNumber && value >= newData.eddingtonNumber) {
          return { x: index + 1, y: value, color: "red" };
        }
        return { x: index + 1, y: value, color: "black" };
      });
      if (chartOptions.series && chartOptions.series[0]) {
        (chartOptions.series[0] as SeriesColumnOptions).data = modifiedData;
      }

      if (chartOptions.title) {
        chartOptions.title.text = "Eddington number: " + newData.eddingtonNumber;
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

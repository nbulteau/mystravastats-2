<script setup lang="ts">
import { Chart } from "highcharts-vue";
import { reactive, ref, watch } from "vue";
import type {
  Options,
  SeriesColumnOptions,
  SeriesLineOptions,
  XAxisOptions,
  YAxisOptions,
} from "highcharts";
import { EddingtonNumber } from "@/models/eddington-number.model";
import {
  EDDINGTON_CURRENT_COLOR,
  EDDINGTON_REFERENCE_COLOR,
  buildEddingtonChartData,
  formatEddingtonTooltip,
  pluralizeEddingtonCount,
  type EddingtonChartPoint,
} from "@/utils/eddington-chart";

const props = defineProps<{
  title: string;
  eddingtonNumber: EddingtonNumber;
}>();

const hasChartData = ref(false);

const chartOptions: Options = reactive({
  chart: {
    type: "column",
    height: 360,
  },
  title: {
    text: "",
  },
  subtitle: {
    text: "",
  },
  credits: {
    text:
      "Source: <a href='https://en.wikipedia.org/wiki/Eddington_number' target='_blank'>Eddington number</a>",
  },
  xAxis: {
    min: 0,
    max: 1,
    crosshair: true,
    title: {
      text: "Distance threshold",
    },
    labels: {
      format: "{value} km",
    },
  },
  yAxis: {
    min: 0,
    max: 1,
    allowDecimals: false,
    title: {
      text: "Days at or above threshold",
    },
  },
  tooltip: {
    formatter: function (this: any): string {
      const pointOptions = this.point?.options as EddingtonChartPoint | undefined;
      if (!pointOptions) {
        return "";
      }
      return formatEddingtonTooltip(pointOptions);
    },
    shared: false,
  },
  legend: {
    enabled: true,
  },
  plotOptions: {
    column: {
      borderWidth: 0,
      pointPadding: 0.08,
      groupPadding: 0.08,
    },
  },
  series: [
    {
      name: "Recorded days",
      type: "column",
      data: [],
    },
    {
      name: "Requirement",
      type: "line",
      color: EDDINGTON_REFERENCE_COLOR,
      dashStyle: "ShortDash",
      marker: {
        enabled: false,
      },
      enableMouseTracking: false,
      data: [],
    },
  ],
});

function updatePlotLines(currentNumber: number) {
  const plotLine = currentNumber > 0
    ? [
        {
          value: currentNumber,
          color: EDDINGTON_CURRENT_COLOR,
          width: 1,
          dashStyle: "ShortDash" as const,
          zIndex: 4,
          label: {
            text: `E=${currentNumber}`,
            style: {
              color: EDDINGTON_CURRENT_COLOR,
              fontWeight: "700",
            },
          },
        },
      ]
    : [];

  (chartOptions.xAxis as XAxisOptions).plotLines = plotLine;
  (chartOptions.yAxis as YAxisOptions).plotLines = plotLine;
}

function updateChartData() {
  const chartData = buildEddingtonChartData(props.eddingtonNumber ?? {});
  hasChartData.value = chartData.hasData;
  const countPlural = pluralizeEddingtonCount(2, chartData.countSingular);

  if (chartOptions.title) {
    chartOptions.title.text = props.title;
  }
  if (chartOptions.subtitle) {
    chartOptions.subtitle.text = chartData.summary;
  }

  (chartOptions.xAxis as XAxisOptions).min = chartData.axisMin;
  (chartOptions.xAxis as XAxisOptions).max = chartData.axisMax;
  (chartOptions.xAxis as XAxisOptions).title = {
    text: `${chartData.metricLabel === "elevation" ? "Elevation" : "Distance"} threshold`,
  };
  (chartOptions.xAxis as XAxisOptions).labels = {
    format: `{value} ${chartData.unit}`,
  };
  (chartOptions.yAxis as YAxisOptions).max = chartData.yAxisMax;
  (chartOptions.yAxis as YAxisOptions).title = {
    text: `${countPlural[0].toUpperCase()}${countPlural.slice(1)} at or above threshold`,
  };
  updatePlotLines(chartData.currentNumber);

  if (chartOptions.series && chartOptions.series[0]) {
    (chartOptions.series[0] as SeriesColumnOptions).name = `Recorded ${countPlural}`;
    (chartOptions.series[0] as SeriesColumnOptions).data = chartData.points;
  }
  if (chartOptions.series && chartOptions.series[1]) {
    (chartOptions.series[1] as SeriesLineOptions).data = chartData.referenceLine;
  }
}

watch(
  () => props.eddingtonNumber,
  updateChartData,
  { immediate: true }
);

watch(
  () => props.title,
  updateChartData,
);
</script>

<template>
  <div class="chart-container">
    <div
      v-if="!hasChartData"
      class="chart-empty"
    >
      No Eddington data available for this sport filter.
    </div>
    <Chart
      v-else
      :options="chartOptions"
    />
  </div>
</template>

<style scoped></style>

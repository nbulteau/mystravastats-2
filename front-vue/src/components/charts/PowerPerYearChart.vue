<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import {Chart} from "highcharts-vue";
import type {SeriesLineOptions, SeriesOptionsType} from "highcharts";

type PowerSourceMode = "device" | "all";
type PowerPoint = { y: number; marker?: { enabled: boolean; radius: number; fillColor: string } };

function formatWatts(value: unknown): string {
  return typeof value === "number" && Number.isFinite(value)
    ? `${Math.round(value)} watts`
    : "";
}

const props = defineProps<{
  averageWattsByYear: Record<string, number>;
  maxWattsByYear: Record<string, number>;
  deviceAverageWattsByYear?: Record<string, number>;
  deviceMaxWattsByYear?: Record<string, number>;
}>();

const selectedPowerSource = ref<PowerSourceMode>("device");
const hasDevicePowerData = computed(() =>
  Object.keys(props.deviceAverageWattsByYear ?? {}).length > 0
  || Object.keys(props.deviceMaxWattsByYear ?? {}).length > 0
);
const effectivePowerSource = computed<PowerSourceMode>(() =>
  selectedPowerSource.value === "device" && hasDevicePowerData.value ? "device" : "all"
);

const chartOptions = reactive({
  chart: {
    type: 'line',
  },
  title: {
    text: "Power",
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
      text: `Power (watts)`,
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
                color: any; series: { name: string }; y: number
              }
          ) {
            return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${formatWatts(point.y)}`;
          },
          "<b>" + this.key + "</b>");
    },
    shared: true,
  },
  series: [
    {
      name: "Average watts",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return formatWatts(this.y);
      },
      },
      data: [], // Initialize with an empty array
    },
    {
      name: "Maximum watts",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return formatWatts(this.y);
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

function selectedPowerData(): {
  average: Record<string, number>;
  maximum: Record<string, number>;
} {
  if (effectivePowerSource.value === "device") {
    return {
      average: props.deviceAverageWattsByYear ?? {},
      maximum: props.deviceMaxWattsByYear ?? {},
    };
  }
  return {
    average: props.averageWattsByYear,
    maximum: props.maxWattsByYear,
  };
}

function hasPositiveValue(source: Record<string, number>, year: string): boolean {
  return Number.isFinite(source[year]) && source[year] > 0;
}

function sortedYears(average: Record<string, number>, maximum: Record<string, number>): string[] {
  return Array.from(
    new Set([...Object.keys(average ?? {}), ...Object.keys(maximum ?? {})]),
  )
    .filter((year) => hasPositiveValue(average, year) && hasPositiveValue(maximum, year))
    .sort((left, right) => Number.parseInt(left, 10) - Number.parseInt(right, 10));
}

function valuesForYears(source: Record<string, number>, years: string[]): number[] {
  return years.map((year) => source[year]);
}

function highlightedSeries(values: number[]): PowerPoint[] {
  if (values.length === 0) {
    return [];
  }
  const maximum = Math.max(...values);
  const maximumIndex = values.indexOf(maximum);
  return values.map((value, index) => {
    if (index === maximumIndex) {
      return {
        y: value,
        marker: { enabled: true, radius: 6, fillColor: "red" },
      };
    }
    return { y: value };
  });
}

function updateChartData() {

  if (!props.averageWattsByYear || !props.maxWattsByYear) {
    return;
  }

  if (chartOptions.series && chartOptions.series.length > 0) {
    const powerData = selectedPowerData();
    const years = sortedYears(powerData.average, powerData.maximum);
    const averageWattsByYear = valuesForYears(powerData.average, years);
    const maxWattsByYear = valuesForYears(powerData.maximum, years);

    chartOptions.xAxis.categories = years;

    (chartOptions.series[0] as SeriesLineOptions).name =
      effectivePowerSource.value === "device" ? "Average watts (meter)" : "Average watts";
    (chartOptions.series[1] as SeriesLineOptions).name =
      effectivePowerSource.value === "device" ? "Maximum watts (meter)" : "Maximum watts";

    (chartOptions.series[0] as SeriesLineOptions).data = highlightedSeries(averageWattsByYear);
    (chartOptions.series[1] as SeriesLineOptions).data = highlightedSeries(maxWattsByYear);

    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageWattsByYear);
  }
}

function calculateTrendLine(data: number[]): number[] {
  const n = data.length;
  if (n < 2) {
    return data;
  }
  const xSum = data.reduce((sum, _, index) => sum + index, 0);
  const ySum = data.reduce((sum, value) => sum + value, 0);
  const xySum = data.reduce((sum, value, index) => sum + index * value, 0);
  const xSquaredSum = data.reduce((sum, _, index) => sum + index * index, 0);

  const slope = (n * xySum - xSum * ySum) / (n * xSquaredSum - xSum * xSum);
  const intercept = (ySum - slope * xSum) / n;

  return data.map((_, index) => slope * index + intercept);
}


watch(
    () => [
      props.averageWattsByYear,
      props.maxWattsByYear,
      props.deviceAverageWattsByYear,
      props.deviceMaxWattsByYear,
      effectivePowerSource.value,
    ],
    updateChartData,
    {immediate: true}
);

function selectPowerSource(mode: PowerSourceMode) {
  if (mode === "device" && !hasDevicePowerData.value) {
    return;
  }
  selectedPowerSource.value = mode;
}

</script>

<template>
  <div class="chart-container power-chart">
    <div
      class="power-chart__switch"
      role="group"
      aria-label="Power source"
    >
      <button
        type="button"
        class="power-chart__button"
        :class="{ 'power-chart__button--active': effectivePowerSource === 'device' }"
        :disabled="!hasDevicePowerData"
        @click="selectPowerSource('device')"
      >
        Power meter
      </button>
      <button
        type="button"
        class="power-chart__button"
        :class="{ 'power-chart__button--active': effectivePowerSource === 'all' }"
        @click="selectPowerSource('all')"
      >
        All power
      </button>
    </div>
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped>
.power-chart {
  position: relative;
}

.power-chart__switch {
  position: absolute;
  top: 4px;
  right: 8px;
  z-index: 1;
  display: inline-flex;
  overflow: hidden;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #f8f9fc;
}

.power-chart__button {
  border: 0;
  border-right: 1px solid var(--ms-border);
  padding: 5px 10px;
  color: var(--ms-text-muted);
  background: transparent;
  font-size: 0.78rem;
  font-weight: 700;
}

.power-chart__button:last-child {
  border-right: 0;
}

.power-chart__button:disabled {
  color: #b7bbc6;
  cursor: not-allowed;
}

.power-chart__button--active {
  color: #ffffff;
  background: var(--ms-primary);
}

</style>

<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { Chart } from "highcharts-vue";
import type {
  Options,
  SeriesLineOptions,
  XAxisOptions,
  YAxisOptions,
} from "highcharts";
import { calculateTrendLine } from "@/utils/charts";

type DetailMetric = "distance" | "elevation";

const props = defineProps<{
  averageDistanceByYear: Record<string, number>;
  averageElevationByYear: Record<string, number>;
  maxDistanceByYear: Record<string, number>;
  maxElevationByYear: Record<string, number>;
}>();

const metricOptions: Array<{ value: DetailMetric; label: string }> = [
  { value: "distance", label: "Distance" },
  { value: "elevation", label: "Elevation" },
];

const selectedMetric = ref<DetailMetric>("distance");
const activeUnit = ref<"km" | "m">("km");

const chartOptions: Options = reactive({
  chart: {
    type: "line",
    height: 360,
  },
  title: {
    text: "",
  },
  subtitle: {
    text: "",
  },
  xAxis: {
    categories: [],
    crosshair: true,
    labels: {
      autoRotation: [-45],
    },
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
      const points = this.points ?? [];
      return points.reduce(
        (summary: string, point: { color: string; series: { name: string }; y: number }) => {
          return `${summary}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${formatValue(point.y, activeUnit.value)}`;
        },
        `<b>${this.key}</b>`,
      );
    },
    shared: true,
  },
  plotOptions: {
    line: {
      marker: {
        enabled: false,
      },
    },
  },
  series: [
    {
      name: "Average distance",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return formatValue(this.y, activeUnit.value);
        },
      },
      data: [],
    },
    {
      name: "Maximum distance",
      type: "line",
      dataLabels: {
        enabled: true,
        y: -10,
        formatter: function (this: any): string {
          return formatValue(this.y, activeUnit.value);
        },
      },
      data: [],
    },
    {
      name: "Average trend",
      type: "line",
      dashStyle: "ShortDash",
      marker: {
        enabled: false,
      },
      enableMouseTracking: false,
      data: [],
    },
  ],
});

function setSelectedMetric(metric: DetailMetric) {
  selectedMetric.value = metric;
}

function safeValue(value: number | undefined): number {
  return Number.isFinite(value) ? Number(value) : 0;
}

function formatValue(value: number, unit: "km" | "m"): string {
  return `${Math.round(value).toLocaleString()} ${unit}`;
}

function maxValueSummary(values: number[], years: string[], label: string, unit: "km" | "m"): string | null {
  if (values.length === 0) {
    return null;
  }

  const maxValue = Math.max(...values);
  const maxIndex = values.indexOf(maxValue);
  const year = years[maxIndex];
  if (!year || !Number.isFinite(maxValue)) {
    return null;
  }
  return `${label}: ${formatValue(maxValue, unit)} in ${year}`;
}

function highlightedPoints(values: number[]): Array<number | { y: number; marker: { enabled: true; radius: number; fillColor: string } }> {
  if (values.length === 0) {
    return [];
  }

  const maxValue = Math.max(...values);
  const maxIndex = values.indexOf(maxValue);
  return values.map((value, index) => index === maxIndex
    ? { y: value, marker: { enabled: true, radius: 6, fillColor: "#fc4c02" } }
    : value);
}

function updateChartData() {
  const isDistance = selectedMetric.value === "distance";
  const averageSource = isDistance ? props.averageDistanceByYear : props.averageElevationByYear;
  const maxSource = isDistance ? props.maxDistanceByYear : props.maxElevationByYear;
  const unit = isDistance ? "km" : "m";
  const metricLabel = isDistance ? "Distance" : "Elevation";
  const averageLabel = isDistance ? "Average distance" : "Average elevation";
  const maxLabel = isDistance ? "Maximum distance" : "Maximum elevation";
  const years = Array.from(
    new Set([...Object.keys(averageSource ?? {}), ...Object.keys(maxSource ?? {})]),
  ).sort();
  const averageValues = years.map((year) => safeValue(averageSource?.[year]));
  const maxValues = years.map((year) => safeValue(maxSource?.[year]));
  const subtitle = [
    maxValueSummary(averageValues, years, "Best average", unit),
    maxValueSummary(maxValues, years, "Best max", unit),
  ].filter((part): part is string => part !== null).join(" - ");

  activeUnit.value = unit;
  if (chartOptions.title) {
    chartOptions.title.text = `${metricLabel} details by year`;
  }
  if (chartOptions.subtitle) {
    chartOptions.subtitle.text = subtitle;
  }
  (chartOptions.xAxis as XAxisOptions).categories = years;
  (chartOptions.yAxis as YAxisOptions).title = {
    text: isDistance ? "Distance (km)" : "Elevation (m)",
  };

  if (chartOptions.series?.[0]) {
    (chartOptions.series[0] as SeriesLineOptions).name = averageLabel;
    (chartOptions.series[0] as SeriesLineOptions).data = highlightedPoints(averageValues);
  }
  if (chartOptions.series?.[1]) {
    (chartOptions.series[1] as SeriesLineOptions).name = maxLabel;
    (chartOptions.series[1] as SeriesLineOptions).data = highlightedPoints(maxValues);
  }
  if (chartOptions.series?.[2]) {
    (chartOptions.series[2] as SeriesLineOptions).data = calculateTrendLine(averageValues);
  }
}

watch(
  [
    () => props.averageDistanceByYear,
    () => props.averageElevationByYear,
    () => props.maxDistanceByYear,
    () => props.maxElevationByYear,
    selectedMetric,
  ],
  updateChartData,
  { immediate: true },
);
</script>

<template>
  <div class="distance-elevation-details">
    <div
      class="distance-elevation-details__controls"
      role="group"
      aria-label="Distance elevation details metric"
    >
      <div
        class="scope-switch"
        role="group"
        aria-label="Distance elevation metric"
      >
        <button
          v-for="option in metricOptions"
          :key="option.value"
          type="button"
          class="scope-switch__button"
          :class="{ 'scope-switch__button--active': selectedMetric === option.value }"
          :aria-pressed="selectedMetric === option.value"
          @click="setSelectedMetric(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </div>
    <div class="chart-container">
      <Chart :options="chartOptions" />
    </div>
  </div>
</template>

<style scoped>
.distance-elevation-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.distance-elevation-details__controls {
  display: flex;
  justify-content: flex-end;
}

.scope-switch {
  display: inline-flex;
  overflow: hidden;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #f8f9fc;
}

.scope-switch__button {
  border: 0;
  border-right: 1px solid var(--ms-border);
  padding: 5px 10px;
  color: var(--ms-text-muted);
  background: transparent;
  font-size: 0.78rem;
  font-weight: 700;
}

.scope-switch__button:last-child {
  border-right: 0;
}

.scope-switch__button--active {
  color: #ffffff;
  background: var(--ms-primary);
}

@media (max-width: 640px) {
  .distance-elevation-details__controls {
    justify-content: stretch;
  }

  .scope-switch {
    flex: 1;
  }

  .scope-switch__button {
    flex: 1;
  }
}
</style>

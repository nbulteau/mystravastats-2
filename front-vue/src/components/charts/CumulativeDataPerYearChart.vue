<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { Chart } from "highcharts-vue";
import type {
  DashStyleValue,
  Options,
  SeriesLineOptions,
  XAxisOptions,
  YAxisOptions,
} from "highcharts";
import {
  buildCumulativeYearChartData,
  formatCumulativeDateLabel,
  type CumulativeChartSeries,
  type CumulativeComparisonMode,
  type CumulativeMetric,
} from "@/utils/cumulative-year-chart";

const props = defineProps<{
  activityTypeLabel: string;
  cumulativeDistancePerYear: Map<string, Map<string, number>>;
  cumulativeElevationPerYear: Map<string, Map<string, number>>;
}>();

const metricOptions: Array<{ value: CumulativeMetric; label: string }> = [
  { value: "distance", label: "Distance" },
  { value: "elevation", label: "Elevation" },
];

const comparisonOptions: Array<{ value: CumulativeComparisonMode; label: string }> = [
  { value: "all-years", label: "All years" },
  { value: "best-year", label: "Best year" },
  { value: "previous-year", label: "Previous year" },
];

const HISTORY_COLORS = [
  "#6c7a89",
  "#4e79a7",
  "#59a14f",
  "#b6992d",
  "#af7aa1",
  "#17becf",
  "#9c755f",
  "#e15759",
  "#76b7b2",
  "#edc948",
  "#457b9d",
  "#2a9d8f",
  "#f28e2b",
  "#8d99ae",
];
const CURRENT_YEAR_COLOR = "#fc4c02";
const PROJECTION_COLOR = "#e76f51";

const chartMetric = ref<CumulativeMetric>("distance");
const comparisonMode = ref<CumulativeComparisonMode>("all-years");
const hasChartData = ref(false);
const activeCategories = ref<string[]>([]);
const activeUnit = ref<"km" | "m">("km");

const chartOptions: Options = reactive({
  chart: {
    type: "line",
    height: 500,
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
    plotLines: [],
    title: {
      text: "Day of year",
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
      return points.reduce(function (
        tooltip: string,
        point: {
          color: string;
          series: { name: string };
          y: number;
        },
      ) {
        const value = Number(point.y);
        const formattedValue = Number.isFinite(value)
          ? Math.round(value).toLocaleString()
          : "0";
        return `${tooltip}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${formattedValue} ${activeUnit.value}`;
      }, `<b>${formatCumulativeDateLabel(this.key)}</b>`);
    },
    shared: true,
  },
  plotOptions: {
    line: {
      marker: {
        enabled: false,
      },
    },
    series: {
      states: {
        inactive: {
          opacity: 0.35,
        },
      },
    },
  },
  series: [],
});

function setChartMetric(metric: CumulativeMetric) {
  chartMetric.value = metric;
}

function setComparisonMode(mode: CumulativeComparisonMode) {
  comparisonMode.value = mode;
}

function historyColor(year: string, index: number): string {
  const parsedYear = Number.parseInt(year, 10);
  if (Number.isFinite(parsedYear)) {
    return HISTORY_COLORS[Math.abs(parsedYear - 2010) % HISTORY_COLORS.length];
  }
  return HISTORY_COLORS[index % HISTORY_COLORS.length];
}

function highchartsSeries(series: CumulativeChartSeries, index: number): SeriesLineOptions {
  const isFocusedComparison = comparisonMode.value !== "all-years" && series.role === "history";
  const color = series.role === "current"
    ? CURRENT_YEAR_COLOR
    : series.role === "projection"
      ? PROJECTION_COLOR
      : historyColor(series.year, index);

  return {
    type: "line",
    name: series.name,
    data: series.data,
    color,
    dashStyle: series.role === "projection" ? "ShortDash" as DashStyleValue : "Solid" as DashStyleValue,
    enableMouseTracking: true,
    lineWidth: series.role === "current" || isFocusedComparison ? 3 : 2,
    opacity: series.role === "projection" ? 0.85 : 1,
    zIndex: series.role === "current" ? 5 : series.role === "projection" ? 4 : 2,
  };
}

function labelForCategoryIndex(value: number | string): string {
  const category = typeof value === "number"
    ? activeCategories.value[value]
    : String(value);
  return category ? formatCumulativeDateLabel(category).split(" ")[1] ?? category : "";
}

function monthTickPositions(categories: string[]): number[] {
  return categories
    .map((category, index) => ({ category, index }))
    .filter(({ category }) => category.endsWith("-01"))
    .map(({ index }) => index);
}

function updateChartData() {
  const chartData = buildCumulativeYearChartData({
    comparisonMode: comparisonMode.value,
    distancePerYear: props.cumulativeDistancePerYear,
    elevationPerYear: props.cumulativeElevationPerYear,
    metric: chartMetric.value,
  });

  hasChartData.value = chartData.hasData;
  activeCategories.value = chartData.categories;
  activeUnit.value = chartData.unit;

  if (chartOptions.title) {
    chartOptions.title.text = `${chartData.title} for ${props.activityTypeLabel}`;
  }
  if (chartOptions.subtitle) {
    chartOptions.subtitle.text = chartData.summary;
  }

  (chartOptions.xAxis as XAxisOptions).categories = chartData.categories;
  (chartOptions.xAxis as XAxisOptions).tickPositions = monthTickPositions(chartData.categories);
  (chartOptions.xAxis as XAxisOptions).labels = {
    autoRotation: [-45],
    formatter: function (this: { value: number | string }) {
      return labelForCategoryIndex(this.value);
    },
  };
  (chartOptions.xAxis as XAxisOptions).plotLines = chartData.todayIndex >= 0
    ? [
        {
          color: "#f39b76",
          dashStyle: "Dash" as DashStyleValue,
          label: {
            rotation: 0,
            style: {
              color: "#74402b",
              fontWeight: "700",
            },
            text: "Today",
            y: 20,
          },
          value: chartData.todayIndex,
          width: 2,
          zIndex: 4,
        },
      ]
    : [];
  (chartOptions.yAxis as YAxisOptions).title = {
    text: chartData.yAxisTitle,
  };
  chartOptions.series = chartData.series.map(highchartsSeries);
}

watch(
  [
    () => props.cumulativeDistancePerYear,
    () => props.cumulativeElevationPerYear,
    () => props.activityTypeLabel,
    chartMetric,
    comparisonMode,
  ],
  updateChartData,
  { immediate: true },
);
</script>

<template>
  <div class="cumulative-chart">
    <div
      class="cumulative-chart__controls"
      role="group"
      aria-label="Cumulative chart controls"
    >
      <div
        class="scope-switch"
        role="group"
        aria-label="Cumulative metric"
      >
        <button
          v-for="option in metricOptions"
          :key="option.value"
          type="button"
          class="scope-switch__button"
          :class="{ 'scope-switch__button--active': chartMetric === option.value }"
          :aria-pressed="chartMetric === option.value"
          @click="setChartMetric(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
      <div
        class="scope-switch"
        role="group"
        aria-label="Cumulative comparison"
      >
        <button
          v-for="option in comparisonOptions"
          :key="option.value"
          type="button"
          class="scope-switch__button"
          :class="{ 'scope-switch__button--active': comparisonMode === option.value }"
          :aria-pressed="comparisonMode === option.value"
          @click="setComparisonMode(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </div>
    <div class="chart-container">
      <div
        v-if="!hasChartData"
        class="chart-empty"
      >
        No cumulative data available for this sport filter.
      </div>
      <Chart
        v-else
        :options="chartOptions"
      />
    </div>
  </div>
</template>

<style scoped>
.cumulative-chart {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cumulative-chart__controls {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
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
  .cumulative-chart__controls {
    justify-content: stretch;
  }

  .scope-switch {
    flex: 1 1 100%;
  }

  .scope-switch__button {
    flex: 1;
  }
}
</style>

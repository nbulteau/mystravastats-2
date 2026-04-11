<script setup lang="ts">
import { computed } from "vue";
import { Chart } from "highcharts-vue";

const MONTH_NAMES = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const DAY_LABELS = Array.from({ length: 31 }, (_, i) => String(i + 1));

const props = defineProps<{
  activityHeatmap: Record<string, Record<string, number>>;
  selectedYear?: string;
}>();

const availableYears = computed(() =>
  Object.keys(props.activityHeatmap).sort((a, b) => parseInt(b, 10) - parseInt(a, 10))
);

const hasData = computed(() => availableYears.value.length > 0);

const displayYear = computed(() => {
  if (!hasData.value) {
    return "";
  }
  if (
    props.selectedYear &&
    props.selectedYear !== "All years" &&
    availableYears.value.includes(props.selectedYear)
  ) {
    return props.selectedYear;
  }
  return availableYears.value[0] ?? "";
});

const isAllYearsSelected = computed(() => props.selectedYear === "All years");
const isFallbackYear = computed(
  () =>
    !!props.selectedYear &&
    props.selectedYear !== "All years" &&
    props.selectedYear !== displayYear.value
);

function buildHeatmapData(yearData: Record<string, number>): [number, number, number][] {
  return Object.entries(yearData).map(([key, value]) => {
    const [monthStr, dayStr] = key.split("-");
    const month = parseInt(monthStr, 10) - 1;
    const day = parseInt(dayStr, 10) - 1;
    return [month, day, Math.round(value * 10) / 10];
  });
}

const chartOptions = computed((): any => {
  const year = displayYear.value;
  const yearData = year ? (props.activityHeatmap[year] ?? {}) : {};
  const data = buildHeatmapData(yearData);
  const maxValue = Math.max(...data.map((d) => d[2]), 1);

  return {
    chart: {
      type: "heatmap",
      height: 420,
      marginTop: 50,
      marginBottom: 60,
    },
    title: {
      text: `Activity heatmap${year ? " - " + year : ""}`,
    },
    xAxis: {
      categories: MONTH_NAMES,
      labels: { style: { fontSize: "11px" } },
    },
    yAxis: {
      title: { text: null },
      categories: DAY_LABELS,
      reversed: false,
    },
    colorAxis: {
      min: 0,
      max: maxValue,
      stops: [
        [0, "#ebedf0"],
        [0.1, "#9be9a8"],
        [0.4, "#40c463"],
        [0.7, "#30a14e"],
        [1, "#216e39"],
      ],
    },
    legend: {
      align: "right",
      layout: "vertical",
      verticalAlign: "middle",
      symbolHeight: 180,
      title: { text: "km" },
    },
    tooltip: {
      formatter: function (this: any): string {
        const day = parseInt(this.point.y, 10) + 1;
        const month = MONTH_NAMES[this.point.x];
        const value = this.point.value as number;
        if (!value) return `<b>${month} ${day}</b>: no activity`;
        return `<b>${month} ${day}</b>: ${value.toFixed(1)} km`;
      },
    },
    series: [
      {
        type: "heatmap",
        name: "Distance (km)",
        borderWidth: 2,
        borderColor: "#ffffff",
        data,
        dataLabels: { enabled: false },
      },
    ],
    credits: { enabled: false },
  };
});
</script>

<template>
  <div class="heatmap-wrapper">
    <div v-if="!hasData" class="chart-empty">
      Activity heatmap — no data available
    </div>

    <template v-else>
      <div v-if="isAllYearsSelected" class="heatmap-note">
        All years selected: showing latest available year ({{ displayYear }}).
      </div>
      <div v-else-if="isFallbackYear" class="heatmap-note">
        No heatmap data for {{ selectedYear }}. Showing {{ displayYear }}.
      </div>

      <div class="heatmap-chart">
        <Chart :options="chartOptions" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.heatmap-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.heatmap-note {
  color: #4c617b;
  font-size: 0.88rem;
  background: #f3f8ff;
  border: 1px solid #d9e6f5;
  border-radius: 8px;
  padding: 0.45rem 0.65rem;
}

.heatmap-chart {
  width: 100%;
}
</style>

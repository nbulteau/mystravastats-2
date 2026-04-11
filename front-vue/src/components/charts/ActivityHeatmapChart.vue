<script setup lang="ts">
import { computed, ref, watch } from "vue";
import Highcharts from 'highcharts';
// Highcharts v11+ modules auto-register when imported
import 'highcharts/modules/heatmap';
import { Chart } from "highcharts-vue";
import type { Options } from "highcharts";


const MONTH_NAMES = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const DAY_LABELS  = Array.from({ length: 31 }, (_, i) => String(i + 1));

const props = defineProps<{
  activityHeatmap: Record<string, Record<string, number>>;
}>();

// Sorted list of years present in the data (most recent first)
const availableYears = computed(() =>
  Object.keys(props.activityHeatmap).sort((a, b) => parseInt(b) - parseInt(a))
);

// Year currently shown in the chart
const selectedYear = ref<string>('');

// Reset selection when the data changes (e.g. activity-type switch)
watch(availableYears, (years) => {
  if (!selectedYear.value || !years.includes(selectedYear.value)) {
    selectedYear.value = years[0] ?? '';
  }
}, { immediate: true });

// Convert "MM-DD → km" map to Highcharts [month, day, value] triples
function buildHeatmapData(yearData: Record<string, number>): [number, number, number][] {
  return Object.entries(yearData).map(([key, value]) => {
    const [monthStr, dayStr] = key.split('-');
    const month = parseInt(monthStr) - 1; // 0-indexed (Jan = 0)
    const day   = parseInt(dayStr)   - 1; // 0-indexed (1st = 0)
    return [month, day, Math.round(value * 10) / 10];
  });
}

const chartOptions = computed<Options>(() => {
  const year     = selectedYear.value;
  const yearData = year ? (props.activityHeatmap[year] ?? {}) : {};
  const data     = buildHeatmapData(yearData);
  const maxValue = Math.max(...data.map(d => d[2]), 1);

  return {
    chart: {
      type: 'heatmap',
      marginTop: 50,
      marginBottom: 80,
    },
    title: {
      text: `Activity heatmap${year ? ' – ' + year : ''}`,
    },
    xAxis: {
      // Months on the horizontal axis
      categories: MONTH_NAMES,
      opposite: false,
      labels: { style: { fontSize: '11px' } },
    },
    yAxis: {
      // Day-of-month on the vertical axis (1 at the top)
      title: { text: null as any },
      categories: DAY_LABELS,
      reversed: false,
    },
    colorAxis: {
      min: 0,
      max: maxValue,
      stops: [
        [0,   '#ebedf0'],  // No activity – light grey
        [0.1, '#9be9a8'],  // Low activity – light green
        [0.4, '#40c463'],  // Moderate
        [0.7, '#30a14e'],  // High
        [1,   '#216e39'],  // Peak – dark green
      ],
    },
    legend: {
      align: 'right',
      layout: 'vertical',
      verticalAlign: 'middle',
      symbolHeight: 180,
      title: { text: 'km' },
    },
    tooltip: {
      formatter: function (this: any): string {
        const day   = parseInt(this.point.y) + 1;
        const month = MONTH_NAMES[this.point.x];
        const value = this.point.value as number;
        if (!value) return `<b>${month} ${day}</b>: no activity`;
        return `<b>${month} ${day}</b>: ${value.toFixed(1)} km`;
      },
    },
    series: [{
      type: 'heatmap',
      name: 'Distance (km)',
      borderWidth: 2,
      borderColor: '#ffffff',
      data,
      dataLabels: { enabled: false },
    }],
    credits: { enabled: false },
  };
});
</script>

<template>
  <div class="heatmap-wrapper">
    <!-- Year selector -->
    <div class="year-selector" v-if="availableYears.length > 1">
      <span class="year-label">Year</span>
      <div class="year-tabs">
        <button
          v-for="year in availableYears"
          :key="year"
          :class="['year-tab', { active: year === selectedYear }]"
          @click="selectedYear = year"
        >
          {{ year }}
        </button>
      </div>
    </div>

    <div class="chart-container">
      <Chart :options="chartOptions" />
    </div>
  </div>
</template>

<style scoped>
.heatmap-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.year-selector {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.year-label {
  font-size: 0.8rem;
  color: #4c617b;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.year-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
}

.year-tab {
  padding: 0.2rem 0.7rem;
  border: 1px solid #c8d5e3;
  border-radius: 4px;
  background: #f5f8fb;
  font-size: 0.82rem;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
  color: #4c617b;
}

.year-tab:hover {
  background: #e2ecf6;
  border-color: #9bb5ce;
}

.year-tab.active {
  background: #2e7eed;
  border-color: #2e7eed;
  color: #ffffff;
  font-weight: 600;
}
</style>


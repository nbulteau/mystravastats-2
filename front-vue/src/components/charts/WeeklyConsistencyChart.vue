<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { PointOptionsObject, SeriesPieOptions } from "highcharts";
import type { ChartPeriodPoint } from "@/models/chart-period-point.model";
import { extractPeriodEntries, weeksInIsoYear } from "@/utils/charts";

const props = defineProps<{
  distanceByWeeks: ChartPeriodPoint[];
  selectedYear: string;
}>();

type WeeklyConsistencyData = {
  activeWeeks: number;
  inactiveWeeks: number;
  consistencyPct: number;
};

function computeConsistencyData(): WeeklyConsistencyData {
  const entries = extractPeriodEntries(props.distanceByWeeks);
  const selectedYear = Number.parseInt(props.selectedYear, 10);
  const expectedWeeks = Number.isNaN(selectedYear) ? Math.max(entries.length, 52) : weeksInIsoYear(selectedYear);

  const activeWeeks = entries.filter((entry) => entry.activityCount > 0).length;
  const inactiveWeeks = Math.max(0, expectedWeeks - activeWeeks);
  const consistencyPct = expectedWeeks > 0 ? (activeWeeks / expectedWeeks) * 100 : 0;

  return {
    activeWeeks,
    inactiveWeeks,
    consistencyPct,
  };
}

const chartOptions: Highcharts.Options = reactive({
  chart: {
    type: "pie",
  },
  title: {
    text: "Weekly consistency",
  },
  subtitle: {
    text: "",
  },
  tooltip: {
    pointFormat: "<b>{point.y}</b> week(s) · <b>{point.percentage:.1f}%</b>",
  },
  plotOptions: {
    pie: {
      innerSize: "60%",
      dataLabels: {
        enabled: true,
        format: "<b>{point.name}</b>: {point.y}",
      },
    },
  },
  series: [
    {
      type: "pie",
      name: "Weeks",
      data: [],
    },
  ],
});

function updateChart() {
  const consistency = computeConsistencyData();
  const data: PointOptionsObject[] = [
    {
      name: "Active weeks",
      y: consistency.activeWeeks,
      color: "#37a845",
    },
    {
      name: "Inactive weeks",
      y: consistency.inactiveWeeks,
      color: "#d9dee7",
    },
  ];

  (chartOptions.series?.[0] as SeriesPieOptions).data = data;
  chartOptions.subtitle = {
    text: `${consistency.activeWeeks} active week(s) / ${consistency.activeWeeks + consistency.inactiveWeeks} · ${consistency.consistencyPct.toFixed(1)}% consistency`,
  };
}

watch(
  () => [props.distanceByWeeks, props.selectedYear],
  updateChart,
  { immediate: true, deep: true },
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>

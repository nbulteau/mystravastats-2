<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions, SeriesLineOptions, XAxisOptions, YAxisOptions } from "highcharts";
import type { HeartRateZonePeriodSummary } from "@/models/heart-rate-zone.model";

const props = defineProps<{
  byMonth: HeartRateZonePeriodSummary[];
  selectedYear: string;
}>();

const MONTH_LABELS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

type MonthlyEasyHardData = {
  easyHours: number[];
  hardHours: number[];
  ratio: Array<number | null>;
  totalEasyHours: number;
  totalHardHours: number;
};

function computeMonthlyData(): MonthlyEasyHardData {
  const easyHours = Array<number>(12).fill(0);
  const hardHours = Array<number>(12).fill(0);
  const ratio = Array<number | null>(12).fill(null);

  const selectedYear = Number.parseInt(props.selectedYear, 10);
  const hasYearFilter = !Number.isNaN(selectedYear);

  for (const period of props.byMonth) {
    const [yearPart, monthPart] = period.period.split("-");
    const year = Number.parseInt(yearPart, 10);
    const month = Number.parseInt(monthPart, 10);
    if (Number.isNaN(month) || month < 1 || month > 12) {
      continue;
    }
    if (hasYearFilter && year !== selectedYear) {
      continue;
    }

    const index = month - 1;
    easyHours[index] += period.easySeconds / 3600;
    hardHours[index] += period.hardSeconds / 3600;
    ratio[index] = period.easyHardRatio ?? null;
  }

  return {
    easyHours: easyHours.map((value) => Math.round(value * 10) / 10),
    hardHours: hardHours.map((value) => Math.round(value * 10) / 10),
    ratio: ratio.map((value) => (value === null ? null : Math.round(value * 100) / 100)),
    totalEasyHours: easyHours.reduce((sum, value) => sum + value, 0),
    totalHardHours: hardHours.reduce((sum, value) => sum + value, 0),
  };
}

const chartOptions: Highcharts.Options = reactive({
  chart: {
    type: "column",
  },
  title: {
    text: "Easy / Hard ratio by month",
  },
  subtitle: {
    text: "",
  },
  xAxis: {
    categories: MONTH_LABELS,
  },
  yAxis: [
    {
      min: 0,
      title: {
        text: "Tracked time (hours)",
      },
    },
    {
      min: 0,
      title: {
        text: "Easy/Hard ratio",
      },
      opposite: true,
    },
  ],
  tooltip: {
    shared: true,
    formatter: function (this: any): string {
      const details = this.points.map((point: any) => {
        if (point.series.name === "Easy/Hard ratio") {
          return `<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y?.toFixed?.(2) ?? point.y}`;
        }
        return `<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${point.y?.toFixed?.(1) ?? point.y} h`;
      });
      return `<b>${this.x}</b>${details.join("")}`;
    },
  },
  plotOptions: {
    column: {
      stacking: "normal",
    },
  },
  series: [
    {
      name: "Easy time",
      type: "column",
      color: "#37a845",
      data: [],
    },
    {
      name: "Hard time",
      type: "column",
      color: "#fc4c02",
      data: [],
    },
    {
      name: "Easy/Hard ratio",
      type: "line",
      color: "#2f6df6",
      yAxis: 1,
      marker: { enabled: true, radius: 3 },
      data: [],
    },
  ],
});

function updateChart() {
  const data = computeMonthlyData();
  (chartOptions.xAxis as XAxisOptions).categories = MONTH_LABELS;
  (chartOptions.yAxis as YAxisOptions[])[1].max = undefined;
  (chartOptions.series?.[0] as SeriesColumnOptions).data = data.easyHours;
  (chartOptions.series?.[1] as SeriesColumnOptions).data = data.hardHours;
  (chartOptions.series?.[2] as SeriesLineOptions).data = data.ratio;

  const combinedRatio = data.totalHardHours > 0 ? data.totalEasyHours / data.totalHardHours : null;
  chartOptions.subtitle = {
    text:
      data.totalEasyHours > 0 || data.totalHardHours > 0
        ? `Easy ${data.totalEasyHours.toFixed(1)} h · Hard ${data.totalHardHours.toFixed(1)} h · Ratio ${combinedRatio === null ? "N/A" : combinedRatio.toFixed(2)}`
        : "No heart-rate zone data available by month.",
  };
}

watch(
  () => [props.byMonth, props.selectedYear],
  updateChart,
  { immediate: true, deep: true },
);
</script>

<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
</style>

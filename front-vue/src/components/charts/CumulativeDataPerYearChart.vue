<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesOptionsType, DashStyleValue } from "highcharts";

const props = defineProps<{
  cumulativeDistancePerYear: Map<string, Map<string, number>>;
  cumulativeElevationPerYear: Map<string, Map<string, number>>;
}>();

const actual = new Date().getFullYear();
const currentDate = new Date();
const currentMonthDay = `${String(currentDate.getMonth() + 1).padStart(2, "0")}-${String(
  currentDate.getDate()
).padStart(2, "0")}`;

const today = computed(() => {
  const currentYearData = props.cumulativeDistancePerYear.get(actual.toString());
  if (currentYearData) {
    const keysArray = Array.from(currentYearData.keys());

    return keysArray.indexOf(currentMonthDay);
  }
  return -1; // Return a default value if currentYearData is not found
});

const monthNames = [
        'January', 'February', 'March', 'April', 'May', 'June',
        'July', 'August', 'September', 'October', 'November', 'December'
      ];

function formatTooltip(dateString: string): string {
      const [month, day] = dateString.split('-').map(Number);
      return month && day
        ? `${day} ${monthNames[month - 1]}`
        : dateString;
    }

const chartOptions = reactive({
  chart: {
    type: "line",
    height: "50%", // Make the chart responsive to the container's height
  },
  title: {
    text: "Cumulative distance per year",
  },
  xAxis: {
    labels: {
      autoRotation: [-45, -90],
      style: {
        fontSize: "13px",
        fontFamily: "Verdana, sans-serif",
      },
    },
    plotLines: [
      {
        color: "#f39b76",
        width: 2,
        value: today,
        zIndex: 2,
        dashStyle: "Dash" as DashStyleValue,
        label: {
          text: "Today",
          rotation: 0,
          y: 20,
          style: {
            color: "#74402b",
          },
        },
      },
    ],
    categories: [] as string[],
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
      const unit = chartType.value === "distance" ? "km" : "m";
      return this.points.reduce(function (
        s: any,
        point: {
          color: any;
          series: { name: string };
          y: number;
        }
      ) {
        const value = Number(point.y);
        const formattedValue = Number.isFinite(value) ? Math.round(value).toLocaleString() : "0";
        return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${formattedValue} ${unit}`;
      }, "<b>" + formatTooltip(this.key) + "</b>");
    },
    shared: true,
  },

  series: [] as SeriesOptionsType[],
});

function convertToNumberArray(data: Map<string, number>): number[] {
  const numberArray: number[] = [];
  data.forEach((value) => {
    numberArray.push(value);
  });

  return numberArray;
}

function toggleChartType() {
  chartType.value = chartType.value === "distance" ? "elevation" : "distance";
}

const chartType = ref<"distance" | "elevation">("distance");

const title = computed(() => `Cumulative ${chartType.value} per year`);

const YEAR_COLORS = [
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
  "#e76f51",
  "#8d99ae",
  "#ff7f0e",
  "#fc4c02",
];

function getYearColor(year: number): string {
  const paletteIndex = Math.max(0, Math.min(year - 2010, YEAR_COLORS.length - 1));
  return YEAR_COLORS[paletteIndex] ?? "#fc4c02";
}

function updateChartData() {
  const data =
    chartType.value === "distance"
      ? props.cumulativeDistancePerYear
      : props.cumulativeElevationPerYear;


  // reset all series
  chartOptions.series = [];

  let year = 2010;
  do {
    const yearStr = year.toString();
    const yearData = data.get(yearStr);
    if (yearData !== undefined) {
      let series: SeriesOptionsType;
      if (year === actual) {
        series = {
          type: "line",
          name: yearStr,
          data: convertToNumberArray(yearData),
          color: getYearColor(year),
          lineWidth: 3,
          zoneAxis: "x",
          zones: [
            {
              value: today,
            },
            {
              dashStyle: "Dot",
              color: "#fc4c02",
            },
          ],
        } as SeriesOptionsType;
      } else {
        series = {
          type: "line",
          name: yearStr,
          data: convertToNumberArray(yearData),
          color: getYearColor(year),
          lineWidth: 2,
          zones: [],
        } as SeriesOptionsType;
      }

      chartOptions.series.push(series);

      // Put all the keys as category
      chartOptions.xAxis.categories = Array.from(yearData.keys());

      chartOptions.yAxis.title.text = chartType.value === "distance" ? "Distance (km)" : "Elevation (m)";
    }
  } while (year++ < actual);

  if (chartOptions.title) {
    chartOptions.title.text = title.value;
  }
}

watch(
    () => props.cumulativeDistancePerYear,
    updateChartData,
    { immediate: true }
);

watch(
    () => chartType.value,
    updateChartData,
    { immediate: true }
);
</script>

<template>
  <div class="button-container">
    <button 
      class="btn btn-primary"
      @click="toggleChartType"
    >
      Switch to {{ chartType === "distance" ? "Elevation" : "Distance" }}
    </button>
  </div>
  <div class="chart-container">
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped>
.button-container {
  display: flex;
  justify-content: center;
  margin-bottom: 1rem; /* Optional: Adds some space below the button */
}
</style>

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

let today = computed(() => {
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
      return `${day} ${monthNames[month - 1]}`;
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
        color: "#4840d6",
        width: 2,
        value: today,
        zIndex: 2,
        dashStyle: "Dash" as DashStyleValue,
        label: {
          text: "Current time",
          rotation: 0,
          y: 20,
          style: {
            color: "#333333",
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
      return this.points.reduce(function (
        s: any,
        point: {
          color: any;
          series: { name: string };
          y: string;
        }
      ) {
        return `${s}<br/><span style="color:${point.color}">\u25CF</span> ${point.series.name}: ${parseInt(point.y)} km`;
      }, "<b>" + formatTooltip(this.x) + "</b>");
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

const yearColors: { [key: string]: string } = {
  "2010": "#FF0000", // Red
  "2011": "#00FF00", // Green
  "2012": "#0000FF", // Blue
  "2013": "#FFFF00", // Yellow
  "2014": "#FF00FF", // Magenta
  "2015": "#00FFFF", // Cyan
  "2016": "#800000", // Maroon
  "2017": "#808000", // Olive
  "2018": "#008000", // Dark Green
  "2019": "#800080", // Purple
  "2020": "#008080", // Teal
  "2021": "#000080", // Navy
  "2022": "#FFC0CB", // Pink
  "2023": "#A52A2A", // Brown
  "2024": "#FFA500", // Orange
  "2025": "#808080", // Gray
};

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
          color: yearColors[year] || "#000000", // Default to black if color not defined
          zoneAxis: "x",
          zones: [
            {
              value: today,
            },
            {
              dashStyle: "Dot",
            },
          ],
        } as SeriesOptionsType;
      } else {
        series = {
          type: "line",
          name: yearStr,
          data: convertToNumberArray(yearData),
          color: yearColors[year] || "#000000", // Default to black if color not defined
          zones: [],
        } as SeriesOptionsType;
      }

      chartOptions.series.push(series);

      // Put all the keys as category
      chartOptions.xAxis.categories = Array.from(yearData.keys());
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

<template>
    <!-- Back arrow button at the top left -->
  <button
    class="back-arrow"
    @click="goBack"
    aria-label="Back to activities"
    title="Back to activities"
  >
    <!-- SVG left arrow icon -->
    <svg width="32" height="32" viewBox="0 0 24 24" aria-hidden="true">
      <path d="M15 19l-7-7 7-7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
    </svg>
    
  </button>
  <div
    v-if="loadError"
    class="alert alert-danger"
    role="alert"
  >
    {{ loadError }}
  </div>
  <template v-else>
  <div
    id="title"
    style="text-align: center; margin-bottom: 20px"
  >
    <span style="display: block; font-size: 1.5em; font-weight: bold">
      {{ activity?.name }}
    </span>
    <span style="display: block; font-size: 1.2em">
      Distance: {{ ((activity?.distance ?? 0) / 1000).toFixed(1) }} km |
      <template v-if="isHike">
        Total elevation gain: {{ activity?.totalElevationGain.toFixed(0) }} m | Total
        Descent: {{ activity?.totalDescent.toFixed(0) }} m |
      </template>
      <template v-else>
        Average speed:
        {{ formatSpeedWithUnit(activity?.averageSpeed ?? 0, activity?.type ?? "Ride") }} |
        Total elevation gain: {{ activity?.totalElevationGain.toFixed(0) }} m |
      </template>
      Elapsed time: {{ formatTime(activity?.elapsedTime ?? 0) }}
    </span>
    <span class="strava-link-row">
      <a
        :href="stravaActivityUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="strava-link"
      >
        Open on Strava
      </a>
    </span>
  </div>

  <div style="display: flex; width: 100%; height: 400px">
    <div
      id="map-container"
      style="width: 80%; height: 100%"
    />
    <div
      id="radio-container"
      class="radio-scroll-container"
      style="width: 20%; height: 100%; padding-left: 10px"
    >
      <form>
        <div
          v-for="option in radioOptions"
          :key="option.value"
        >
          <input
            :id="option.value"
            v-model="selectedOption"
            type="radio"
            :value="option.value"
            class="radio-input"
            @click="handleRadioClick(option.value)"
          >
          <label
            ref="radioLabels"
            :for="option.value"
            class="radio-label"
            :title="option.description"
          >{{ option.label }}</label>
        </div>
      </form>
    </div>
  </div>

  <div
    v-if="hasPowerData"
    class="switch-container"
  >
    <switch-button
      v-model="showPowerCurve"
      button-text="Show Power curve"
    />
  </div>
  <div
    id="chart-container"
    style="width: 100%; height: 400px"
  >
    <Chart :options="chartOptions" />
  </div>

  <div v-if="hasPowerData">
    <PowerDistributionChart
      v-if="activity"
      :activity="activity"
    />
  </div>

  <div v-if="hasPowerData">
    <PowerCurveDetailsChart
      v-if="activity"
      :activity="activity"
      :historical-data="[]"
      :weight="85"
      :display-in-watts-per-kg="true"
    />
  </div>
  <ActivityMetrics
    v-if="activity"
    :activity="activity"
  />
  </template>
</template>

<script setup lang="ts">
import "bootstrap";
import "bootstrap/dist/css/bootstrap.min.css";
import { Tooltip } from "bootstrap"; // Import Bootstrap Tooltip
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ActivityEffort, DetailedActivity } from "@/models/activity.model"; 
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import { useContextStore } from "@/stores/context.js";
import { useAthleteStore } from "@/stores/athlete";
import { ErrorService } from "@/services/error.service";
import type { Options, SeriesAreaOptions, SeriesLineOptions } from "highcharts";
import Highcharts from "highcharts";
import { Chart } from "highcharts-vue";
import ActivityMetrics from "@/components/ActivityMetrics.vue";
import PowerDistributionChart from "@/components/charts/PowerDistributionChart.vue";
import SwitchButton from '@/components/SwitchButton.vue';
import PowerCurveDetailsChart from "@/components/charts/PowerCurveDetailsChart.vue";

// Import the leaflet library
import "leaflet/dist/leaflet.css";
import "leaflet-defaulticon-compatibility/dist/leaflet-defaulticon-compatibility.css"; 
import L from "leaflet";
import "leaflet-defaulticon-compatibility";

import { useRouter } from "vue-router"; // Import useRouter from vue-router

const router = useRouter(); // Get router instance

// Function to go back to the previous page
function goBack() {
  router.back();
}

import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';

L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

const showPowerCurve = ref(false);

const hasPowerData = computed(() => {
  const powerData = activity.value?.stream?.watts;

  return powerData && powerData.length > 0;
});

const contextStore = useContextStore();
const athleteStore = useAthleteStore();
contextStore.updateCurrentView("activity");

const route = useRoute();

const activityId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;


const activity = ref<DetailedActivity | null>(null);
const loadError = ref<string | null>(null);

const map = ref<L.Map>();
const basePolyline = ref<L.Polyline | null>(null);
const selectedPolyline = ref<L.Polyline | null>(null);
const hoverMarker = ref<L.Marker | null>(null);
const lastHoveredPointIndex = ref<number | null>(null);

const radioOptions = ref<{ label: string; value: string; description: string }[]>([]);

const selectedOption = ref(null);

const radioLabels = ref<HTMLElement[]>([]); // Ref to hold radio labels

const isHike = computed(() => activity.value?.type === "Hike");
const stravaActivityUrl = computed(() => `https://www.strava.com/activities/${activity.value?.id ?? activityId ?? ""}`);

// Icon options

const buildRadioOptions = () => {
  if (activity.value?.activityEfforts) {
    radioOptions.value = activity.value?.activityEfforts.map((effort) => {
      return {
        label:
          effort.label.length > 20 ? effort.label.substring(0, 20) + "..." : effort.label,
        value: effort.id,
        description: effort?.description ?? "",
      };
    });
  } else {
    radioOptions.value = [];
  }
};

async function fetchDetailedActivity(id: string) {
  const url = `/api/activities/${id}`;
  const response = await fetch(url);
  if (!response.ok) {
    await ErrorService.catchError(response);
  }
  activity.value = await response.json();
  loadError.value = null;
}

const initMap = () => {
  const mapContainer = document.getElementById("map-container");
  if (mapContainer) {
    map.value = L.map(mapContainer);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
    }).addTo(map.value);
  }
};

const updateMap = () => {
  if (map.value) {
    const latlngs = activity.value?.stream?.latlng?.map((latlng: number[]) =>
      (typeof latlng[0] === "number" && typeof latlng[1] === "number")
        ? L.latLng(latlng[0], latlng[1])
        : null
    );

    if (latlngs) {
      const filteredLatlngs = latlngs.filter((latlng): latlng is L.LatLng => latlng !== null);
      if (filteredLatlngs.length === 0) {
        if (basePolyline.value) {
          basePolyline.value.remove();
          basePolyline.value = null;
        }
        return;
      }

      if (basePolyline.value) {
        basePolyline.value.setLatLngs(filteredLatlngs);
      } else {
        basePolyline.value = L.polyline(filteredLatlngs, { color: "red" }).addTo(map.value);
      }

      const bounds = L.latLngBounds(filteredLatlngs);
      if (bounds.isValid()) {
        map.value.fitBounds(bounds);
      }
    }
  }
};

const chartOptions: Options = reactive({
  chart: {
    renderTo: 'chart-container',
  },
  title: {
    text: "",
  },
  credits: {
    text:
      "Source: <a href='https://en.wikipedia.org/wiki/Eddington_number' target='_blank'>Eddington number</a>",
  },
  xAxis: [
    {
      categories: [],
      crosshair: true,
      allowDecimals: false,
      labels: {
        format: "{value} km",
      },
    },
  ],
  yAxis: [
    {
      title: {
        text: "Speed",
      },
      allowDecimals: false,
      labels: {
        formatter: function (this: any): string {
          if (this.isFirst) {
            return "";
          }
          return formatSpeedWithUnit(this.value, activity.value?.type ?? "Ride");
        },
        style: {
          color: (Highcharts.getOptions().colors?.[0] as string) ?? "#000000",
        },
      },
    },
    {
      title: {
        text: "Altitude",
      },
      labels: {
        format: "{value} m",
        style: {
          color: "#d3d3d3",
        },
      },
      opposite: true,
    },
    {
      title: {
        text: "",
      },
      labels: {
        format: "{value} watts",
        style: {
          color: "red",
        },
      },
    },
  ],
  tooltip: {
    formatter: function (this: any): string {
      const altitude = this.points?.[1]?.y ?? 0;
      const velocityMS =
        activity.value?.stream?.velocitySmooth?.[this.points?.[0]?.point?.index ?? 0];
      const velocity = formatSpeedWithUnit(
        velocityMS ?? 0,
        activity.value?.type ?? "Ride"
      );

      return (
        "Distance: " +
        this.point.x.toFixed(1) +
        " km<br/>Speed: <b>" +
        velocity +
        "</b></br>Altitude: " +
        altitude.toFixed(0) +
        " m"
      );
    },
    shared: true,
  },
  legend: {
    enabled: false,
  },
  series: [
    {
      name: "Speed",
      type: "line",
      data: [],
    },
    {
      name: "Altitude",
      type: "area",
      data: [],
      color: "#d3d3d3",
      yAxis: 1,
    },
    {
      name: "",
      type: "area",
      data: [],
      color: "blue",
      yAxis: 1,
    },
    {
      name: "Power Curve",
      type: "line",
      data: [],
      color: "red",
    },
  ],
});

let chartInstance: Highcharts.Chart | null = null;
let chartMouseMoveHandler: ((e: MouseEvent) => void) | null = null;

const initChart = () => {
  const altitudeStream = activity.value?.stream?.altitude;
  const distanceStream = activity.value?.stream?.distance;
  const speedStream = activity.value?.stream?.velocitySmooth;

  if (
    speedStream &&
    distanceStream &&
    chartOptions.series &&
    chartOptions.series.length > 0
  ) {
    (chartOptions.series[0] as SeriesLineOptions).data = speedStream.map(
      (speed, index) => ({
        x: (distanceStream?.[index] ?? 0) / 1000,
        y: speed,
      })
    );
  }

  if (
    altitudeStream &&
    distanceStream &&
    chartOptions.series &&
    chartOptions.series.length > 0
  ) {
    (chartOptions.series[1] as SeriesAreaOptions).data = altitudeStream.map(
      (altitude, index) => ({
        x: (distanceStream?.[index] ?? 0) / 1000,
        y: altitude,
      })
    );

    // Calculate the minimum altitude value
    const minAltitude = Math.min(...altitudeStream);
    const yAxisMin = minAltitude * 0.95; // Set the min property of the altitude yAxis

    // Set the min property of the altitude yAxis
    if (Array.isArray(chartOptions.yAxis) && chartOptions.yAxis[1]) {
      chartOptions.yAxis[1].min = yAxisMin;
    }
  }

  const chartContainer = document.getElementById("chart-container");
  if (chartContainer) {
    chartInstance = Highcharts.chart(chartContainer, chartOptions);

    chartMouseMoveHandler = (e: MouseEvent) => {
      if (!chartInstance || !map.value) return;

      const event: Highcharts.PointerEventObject = chartInstance.pointer.normalize(e);
      let point: Highcharts.Point | undefined = undefined;
      if (chartInstance.series[0]) {
        point = chartInstance.series[0].searchPoint(event, true);
      }

      if (point) {
        if (lastHoveredPointIndex.value === point.index) {
          return;
        }

        const latlng = activity.value?.stream?.latlng?.[point.index];
        if (
          latlng &&
          Array.isArray(latlng) &&
          typeof latlng[0] === "number" &&
          typeof latlng[1] === "number"
        ) {
          const nextLatLng = L.latLng(latlng[0], latlng[1]);
          if (hoverMarker.value) {
            hoverMarker.value.setLatLng(nextLatLng);
          } else if (map.value) {
            hoverMarker.value = L.marker(nextLatLng).addTo(map.value);
          }
          lastHoveredPointIndex.value = point.index;
        }
      }
    };
    chartContainer.addEventListener("mousemove", chartMouseMoveHandler);
  }
};

const handleRadioClick = (key: string) => {
  console.log(`Radio button with value ${key} clicked`);

  // 1 - Get the selected effort
  const selectedEffort: ActivityEffort | undefined = activity.value?.activityEfforts.find(
    (effort) => effort.id === key
  );
  if (!selectedEffort) {
    console.error(`No effort found for value: ${key}`);
    return;
  }

  // 2 - Get the stream data for the selected effort
  const stream = activity.value?.stream;
  if (!stream) {
    console.error(`No stream data found for effort: ${selectedEffort}`);
    return;
  }
  const startIndex = selectedEffort.idxStart;
  const endIndex = selectedEffort.idxEnd;
  const selectedStream = {
    latitudeLongitude: stream.latlng ? stream.latlng.slice(startIndex, endIndex) : [],
    altitude: stream.altitude ? stream.altitude.slice(startIndex, endIndex) : [],
    distance: stream.distance.slice(startIndex, endIndex),
    time: stream.time.slice(startIndex, endIndex),
  };

  if (map.value) {
    const latlngs = selectedStream.latitudeLongitude
      .map((latlng: number[]) =>
        typeof latlng[0] === "number" && typeof latlng[1] === "number"
          ? L.latLng(latlng[0], latlng[1])
          : null
      )
      .filter((latlng): latlng is L.LatLng => latlng !== null);

    if (latlngs.length > 0) {
      if (selectedPolyline.value) {
        selectedPolyline.value.setLatLngs(latlngs);
      } else {
        selectedPolyline.value = L.polyline(latlngs, { color: "blue" }).addTo(map.value);
      }

      const bounds = L.latLngBounds(latlngs);
      if (bounds.isValid()) {
        map.value.fitBounds(bounds);
      }
    } else if (selectedPolyline.value) {
      selectedPolyline.value.remove();
      selectedPolyline.value = null;
    }

  }

// 4 - Update the chart with the new stream data
  if (
      selectedStream.altitude &&
      selectedStream.distance &&
      chartOptions.series &&
      chartOptions.series.length > 0
  ) {
    (chartOptions.series[2] as SeriesAreaOptions).data = selectedStream.altitude.map(
        (altitude, index) => ({
          x: (selectedStream.distance?.[index] ?? 0) / 1000,
          y: altitude,
          color: "blue",
        })
    );

    // Forcer la mise à jour du graphique
    if (chartInstance) {
      chartInstance.update({
        series: chartOptions.series,
      });
    }
  }
};

onMounted(async () => {
  initMap();
  try {
    await athleteStore.fetchHeartRateZoneSettings().catch(() => undefined);
    await fetchDetailedActivity(activityId ?? "");
    updateMap();
    initChart();
    buildRadioOptions();

    // Ensure DOM is updated before initializing tooltips
    await nextTick();

    // Initialize tooltips for radio labels
    radioLabels.value.forEach((label) => {
      new Tooltip(label, {
        title: label.getAttribute("title") || "",
        html: true,
      });
    });
  } catch (error) {
    activity.value = null;
    loadError.value = "Unable to load this activity.";
  }
});

onBeforeUnmount(() => {
  const chartContainer = document.getElementById("chart-container");
  if (chartContainer && chartMouseMoveHandler) {
    chartContainer.removeEventListener("mousemove", chartMouseMoveHandler);
  }
  if (chartInstance) {
    chartInstance.destroy();
    chartInstance = null;
  }
  if (map.value) {
    map.value.remove();
    map.value = undefined;
  }
  basePolyline.value = null;
  selectedPolyline.value = null;
  hoverMarker.value = null;
  lastHoveredPointIndex.value = null;
});

// Watcher to update chart options when showPowerCurve changes
watch([showPowerCurve, activity], () => {
  if (chartOptions.series) {
    if (showPowerCurve.value && hasPowerData.value) {
      chartOptions.series[3] = {
        name: "Power Curve",
        type: "line",
        data: (activity.value?.stream?.watts ?? []).map((watts, index) => ({
          x: (activity.value?.stream?.distance?.[index] ?? 0) / 1000,
          y: watts,
        })),
        color: "red",
        yAxis: 2,
      };
    } else {
      chartOptions.series[3] = {
        name: "Power Curve",
        type: "line",
        data: [],
        color: "red",
      };
    }
    if (chartInstance) {
      chartInstance.update({
        series: chartOptions.series,
      });
    }
  }
});

</script>

<style scoped>
.radio-input {
  margin-right: 5px;
}

.strava-link-row {
  display: block;
  margin-top: 8px;
}

.strava-link {
  color: var(--ms-primary);
  font-weight: 600;
  text-decoration: underline;
}

.radio-label {
  font-weight: bold;
  text-align: left;
  /* Align text to the left */
}

.tooltip-inner ul,
.tooltip-inner li {
  text-align: left; /* Ensure list items are left-aligned */
}

/* Scrollable container for radio buttons */
.radio-scroll-container {
  height: 100%; /* Set height to 100% to match the map container */
  overflow-y: auto; /* Enable vertical scrolling */
}

.switch-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}

/* Style for the back arrow button */
.back-arrow {
  position: absolute;
  top: 20px;
  left: 20px;
  font-size: 2rem;
  background: none;
  border: none;
  cursor: pointer;
  color: #333;
  z-index: 10;
  transition: color 0.2s;
}
.back-arrow:hover {
  color: #007bff;
}
</style>

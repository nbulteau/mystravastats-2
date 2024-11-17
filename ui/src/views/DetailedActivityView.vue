<template>
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
          >{{ option.label
          }}</label>
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

<script setup lang="ts">
import "bootstrap";
import "bootstrap/dist/css/bootstrap.min.css";
import { Tooltip } from "bootstrap"; // Import Bootstrap Tooltip
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ActivityEffort, DetailedActivity } from "@/models/activity.model"; 
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import { useContextStore } from "@/stores/context.js";
import type { Options, SeriesAreaOptions, SeriesLineOptions } from "highcharts";
import Highcharts from "highcharts";
import { Chart } from "highcharts-vue";
import ActivityMetrics from "@/components/ActivityMetrics.vue";
import PowerDistributionChart from "@/components/charts/PowerDistributionChart.vue";
import SwitchButton from '@/components/SwitchButton.vue';

// Import the leaflet library
import "leaflet/dist/leaflet.css";
import "leaflet-defaulticon-compatibility/dist/leaflet-defaulticon-compatibility.css"; 
import L from "leaflet";
import "leaflet-defaulticon-compatibility";
import PowerCurveDetailsChart from "@/components/charts/PowerCurveDetailsChart.vue";

// Set the default icon options
L.Icon.Default.mergeOptions({
  iconRetinaUrl: "/markers/marker-icon-2x.png",
  iconUrl: "/markers/marker-icon.png",
  shadowUrl: "/markers/marker-shadow.png",
});

const showPowerCurve = ref(false);

const hasPowerData = computed(() => {
  const powerData = activity.value?.stream?.watts;

  return powerData && powerData.length > 0;
});

const contextStore = useContextStore();
contextStore.updateCurrentView("activity");

const route = useRoute();

const activityId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;


const activity = ref<DetailedActivity | null>(null);

const map = ref<L.Map>();

const radioOptions = ref<{ label: string; value: string; description: string }[]>([]);

const selectedOption = ref(null);

const radioLabels = ref<HTMLElement[]>([]); // Ref to hold radio labels

const isHike = computed(() => activity.value?.type === "Hike");

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
  const url = `http://localhost:8080/api/activities/${id}`;
  activity.value = await fetch(url).then((response) => response.json());
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
    // Add a polyline
    const latlngs = activity.value?.stream?.latlng?.map((latlng: number[]) =>
      L.latLng(latlng[0], latlng[1])
    );

    if (latlngs) {
      const polyline = L.polyline(latlngs, { color: "red" }).addTo(map.value);
      // Fit the map to the bounds of all polylines
      const bounds = L.latLngBounds(polyline.getLatLngs() as L.LatLng[]);
      map.value.fitBounds(bounds);
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
    formatter: function (this: Highcharts.TooltipFormatterContextObject): string {
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
        x: distanceStream[index] / 1000,
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
        x: distanceStream[index] / 1000,
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
    chartContainer.addEventListener("mousemove", function (e: MouseEvent) {
      const chart: Highcharts.Chart | undefined = Highcharts.charts[0]; // Get the chart instance from the global array of charts

      if (chart) {
        // Find coordinates within the chart
        const event: Highcharts.PointerEventObject = chart.pointer.normalize(e);
        // Get the hovered point
        const point: Highcharts.Point | undefined = chart.series[0].searchPoint(event, true);

        if (point) {
          const mapContainer = document.getElementById("map-container");
          if (mapContainer) {
            const latlng = activity.value?.stream?.latlng?.[point.index];
            if (latlng) {
              // Remove previous marker
              map.value?.eachLayer((layer) => {
                if (layer instanceof L.Marker) {
                  map.value?.removeLayer(layer);
                }
              });
              // Add a marker
              L.marker(L.latLng(latlng[0], latlng[1])).addTo(map.value!);
            }
          }
        }
      }
    });
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

  // 3 - Update the map with the new stream data
  // Remove previous blue polyline
  map.value?.eachLayer((layer) => {
    if (layer instanceof L.Polyline && layer.options.color === "blue") {
      map.value?.removeLayer(layer);
    }
  });

  if (map.value) {
    const latlngs = selectedStream.latitudeLongitude.map((latlng: number[]) =>
      L.latLng(latlng[0], latlng[1])
    );
    if (latlngs) {
      const polyline = L.polyline(latlngs, { color: "blue" }).addTo(map.value);
      // Fit the map to the bounds of all polylines
      const bounds = L.latLngBounds(polyline.getLatLngs() as L.LatLng[]);
      map.value.fitBounds(bounds);
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
        x: selectedStream.distance[index] / 1000,
        y: altitude,
        color: "blue",
      })
    );
  }
};

onMounted(async () => {
  initMap();
  await fetchDetailedActivity(activityId).then(async () => {
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
  });
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
  }
}); 

</script>

<style scoped>
.chart-container {
  margin-top: 20px 0;
}

.radio-input {
  margin-right: 5px;
}

.radio-label {
  font-weight: bold;
  text-align: left;
  /* Align text to the left */
}

.tooltip-inner {
  --bs-tooltip-max-width: 300px;
  /* Define the custom property for max-width */
  max-width: var(--bs-tooltip-max-width);
  /* Apply the custom property */
  background-color: #343a40;
  /* Dark background color */
  color: #ffffff;
  /* White text color */
  font-size: 1rem;
  /* Increase font size */
  padding: 10px;
  /* Add padding */
  border-radius: 5px;
  /* Rounded corners */
}

.tooltip-inner ul,
.tooltip-inner li {
  text-align: left;
  /* Ensure list items are left-aligned */
}

/* Scrollable container for radio buttons */
.radio-scroll-container {
  height: 100%;
  /* Set height to 100% to match the map container */
  overflow-y: auto;
  /* Enable vertical scrolling */
}

.switch-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  /* Adjust as needed */
}

.switch-button {
  /* Example styles */
  width: 50px;
  height: 25px;
  background-color: #ccc;
  border-radius: 25px;
  position: relative;
  cursor: pointer;
}

.switch-button::before {
  content: '';
  position: absolute;
  width: 23px;
  height: 23px;
  background-color: white;
  border-radius: 50%;
  top: 1px;
  left: 1px;
  transition: transform 0.3s;
}

.switch-button.on {
  background-color: #4caf50;
}

.switch-button.on::before {
  transform: translateX(25px);
}
</style>

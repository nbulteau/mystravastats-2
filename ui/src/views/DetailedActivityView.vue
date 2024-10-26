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
          >{{ option.label }}</label>
        </div>
      </form>
    </div>
  </div>
  <div
    id="chart-container"
    style="width: 100%; height: 400px"
  >
    <Chart :options="chartOptions" />
  </div>
  <div id="activity-details-container">
    <div id="activity-details">
      <h3>Basic Information</h3>
      <ul>
        <li><strong>Type:</strong> {{ activity?.type }}</li>
        <li>
          <strong>Date:</strong>
          {{
            activity?.startDate
              ? new Date(activity.startDate).toLocaleDateString("en-GB", {
                weekday: "long",
                day: "numeric",
                month: "long",
                year: "numeric",
              })
              : "N/A"
          }}
        </li>
        <li>
          <strong>Distance:</strong>
          {{ ((activity?.distance ?? 0) / 1000).toFixed(1) }} km
        </li>
        <li>
          <strong>Total Elevation Gain:</strong>
          {{ activity?.totalElevationGain.toFixed(0) }} m
        </li>
        <li>
          <strong>Elapsed Time:</strong> {{ formatTime(activity?.elapsedTime ?? 0) }}
        </li>
        <li><strong>Moving Time:</strong> {{ formatTime(activity?.movingTime ?? 0) }}</li>
      </ul>
    </div>
    <div id="performance-metrics">
      <h3>Performance Metrics</h3>
      <ul>
        <li>
          <strong>Average Speed:</strong>
          {{ formatSpeedWithUnit(activity?.averageSpeed ?? 0, activity?.type ?? "Ride") }}
        </li>
        <li>
          <strong>Max Speed:</strong>
          {{ formatSpeedWithUnit(activity?.maxSpeed ?? 0, activity?.type ?? "Ride") }}
        </li>
        <li v-if="(activity?.averageCadence ?? 0) > 0.0">
          <strong>Average Cadence:</strong>
          {{ (activity?.averageCadence ?? 0).toFixed(0) }} rpm
        </li>
        <li v-if="(activity?.averageWatts ?? 0) > 0">
          <strong>Average Watts:</strong> {{ (activity?.averageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity?.weightedAverageWatts ?? 0) > 0">
          <strong>Weighted Average Watts:</strong>
          {{ (activity?.weightedAverageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity?.kilojoules ?? 0) > 0">
          <strong>Kilojoules:</strong> {{ (activity?.kilojoules ?? 0).toFixed(0) }} kJ
        </li>
      </ul>
    </div>
    <div id="heart-rate-metrics">
      <h3>Heart Rate Metrics</h3>
      <ul>
        <li v-if="(activity?.averageHeartrate ?? 0) > 0">
          <strong>Average Heartrate:</strong>
          {{ activity?.averageHeartrate.toFixed(0) }} bpm
        </li>
        <li v-if="(activity?.maxHeartrate ?? 0) > 0">
          <strong>Max Heartrate:</strong> {{ activity?.maxHeartrate.toFixed(0) }} bpm
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import "bootstrap";
import 'leaflet/dist/leaflet.css';
import { onMounted, ref, reactive, nextTick, computed } from "vue";
import { useRoute } from "vue-router";
import L from "leaflet";
import { ActivityEffort, DetailedActivity } from "@/models/activity.model"; // Ensure correct import
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import "bootstrap/dist/css/bootstrap.min.css";
import { useContextStore } from "@/stores/context.js";
import { type Options, type SeriesAreaOptions, type SeriesLineOptions } from "highcharts";
import Highcharts from "highcharts";
import { Chart } from "highcharts-vue";
import { Tooltip } from 'bootstrap'; // Import Bootstrap Tooltip

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
L.Icon.Default.imagePath = '/node_modules/leaflet/dist/images/';

const buildRadioOptions = () => {
  if (activity.value?.activityEfforts) {
    radioOptions.value = activity.value?.activityEfforts.map((effort) => {
      return {
      label: effort.label.length > 20 ? effort.label.substring(0, 20) + '...' : effort.label,
      value: effort.id, 
      description: effort?.description ?? ''
      };
    });
  } else {
    radioOptions.value = [];
  }
};


async function fetchDetailedActivity(id: string) {
  const url = `http://localhost:8080/api/activities/${id}`;
  const detailedActivity = await fetch(url).then((response) => response.json());

  activity.value = detailedActivity;
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
  chart: {},
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
          return formatSpeedWithUnit(this.value, activity.value?.type ?? "Ride");
        },
        style: {
          color: Highcharts.getOptions().colors?.[0] as string ?? '#000000'
        }
      },
    },
    {
      title: {
        text: "Altitude",
      },
      labels: {
        format: '{value} m',
        style: {
          color: "#d3d3d3"
        }
      },
      opposite: true
    },
  ],
  tooltip: {
    formatter: function (this: Highcharts.TooltipFormatterContextObject): string {
      const altitude = this.points?.[1]?.y ?? 0;
      const velocityMS = activity.value?.stream?.velocitySmooth?.[this.points?.[0]?.point?.index ?? 0];
      const velocity = formatSpeedWithUnit(velocityMS ?? 0, activity.value?.type ?? "Ride");

      return "Distance: " + (this.point.x).toFixed(1) + " km<br/>Speed: <b>" + velocity + "</b></br>Altitude: " + altitude.toFixed(0) + " m";
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
      name: "Altitude",
      type: "area",
      data: [],
      color: "blue",
      yAxis: 1,
    },
  ],
});

const initChart = () => {
  const altitudeStream = activity.value?.stream?.altitude;
  const distanceStream = activity.value?.stream?.distance;
  const speedStream = activity.value?.stream?.velocitySmooth;

  if (speedStream && distanceStream && chartOptions.series && chartOptions.series.length > 0) {
    (chartOptions.series[0] as SeriesLineOptions).data = speedStream.map((speed, index) => (
      {
        x: distanceStream[index] / 1000,
        y: speed
      }
    ));
  }

  if (altitudeStream && distanceStream && chartOptions.series && chartOptions.series.length > 0) {
    (chartOptions.series[1] as SeriesAreaOptions).data = altitudeStream.map((altitude, index) => (
      {
        x: distanceStream[index] / 1000,
        y: altitude
      }
    ));

    // Calculate the minimum altitude value
    const minAltitude = Math.min(...altitudeStream);
    const yAxisMin = minAltitude * 0.95; // Set the min property of the altitude yAxis

    // Set the min property of the altitude yAxis
    if (Array.isArray(chartOptions.yAxis) && chartOptions.yAxis[1]) {
      chartOptions.yAxis[1].min = yAxisMin;
    }
  }

  const chartContainer = document.getElementById('chart-container');
  if (chartContainer) {
    chartContainer.addEventListener('mousemove',
      function (e: MouseEvent) {
        let chart: Highcharts.Chart | undefined;
        let point, i, event;

        for (i = 0; i < Highcharts.charts.length; i = i + 1) {
          chart = Highcharts.charts[i];
          if (chart) {
            // Find coordinates within the chart
            event = chart.pointer.normalize(e);
            // Get the hovered point
            point = chart.series[0].searchPoint(event, true);

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
                  L.marker(L.latLng(latlng[0], latlng[1]), ).addTo(map.value!);
                }
              }
            }
          }
        }
      }
    );
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
    time: stream.time.slice(startIndex, endIndex)
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
  if(selectedStream.altitude && selectedStream.distance && chartOptions.series && chartOptions.series.length > 0) {
    (chartOptions.series[2] as SeriesAreaOptions).data = selectedStream.altitude.map((altitude, index) => (
      {
        x: selectedStream.distance[index] / 1000,
        y: altitude,
        color: "blue"
      }
    ));
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
    radioLabels.value.forEach(label => {
      new Tooltip(label, {
        title: label.getAttribute('title') || '',
        html: true
      });
    });
  });
});
</script>

<style scoped>
#chart-container {
  margin-top: 20px;
}

#map {
  width: 100%;
  height: 100%;
  border-radius: 10px;
  /* Example of custom styling */
}

#activity-details-container {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

#activity-details,
#performance-metrics,
#heart-rate-metrics {
  width: 32%;
}

.radio-input {
  margin-right: 5px;
}

.radio-label {
  font-weight: bold;
    text-align: left; /* Align text to the left */

}

/* Custom Tooltip Styles */
.tooltip-inner {
  text-align: left; /* Align text to the left */
  --bs-tooltip-max-width: 500px;
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
  text-align: left; /* Ensure list items are left-aligned */
}

/* Scrollable container for radio buttons */
.radio-scroll-container {
  height: 100%; /* Set height to 100% to match the map container */
  overflow-y: auto; /* Enable vertical scrolling */
}
</style>
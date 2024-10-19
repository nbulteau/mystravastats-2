<template>
  <div
    id="title"
    style="text-align: center; margin-bottom: 20px"
  >
    <span style="display: block; font-size: 1.5em; font-weight: bold">
      {{ activity?.name }}
    </span>
    <span style="display: block; font-size: 1.2em">
      Distance: {{ ((activity?.distance ?? 0) / 1000).toFixed(1) }} km 
      | Average speed: {{ formatSpeedWithUnit(activity?.averageSpeed ?? 0, activity?.type ?? "Ride") }} 
      | Elapsed time: {{ formatTime(activity?.elapsedTime ?? 0) }} 
      | Total elevation gain: {{ activity?.totalElevationGain.toFixed(0) }} m
    </span>
  </div>
  <div style="display: flex; width: 100%; height: 400px">
    <div
      id="map-container"
      style="width: 80%; height: 100%"
    />
    <div
      id="radio-container"
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
            @click="handleRadioClick(option.label)"
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
        <li><strong>Date:</strong> {{ activity?.startDate ? new Date(activity.startDate).toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long', year:'numeric' }) : 'N/A' }}</li>
        <li><strong>Distance:</strong> {{ ((activity?.distance ?? 0) / 1000).toFixed(1) }} km</li>
        <li><strong>Total Elevation Gain:</strong> {{ activity?.totalElevationGain.toFixed(0) }} m</li>
        <li><strong>Elapsed Time:</strong> {{ formatTime(activity?.elapsedTime ?? 0) }}</li>
        <li><strong>Moving Time:</strong> {{ formatTime(activity?.movingTime ?? 0) }}</li>
      </ul>
    </div>
    <div id="performance-metrics">
      <h3>Performance Metrics</h3>
      <ul>
        <li><strong>Average Speed:</strong> {{ formatSpeedWithUnit(activity?.averageSpeed ?? 0, activity?.type ?? "Ride") }}</li>
        <li><strong>Max Speed:</strong> {{ formatSpeedWithUnit(activity?.maxSpeed ?? 0, activity?.type ?? "Ride") }}</li>
        <li v-if="(activity?.averageCadence ?? 0) > 0.0">
          <strong>Average Cadence:</strong> {{ (activity?.averageCadence ?? 0).toFixed(0) }} rpm
        </li>
        <li v-if="(activity?.averageWatts ?? 0) > 0">
          <strong>Average Watts:</strong> {{ (activity?.averageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity?.weightedAverageWatts ?? 0) > 0">
          <strong>Weighted Average Watts:</strong> {{ (activity?.weightedAverageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity?.kilojoules ?? 0) > 0">
          <strong>Kilojoules:</strong> {{ (activity?.kilojoules ?? 0).toFixed(0) }} kJ
        </li>
      </ul>
    </div>
    <div id="heart-rate-metrics">
      <h3>Heart Rate Metrics</h3>
      <ul>
        <li><strong>Average Heartrate:</strong> {{ activity?.averageHeartrate.toFixed(0) }} bpm</li>
        <li><strong>Max Heartrate:</strong> {{ activity?.maxHeartrate.toFixed(0) }} bpm</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import "bootstrap";
import { onMounted, ref, reactive, nextTick } from "vue";
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

const buildRadioOptions = () => {
  if (activity.value?.activityEfforts) {
    radioOptions.value = activity.value?.activityEfforts.map(effort => {
      return {
        label: effort.key,
        value: effort.key,
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
    const latlngs = activity.value?.stream?.latitudeLongitude?.map((latlng: number[]) =>
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
        const speedMS = this.points?.[0]?.y ?? 0;
        const speed = formatSpeedWithUnit(speedMS, activity.value?.type ?? "Ride");

        const altitude = this.points?.[1]?.y ?? 0;
        return "Distance: " + (this.point.x).toFixed(1) + " km<br/>Speed: <b>" + speed + "</b></br>Altitude: " + altitude.toFixed(0) + " m";
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
  ],
});

const initChart = () => {
    const altitudeStream = activity.value?.stream?.altitude;
    const distanceStream = activity.value?.stream?.distance;
    const timeStream = activity.value?.stream?.time;

    if (timeStream && distanceStream && chartOptions.series && chartOptions.series.length > 0) {
      const smoothedSpeedData = timeStream.map((_, index) => {
        if (index === 0) {
          return {
            x: distanceStream[index] / 1000,
            y: 0
          };
        } else {
          const deltaTime = timeStream[index] - timeStream[index - 1];
          const deltaDistance = distanceStream[index] - distanceStream[index - 1];
          const instantSpeed = deltaDistance / deltaTime;
          return {
            x: distanceStream[index] / 1000,
            y: instantSpeed
          };
        }
      });

      // Apply a simple moving average to smooth the speed data
      const windowSize = 20;
      const smoothedData = smoothedSpeedData.map((point, index, array) => {
        const start = Math.max(0, index - windowSize + 1);
        const end = index + 1;
        const window = array.slice(start, end);
        const averageY = window.reduce((sum, p) => sum + p.y, 0) / window.length;
        return {
          x: point.x,
          y: averageY
        };
      });

      (chartOptions.series[0] as SeriesLineOptions).data = smoothedData;
    }

    if (altitudeStream && distanceStream && chartOptions.series && chartOptions.series.length > 0) {
      (chartOptions.series[1] as SeriesAreaOptions).data = altitudeStream.map((altitude, index) => (
        {
            x: distanceStream[index] / 1000,
            y: altitude
        }
      ));
    }

    const chartContainer = document.getElementById('chart-container');
    if(chartContainer) {
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
                            const latlng = activity.value?.stream?.latitudeLongitude?.[point.index];
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
            }
        }
    );
    }
};

const handleRadioClick = (key: string) => {
  console.log(`Radio button with value ${key} clicked`);

  // 1 - Get the selected effort
  const selectedEffort: ActivityEffort | undefined = activity.value?.activityEfforts.find(
    (effort) => effort.key === key
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
    latitudeLongitude: stream.latitudeLongitude ? stream.latitudeLongitude.slice(startIndex, endIndex) : [],
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
  border-radius: 10px; /* Example of custom styling */
}

#activity-details-container {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

#activity-details, #performance-metrics, #heart-rate-metrics {
  width: 32%;
}

.radio-input {
  margin-right: 5px;
}

.radio-label {
  font-weight: bold;
}

/* Custom Tooltip Styles */
.tooltip-inner {
  --bs-tooltip-max-width: 300px; /* Define the custom property for max-width */
  max-width: var(--bs-tooltip-max-width); /* Apply the custom property */
  background-color: #343a40; /* Dark background color */
  color: #ffffff; /* White text color */
  font-size: 1rem; /* Increase font size */
  padding: 10px; /* Add padding */
  border-radius: 5px; /* Rounded corners */
}
</style>
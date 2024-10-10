<script setup lang="ts">
import "bootstrap";
import { onMounted, ref, reactive } from "vue";
import { useRoute } from "vue-router";
import L from "leaflet";
import { DetailedActivity } from "@/models/activity.model"; // Ensure correct import
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import "bootstrap/dist/css/bootstrap.min.css";
import { useContextStore } from "@/stores/context.js";
import { type Options, type SeriesAreaOptions, type SeriesLineOptions } from "highcharts";
import Highcharts from "highcharts";
import { Chart } from "highcharts-vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("activity");

const route = useRoute();

const activityId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;

const activity = ref<DetailedActivity | null>(null);

const map = ref<L.Map>();

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



onMounted(async () => {
  initMap();
  fetchDetailedActivity(activityId).then(() => {
    updateMap();
    initChart();
  });
});
</script>

<template>
  <div
    id="title"
    style="text-align: center; margin-bottom: 20px"
  >
    <span style="display: block; font-size: 1.5em; font-weight: bold">
      {{ activity?.name }}
    </span>
    <span style="display: block; font-size: 1.2em">
      {{ (activity?.distance ?? 0) / 1000 }} km /
      {{ formatSpeedWithUnit(activity?.averageSpeed ?? 0, activity?.type ?? "Ride") }} /
      {{ formatTime(activity?.elapsedTime ?? 0) }} / {{ activity?.totalElevationGain }} m
    </span>
  </div>
  <div
    id="map-container"
    style="width: 100%; height: 400px"
  />

  <div
    id="chart-container"
    style="width: 100%; height: 400px"
  >
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped>
#chart-container {
  margin-top: 20px;
}

#map {
  width: 100%;
  height: 100%;
  border-radius: 10px; /* Example of custom styling */
}

#activity-details {
  margin-top: 20px;
}
</style>

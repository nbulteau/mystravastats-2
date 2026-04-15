<template>
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
    class="alert alert-danger detail-alert"
    role="alert"
  >
    {{ loadError }}
  </div>
  <template v-else>
    <div class="detail-view">
      <div
        v-if="loadWarning"
        class="alert alert-warning detail-alert detail-alert--warning"
        role="alert"
      >
        {{ loadWarning }}
      </div>
      <section class="detail-hero">
        <div class="detail-hero__content">
          <p class="detail-hero__kicker">{{ activityTypeLabel }}</p>
          <h1 class="detail-hero__title">{{ activity?.name }}</h1>
          <div class="detail-hero__meta">
            <span class="detail-chip">{{ activityDateLabel }}</span>
            <span v-if="activity?.commute" class="detail-chip">Commute</span>
            <span class="detail-chip">{{ effortCountLabel }}</span>
          </div>
        </div>
        <div class="detail-hero__actions">
          <a
            :href="stravaActivityUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-primary detail-btn-strava"
          >
            Open on Strava
          </a>
        </div>
      </section>

      <section class="detail-kpi-grid">
        <article
          v-for="kpi in kpis"
          :key="kpi.label"
          class="detail-kpi-card"
        >
          <span class="detail-kpi-card__label">{{ kpi.label }}</span>
          <strong class="detail-kpi-card__value">{{ kpi.value }}</strong>
          <small v-if="kpi.hint" class="detail-kpi-card__hint">{{ kpi.hint }}</small>
        </article>
      </section>

      <section
        v-if="highlights.length > 0"
        class="detail-highlights"
      >
        <article
          v-for="highlight in highlights"
          :key="highlight.title"
          class="detail-highlight-card"
        >
          <span class="detail-highlight-card__label">{{ highlight.title }}</span>
          <strong class="detail-highlight-card__value">{{ highlight.value }}</strong>
          <small class="detail-highlight-card__hint">{{ highlight.subtitle }}</small>
        </article>
      </section>

      <section class="detail-map-layout">
        <article class="detail-card detail-card--map">
          <header class="detail-card__header">
            <h2>Route map</h2>
          </header>
          <div id="map-container" class="detail-map" />
        </article>

        <aside class="detail-card detail-card--efforts">
          <header class="detail-card__header">
            <h2>Efforts in this activity</h2>
          </header>
          <div v-if="selectedEffort && selectedEffortSummary" class="selected-effort-panel">
            <div class="selected-effort-panel__header">
              <strong>Selected effort</strong>
              <button
                type="button"
                class="btn btn-sm btn-outline-secondary"
                @click="clearSelectedEffort"
              >
                Clear
              </button>
            </div>
            <p class="selected-effort-panel__title">{{ selectedEffort.label }}</p>
            <div class="selected-effort-panel__metrics">
              <span>{{ selectedEffortSummary.duration }}</span>
              <span>{{ selectedEffortSummary.distance }}</span>
              <span>{{ selectedEffortSummary.speed }}</span>
              <span>{{ selectedEffortSummary.gradient }}</span>
              <span>{{ selectedEffortSummary.elevation }}</span>
            </div>
          </div>
          <div id="radio-container" class="radio-scroll-container">
            <form>
              <div
                v-for="option in radioOptions"
                :key="option.value"
                class="effort-option"
                :class="{ 'effort-option--active': selectedOption === option.value }"
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
                  :class="{ 'radio-label--active': selectedOption === option.value }"
                  :title="option.description"
                >{{ option.label }}</label>
              </div>
            </form>
          </div>
        </aside>
      </section>

      <section class="detail-card detail-card--chart">
        <header class="detail-card__header detail-card__header--chart">
          <h2>Elevation and speed profile</h2>
          <div class="detail-card__header-actions">
            <span v-if="selectedEffort" class="detail-chip detail-chip--active">
              {{ selectedEffort.label }}
            </span>
            <div v-if="hasPowerData" class="switch-container">
              <switch-button
                v-model="showPowerCurve"
                button-text="Show Power curve"
              />
            </div>
          </div>
        </header>
        <div id="chart-container" class="detail-chart">
          <Chart :options="chartOptions" />
        </div>
      </section>

      <section v-if="hasPowerData" class="detail-card">
        <PowerDistributionChart
          v-if="activity"
          :activity="activity"
        />
      </section>

      <section v-if="hasPowerData" class="detail-card">
        <PowerCurveDetailsChart
          v-if="activity"
          :activity="activity"
          :historical-data="[]"
          :weight="85"
          :display-in-watts-per-kg="true"
        />
      </section>

      <section class="detail-card">
        <ActivityMetrics
          v-if="activity"
          :activity="activity"
        />
      </section>
    </div>
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
const loadWarning = ref<string | null>(null);

const map = ref<L.Map>();
const basePolyline = ref<L.Polyline | null>(null);
const selectedPolyline = ref<L.Polyline | null>(null);
const hoverMarker = ref<L.Marker | null>(null);
const lastHoveredPointIndex = ref<number | null>(null);

const radioOptions = ref<{ label: string; value: string; description: string }[]>([]);

const selectedOption = ref<string | null>(null);

const radioLabels = ref<HTMLElement[]>([]); // Ref to hold radio labels

type SelectedEffortSummary = {
  duration: string;
  distance: string;
  speed: string;
  gradient: string;
  elevation: string;
};

const selectedEffort = computed<ActivityEffort | null>(() => {
  if (!activity.value?.activityEfforts || !selectedOption.value) {
    return null;
  }

  return activity.value.activityEfforts.find(
    (effort) => effort.id === selectedOption.value
  ) ?? null;
});

const selectedEffortSummary = computed<SelectedEffortSummary | null>(() => {
  const effort = selectedEffort.value;
  if (!effort) {
    return null;
  }

  const distanceInKm = effort.distance > 0 ? effort.distance / 1000 : 0;
  const speed = effort.seconds > 0 ? effort.distance / effort.seconds : 0;
  const gradient = effort.distance > 0 ? (effort.deltaAltitude / effort.distance) * 100 : 0;
  const elevationPrefix = effort.deltaAltitude >= 0 ? "D+" : "D-";

  return {
    duration: formatTime(effort.seconds),
    distance: `${distanceInKm.toFixed(2)} km`,
    speed: formatSpeedWithUnit(speed, activity.value?.type ?? "Ride"),
    gradient: `Gradient ${gradient.toFixed(1)}%`,
    elevation: `${elevationPrefix} ${Math.abs(Math.round(effort.deltaAltitude))} m`,
  };
});

const stravaActivityUrl = computed(() => `https://www.strava.com/activities/${activity.value?.id ?? activityId ?? ""}`);
const activityTypeLabel = computed(() => activity.value?.type ?? "Activity");

const activityDateLabel = computed(() => {
  const rawDate = activity.value?.startDateLocal ?? activity.value?.startDate;
  if (!rawDate) {
    return "Date unavailable";
  }

  const parsedDate = new Date(rawDate);
  if (Number.isNaN(parsedDate.getTime())) {
    return rawDate.substring(0, 16);
  }

  return parsedDate.toLocaleString(undefined, {
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
});

const effortCountLabel = computed(() => {
  const effortCount = activity.value?.activityEfforts?.length ?? 0;
  return `${effortCount} effort${effortCount > 1 ? "s" : ""}`;
});

type DetailKpi = {
  label: string;
  value: string;
  hint?: string;
};

const kpis = computed<DetailKpi[]>(() => {
  const currentActivity = activity.value;
  if (!currentActivity) {
    return [];
  }

  const baseKpis: DetailKpi[] = [
    {
      label: "Distance",
      value: `${(currentActivity.distance / 1000).toFixed(1)} km`,
    },
    {
      label: "Elapsed time",
      value: formatTime(currentActivity.elapsedTime),
    },
    {
      label: "Moving time",
      value: formatTime(currentActivity.movingTime),
    },
    {
      label: "Elevation gain",
      value: `${currentActivity.totalElevationGain.toFixed(0)} m`,
    },
    {
      label: "Descent",
      value: `${currentActivity.totalDescent.toFixed(0)} m`,
    },
    {
      label: "Average speed",
      value: formatSpeedWithUnit(currentActivity.averageSpeed, currentActivity.type),
    },
  ];

  if (currentActivity.averageHeartrate > 0) {
    baseKpis.push({
      label: "Avg HR",
      value: `${Math.round(currentActivity.averageHeartrate)} bpm`,
    });
  }

  if (currentActivity.averageWatts > 0) {
    baseKpis.push({
      label: "Avg power",
      value: `${Math.round(currentActivity.averageWatts)} W`,
      hint: currentActivity.deviceWatts ? "Power meter" : "Estimated",
    });
  }

  return baseKpis;
});

type HighlightItem = {
  title: string;
  value: string;
  subtitle: string;
};

const highlights = computed<HighlightItem[]>(() => {
  const efforts = (activity.value?.activityEfforts ?? []).filter(
    (effort) => effort.seconds > 0 && effort.distance > 0
  );
  if (!efforts.length) {
    return [];
  }

  const result: HighlightItem[] = [];

  const fastestEffort = [...efforts].sort((left, right) => left.seconds - right.seconds)[0];
  result.push({
    title: "Fastest effort",
    value: formatTime(fastestEffort.seconds),
    subtitle: `${fastestEffort.label} · ${(fastestEffort.distance / 1000).toFixed(2)} km`,
  });

  const longestEffort = [...efforts].sort((left, right) => right.distance - left.distance)[0];
  result.push({
    title: "Longest effort",
    value: `${(longestEffort.distance / 1000).toFixed(2)} km`,
    subtitle: `${longestEffort.label} · ${formatTime(longestEffort.seconds)}`,
  });

  const steepestAscent = [...efforts]
    .filter((effort) => effort.deltaAltitude > 0 && effort.distance > 0)
    .sort(
      (left, right) =>
        right.deltaAltitude / right.distance - left.deltaAltitude / left.distance
    )[0];

  if (steepestAscent) {
    result.push({
      title: "Steepest ascent",
      value: `${((steepestAscent.deltaAltitude / steepestAscent.distance) * 100).toFixed(1)}%`,
      subtitle: `${steepestAscent.label} · D+ ${Math.round(steepestAscent.deltaAltitude)} m`,
    });
  }

  return result;
});

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
    const apiMessage = await extractApiErrorMessage(response.clone());
    try {
      await ErrorService.catchError(response);
    } catch {
      // The toast has already been emitted by ErrorService.
    }
    throw new Error(apiMessage);
  }
  const detailed = (await response.json()) as DetailedActivity;
  activity.value = detailed;
  loadError.value = null;
  loadWarning.value = getDetailedActivityWarning(detailed);
}

async function extractApiErrorMessage(response: Response): Promise<string> {
  const cacheOnly404Message =
    "This activity is not available in local cache. In cache-only mode, detailed activities must already exist in cache.";

  try {
    const payload = (await response.json()) as {
      message?: string;
      description?: string;
    };

    const description = payload.description?.trim() ?? "";
    const message = payload.message?.trim() ?? "";

    if (response.status === 404) {
      if (description.length > 0 && !description.toLowerCase().startsWith("illegal argument")) {
        return description;
      }
      return cacheOnly404Message;
    }

    if (description.length > 0) {
      return description;
    }
    if (message.length > 0) {
      return message;
    }
  } catch {
    // Ignore JSON parsing errors and fallback to status text.
  }

  if (response.status === 404) {
    return cacheOnly404Message;
  }
  return response.statusText || "Unable to load this activity.";
}

function getDetailedActivityWarning(detailed: DetailedActivity): string | null {
  const hasDistanceStream =
    Array.isArray(detailed.stream?.distance) && detailed.stream.distance.length > 0;

  if (hasDistanceStream) {
    return null;
  }

  return "Detailed streams are missing in local cache for this activity. If you are running in cache-only mode, reconnect to Strava and refresh cache.";
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
        basePolyline.value = L.polyline(filteredLatlngs, {
          color: "#ef5a2a",
          weight: 4,
          opacity: 0.85,
        }).addTo(map.value);
      }

      const bounds = L.latLngBounds(filteredLatlngs);
      if (bounds.isValid()) {
        map.value.fitBounds(bounds);
      }
    }
  }
};

const updateBasePolylineStyle = (isSelectionActive: boolean) => {
  if (!basePolyline.value) {
    return;
  }

  basePolyline.value.setStyle({
    color: "#ef5a2a",
    weight: 4,
    opacity: isSelectionActive ? 0.28 : 0.85,
  });
};

const clearSelectedChartOverlay = () => {
  if (
    chartOptions.series &&
    chartOptions.series.length > 2
  ) {
    (chartOptions.series[2] as SeriesAreaOptions).data = [];
  }

  if (chartInstance) {
    chartInstance.update({
      series: chartOptions.series,
    });
  }
};

const clearSelectedEffort = () => {
  selectedOption.value = null;

  if (selectedPolyline.value) {
    selectedPolyline.value.remove();
    selectedPolyline.value = null;
  }

  updateBasePolylineStyle(false);
  clearSelectedChartOverlay();

  if (map.value && basePolyline.value) {
    const bounds = basePolyline.value.getBounds();
    if (bounds.isValid()) {
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
    enabled: false,
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
  selectedOption.value = key;

  // 1 - Get the selected effort
  const effort: ActivityEffort | undefined = activity.value?.activityEfforts.find(
    (effort) => effort.id === key
  );
  if (!effort) {
    console.error(`No effort found for value: ${key}`);
    return;
  }

  // 2 - Get the stream data for the selected effort
  const stream = activity.value?.stream;
  if (!stream) {
    console.error(`No stream data found for effort: ${effort}`);
    return;
  }
  const startIndex = effort.idxStart;
  const endIndex = effort.idxEnd;
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
        selectedPolyline.value = L.polyline(latlngs, {
          color: "#2a5bd7",
          weight: 5,
          opacity: 0.95,
        }).addTo(map.value);
      }

      updateBasePolylineStyle(true);

      const bounds = L.latLngBounds(latlngs);
      if (bounds.isValid()) {
        map.value.fitBounds(bounds);
      }
    } else if (selectedPolyline.value) {
      selectedPolyline.value.remove();
      selectedPolyline.value = null;
      updateBasePolylineStyle(false);
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
          color: "#2a5bd7",
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
    loadWarning.value = null;
    loadError.value = error instanceof Error && error.message
      ? error.message
      : "Unable to load this activity.";
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
.detail-alert {
  margin: 0;
}

.detail-alert--warning {
  margin-bottom: 0;
}

.detail-view {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 12px;
}

.detail-hero,
.detail-card,
.detail-kpi-card,
.detail-highlight-card {
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  animation: detailFadeInUp 0.36s ease both;
}

.detail-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
  padding: 14px;
}

.detail-hero__kicker {
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 700;
  margin: 0;
}

.detail-hero__title {
  margin: 2px 0 8px;
  font-size: clamp(1.1rem, 1.2vw + 0.9rem, 1.6rem);
  line-height: 1.25;
}

.detail-hero__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.detail-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border: 1px solid var(--ms-border);
  border-radius: 999px;
  font-size: 0.78rem;
  color: var(--ms-text-muted);
  background: #fafbfe;
}

.detail-hero__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.detail-btn-strava {
  background: var(--ms-primary);
  border-color: var(--ms-primary);
}

.detail-kpi-grid,
.detail-highlights {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(155px, 1fr));
  gap: 10px;
}

.detail-kpi-card,
.detail-highlight-card {
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.detail-kpi-card:hover,
.detail-highlight-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(20, 30, 60, 0.1);
}

.detail-kpi-card__label,
.detail-highlight-card__label {
  font-size: 0.74rem;
  text-transform: uppercase;
  letter-spacing: 0.07em;
  color: var(--ms-text-muted);
  font-weight: 700;
}

.detail-kpi-card__value,
.detail-highlight-card__value {
  font-size: 1.08rem;
  color: var(--ms-text);
}

.detail-kpi-card__hint,
.detail-highlight-card__hint {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.detail-map-layout {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr);
  gap: 10px;
}

.detail-card {
  padding: 10px;
}

.detail-card--map {
  min-width: 0;
}

.detail-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.detail-card__header h2 {
  margin: 0;
  font-size: 0.98rem;
  font-weight: 800;
}

.detail-card__header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.detail-map {
  width: 100%;
  height: 420px;
  border: 1px solid var(--ms-border);
  border-radius: 10px;
}

.detail-card--efforts {
  min-width: 0;
}

.selected-effort-panel {
  border: 1px solid #e0e7ff;
  border-radius: 10px;
  background: #f8faff;
  padding: 8px 10px;
  margin-bottom: 10px;
  animation: detailFadeInUp 0.24s ease both;
}

.selected-effort-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.selected-effort-panel__title {
  margin: 0 0 8px;
  font-weight: 700;
  line-height: 1.3;
}

.selected-effort-panel__metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.selected-effort-panel__metrics span {
  border: 1px solid #d9e2ff;
  border-radius: 999px;
  background: #ffffff;
  color: #334155;
  padding: 3px 8px;
  font-size: 0.78rem;
  font-weight: 600;
}

.radio-scroll-container {
  max-height: 420px;
  overflow-y: auto;
  padding-right: 4px;
}

.effort-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 6px;
  border-bottom: 1px solid #f2f3f7;
  border-radius: 8px;
  transition: background-color 0.16s ease, border-color 0.16s ease;
}

.effort-option--active {
  background: #eff5ff;
  border: 1px solid #cfdcff;
}

.effort-option:last-child {
  border-bottom: none;
}

.radio-input {
  margin: 0;
}

.radio-label {
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  color: var(--ms-text);
  font-size: 0.9rem;
}

.radio-label--active {
  color: #1d4ed8;
}

.detail-chip--active {
  color: #1d4ed8;
  border-color: #cfdcff;
  background: #eff5ff;
  font-weight: 700;
}

.detail-card--chart {
  padding-bottom: 12px;
}

.detail-card__header--chart {
  margin-bottom: 10px;
}

.detail-chart {
  width: 100%;
  height: 420px;
}

.switch-container {
  display: flex;
  justify-content: flex-end;
}

.tooltip-inner ul,
.tooltip-inner li {
  text-align: left;
}

.back-arrow {
  position: fixed;
  top: 80px;
  left: 24px;
  width: 42px;
  height: 42px;
  border-radius: 999px;
  background: var(--ms-surface-strong);
  border: 1px solid var(--ms-border);
  box-shadow: var(--ms-shadow-soft);
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--ms-text);
  z-index: 15;
  transition: transform 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.back-arrow:hover {
  transform: translateX(-2px);
  color: var(--ms-primary);
  border-color: #ffd3c1;
}

@keyframes detailFadeInUp {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .detail-hero,
  .detail-card,
  .detail-kpi-card,
  .detail-highlight-card,
  .selected-effort-panel {
    animation: none;
  }

  .detail-kpi-card,
  .detail-highlight-card,
  .effort-option {
    transition: none;
  }
}

@media (max-width: 1200px) {
  .back-arrow {
    position: static;
    margin-bottom: 6px;
  }

  .detail-hero {
    margin-top: 0;
  }
}

@media (max-width: 980px) {
  .detail-map-layout {
    grid-template-columns: 1fr;
  }

  .radio-scroll-container {
    max-height: 240px;
  }

  .detail-map,
  .detail-chart {
    height: 340px;
  }
}

@media (max-width: 680px) {
  .detail-hero {
    flex-direction: column;
  }

  .detail-hero__actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>

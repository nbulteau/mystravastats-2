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
            <span class="detail-chip detail-chip--active">{{ activityVersionLabel }}</span>
            <span class="detail-chip">{{ effortCountLabel }}</span>
          </div>
        </div>
        <div class="detail-hero__actions">
          <div class="detail-version-toggle" aria-label="Activity data version">
            <button
              type="button"
              :class="['btn btn-sm', activityVersion === 'corrected' ? 'btn-primary' : 'btn-outline-secondary']"
              :disabled="!canSelectCorrectedVersion"
              :title="canSelectCorrectedVersion ? 'Show corrected data' : 'No correction applied'"
              @click="switchActivityVersion('corrected')"
            >
              Corrected
            </button>
            <button
              type="button"
              :class="['btn btn-sm', activityVersion === 'raw' ? 'btn-primary' : 'btn-outline-secondary']"
              @click="switchActivityVersion('raw')"
            >
              Raw
            </button>
          </div>
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
        v-if="activity"
        class="detail-insight-grid"
      >
        <article class="detail-card detail-panel">
          <header class="detail-panel__header">
            <h2>Activity Summary</h2>
          </header>
          <dl class="detail-metric-list">
            <div
              v-for="row in summaryRows"
              :key="row.label"
              class="detail-metric-row"
            >
              <dt>{{ row.label }}</dt>
              <dd>
                <strong>{{ row.value }}</strong>
                <small v-if="row.hint">{{ row.hint }}</small>
              </dd>
            </div>
          </dl>
        </article>

        <article class="detail-card detail-panel">
          <header class="detail-panel__header">
            <h2>Data & Source</h2>
          </header>
          <dl class="detail-metric-list">
            <div
              v-for="row in dataSourceRows"
              :key="row.label"
              class="detail-metric-row"
              :class="row.tone ? `detail-metric-row--${row.tone}` : undefined"
            >
              <dt>{{ row.label }}</dt>
              <dd>
                <strong>{{ row.value }}</strong>
                <small v-if="row.hint">{{ row.hint }}</small>
              </dd>
            </div>
          </dl>
          <div
            v-if="versionDifferenceRows.length > 0"
            class="detail-version-diff"
          >
            <h3>Differences vs {{ comparisonVersionLabel }}</h3>
            <ul>
              <li
                v-for="row in versionDifferenceRows"
                :key="row.label"
              >
                <span>{{ row.label }}</span>
                <strong>{{ row.value }}</strong>
              </li>
            </ul>
          </div>
        </article>

        <article class="detail-card detail-panel">
          <header class="detail-panel__header">
            <h2>Power</h2>
          </header>
          <template v-if="powerRows.length > 0 || bestPowerRows.length > 0">
            <dl class="detail-metric-list">
              <div
                v-for="row in powerRows"
                :key="row.label"
                class="detail-metric-row"
              >
                <dt>{{ row.label }}</dt>
                <dd>
                  <strong>{{ row.value }}</strong>
                  <small v-if="row.hint">{{ row.hint }}</small>
                </dd>
              </div>
            </dl>
            <div
              v-if="bestPowerRows.length > 0"
              class="detail-best-grid"
            >
              <div
                v-for="row in bestPowerRows"
                :key="row.label"
                class="detail-best-grid__item"
              >
                <span>{{ row.label }}</span>
                <strong>{{ row.value }}</strong>
              </div>
            </div>
          </template>
          <p
            v-else
            class="detail-empty-state"
          >
            No usable power data.
          </p>
        </article>

        <article class="detail-card detail-panel">
          <header class="detail-panel__header">
            <h2>Heart Rate</h2>
          </header>
          <dl
            v-if="heartRateRows.length > 0"
            class="detail-metric-list"
          >
            <div
              v-for="row in heartRateRows"
              :key="row.label"
              class="detail-metric-row"
            >
              <dt>{{ row.label }}</dt>
              <dd>
                <strong>{{ row.value }}</strong>
                <small v-if="row.hint">{{ row.hint }}</small>
              </dd>
            </div>
          </dl>
          <div
            v-if="activityHeartRateZones"
            class="detail-zone-bars"
          >
            <div
              v-for="zone in activityHeartRateZones.zones"
              :key="zone.zone"
              class="detail-zone-bar"
            >
              <span>{{ zone.zone }}</span>
              <div>
                <i :style="{ width: `${zone.percentage}%` }" />
              </div>
              <strong>{{ zone.percentage.toFixed(0) }}%</strong>
            </div>
          </div>
          <p
            v-if="heartRateRows.length === 0 && !activityHeartRateZones"
            class="detail-empty-state"
          >
            No heart rate data available.
          </p>
        </article>
      </section>

      <section
        v-if="activityComparison"
        class="detail-card detail-comparison"
      >
        <header class="detail-card__header">
          <div>
            <h2>Similar Effort</h2>
            <p class="detail-comparison__subtitle">{{ comparisonScopeLabel }}</p>
          </div>
          <div class="detail-card__header-actions">
            <span
              class="detail-comparison__status"
              :class="comparisonStatusClass"
            >
              {{ activityComparisonDisplayLabel }}
            </span>
            <button
              type="button"
              class="btn btn-sm btn-outline-secondary detail-collapse-toggle"
              :aria-expanded="similarEffortExpanded"
              aria-controls="similar-effort-panel"
              @click="similarEffortExpanded = !similarEffortExpanded"
            >
              <i
                :class="similarEffortExpanded ? 'fa-solid fa-chevron-up' : 'fa-solid fa-chevron-down'"
                aria-hidden="true"
              />
              {{ similarEffortExpanded ? "Hide" : "Show" }}
            </button>
          </div>
        </header>

        <div
          v-if="similarEffortExpanded"
          id="similar-effort-panel"
          class="detail-comparison__collapsible"
        >
          <div
            v-if="activityComparison.criteria.sampleSize > 0"
            class="detail-comparison__metrics"
          >
            <div
              v-for="row in comparisonMetricRows"
              :key="row.label"
              class="detail-comparison__metric"
            >
              <span class="detail-comparison__metric-label">{{ row.label }}</span>
              <strong>{{ row.current }}</strong>
              <small>
                Ref {{ row.baseline }}
                <span :class="row.deltaClass">{{ row.delta }}</span>
              </small>
            </div>
          </div>

          <div
            v-if="activityComparison.criteria.sampleSize > 0"
            class="detail-comparison__body"
          >
            <div class="detail-comparison__table-wrap">
              <h3>Closest Activities</h3>
              <table class="detail-comparison__table">
                <thead>
                  <tr>
                    <th>Activity</th>
                    <th>Distance</th>
                    <th>D+</th>
                    <th>Speed</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="similar in activityComparison.similarActivities"
                    :key="similar.id"
                  >
                    <td>
                      <RouterLink :to="`/activity/${similar.id}`">
                        {{ similar.name }}
                      </RouterLink>
                      <small>{{ formatComparisonDate(similar.date) }}</small>
                    </td>
                    <td>{{ (similar.distance / 1000).toFixed(1) }} km</td>
                    <td>{{ Math.round(similar.elevationGain) }} m</td>
                    <td>{{ formatSpeedWithUnit(similar.averageSpeed, effectiveActivityType) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div class="detail-comparison__segments">
              <h3>Common Segments</h3>
              <ul v-if="activityComparison.commonSegments.length > 0">
                <li
                  v-for="segment in activityComparison.commonSegments"
                  :key="segment.id"
                >
                  <strong>{{ segment.name }}</strong>
                  <span>{{ segment.matchCount }} match{{ segment.matchCount > 1 ? "es" : "" }}</span>
                </li>
              </ul>
              <p v-else class="detail-comparison__empty">
                No cached common segments.
              </p>
            </div>
          </div>

          <p
            v-else
            class="detail-comparison__empty"
          >
            No similar activity found for this season and sport.
          </p>
        </div>
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
            <h2>Route Map</h2>
          </header>
          <div id="map-container" ref="mapContainerRef" class="detail-map" />
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
              <span v-if="selectedEffortSummary.power">{{ selectedEffortSummary.power }}</span>
            </div>
          </div>
          <div class="detail-effort-tabs" role="tablist" aria-label="Effort source">
            <button
              type="button"
              :class="['detail-effort-tab', { 'detail-effort-tab--active': effortPanelTab === 'computed' }]"
              role="tab"
              :aria-selected="effortPanelTab === 'computed'"
              @click="effortPanelTab = 'computed'"
            >
              Efforts
              <span>{{ computedEffortOptions.length }}</span>
            </button>
            <button
              type="button"
              :class="['detail-effort-tab', { 'detail-effort-tab--active': effortPanelTab === 'strava' }]"
              role="tab"
              :aria-selected="effortPanelTab === 'strava'"
              @click="effortPanelTab = 'strava'"
            >
              Segments Strava
              <span>{{ stravaSegmentOptions.length }}</span>
            </button>
          </div>
          <div
            v-if="effortPanelTab === 'strava'"
            class="detail-effort-filters"
          >
            <label>
              <span>Search</span>
              <input
                v-model="segmentSearch"
                type="search"
                placeholder="Segment name"
              >
            </label>
            <label>
              <span>Filter</span>
              <select v-model="segmentFilter">
                <option value="all">All</option>
                <option value="pr">PR only</option>
                <option value="starred">Starred</option>
                <option value="climbs">Climbs</option>
                <option value="descents">Descents</option>
              </select>
            </label>
            <label>
              <span>Sort</span>
              <select v-model="segmentSort">
                <option value="default">Strava order</option>
                <option value="name">Name</option>
                <option value="duration">Duration</option>
                <option value="power">Power</option>
                <option value="grade">Grade</option>
              </select>
            </label>
          </div>
          <div id="radio-container" class="radio-scroll-container">
            <form v-if="visibleEffortOptions.length > 0">
              <div
                v-for="option in visibleEffortOptions"
                :key="option.id"
                class="effort-option"
                :class="{ 'effort-option--active': selectedOption === option.id }"
              >
                <input
                  :id="option.id"
                  v-model="selectedOption"
                  type="radio"
                  :value="option.id"
                  class="radio-input"
                  @click="handleRouteEffortClick(option.id)"
                >
                <label
                  ref="radioLabels"
                  :for="option.id"
                  class="radio-label"
                  :class="{ 'radio-label--active': selectedOption === option.id }"
                  :title="option.description"
                >
                  <span>{{ option.label }}</span>
                  <small>{{ option.description }}</small>
                  <span
                    v-if="option.badges.length"
                    class="effort-option__badges"
                  >
                    <span
                      v-for="badge in option.badges"
                      :key="badge"
                    >{{ badge }}</span>
                  </span>
                </label>
              </div>
            </form>
            <p v-else class="detail-empty-state">
              No {{ effortPanelTab === "strava" ? "Strava segment" : "computed effort" }} available.
            </p>
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
            <div class="detail-chart-controls" aria-label="Profile series">
              <label :class="{ 'detail-chart-toggle--disabled': !hasSpeedData }">
                <input
                  v-model="chartSeriesVisibility.speed"
                  type="checkbox"
                  :disabled="!hasSpeedData"
                >
                Speed
              </label>
              <label :class="{ 'detail-chart-toggle--disabled': !hasAltitudeData }">
                <input
                  v-model="chartSeriesVisibility.altitude"
                  type="checkbox"
                  :disabled="!hasAltitudeData"
                >
                Altitude
              </label>
              <label :class="{ 'detail-chart-toggle--disabled': !hasPowerData }">
                <input
                  v-model="chartSeriesVisibility.power"
                  type="checkbox"
                  :disabled="!hasPowerData"
                >
                Power
              </label>
              <label :class="{ 'detail-chart-toggle--disabled': !hasHeartRateData }">
                <input
                  v-model="chartSeriesVisibility.heartrate"
                  type="checkbox"
                  :disabled="!hasHeartRateData"
                >
                Heart rate
              </label>
              <label :class="{ 'detail-chart-toggle--disabled': !hasCadenceData }">
                <input
                  v-model="chartSeriesVisibility.cadence"
                  type="checkbox"
                  :disabled="!hasCadenceData"
                >
                Cadence
              </label>
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
          :weight="athleteStore.athleteWeight || 75"
          :display-in-watts-per-kg="true"
        />
      </section>

    </div>
  </template>
</template>

<script setup lang="ts">
import { Tooltip } from "bootstrap";
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import type { DetailedActivity, StravaSegmentEffort } from "@/models/activity.model";
import { formatActivityTypeLabel, formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import { useContextStore } from "@/stores/context.js";
import { useAthleteStore } from "@/stores/athlete";
import { useStatisticsStore } from "@/stores/statistics";
import {
  computeHeartRateZoneDistribution,
  resolveHeartRateZoneSettings,
} from "@/utils/heart-rate-zones";
import {
  resolveManualFtpForDate,
  type AthletePerformanceSettings,
  type ResolvedManualFtp,
} from "@/models/athlete-performance-settings.model";
import { ErrorService } from "@/services/error.service";
import type { Options, SeriesAreaOptions, SeriesLineOptions } from "highcharts";
import Highcharts from "highcharts";
import { Chart } from "highcharts-vue";
import PowerDistributionChart from "@/components/charts/PowerDistributionChart.vue";
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

const speedCurveColor = "#2563eb";
const altitudeCurveColor = "#cbd5e1";
const powerCurveColor = "#8b1e3f";
const heartRateCurveColor = "#dc2626";
const cadenceCurveColor = "#047857";

const contextStore = useContextStore();
const athleteStore = useAthleteStore();
const statisticsStore = useStatisticsStore();

const route = useRoute();

const activityId = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id;


const activity = ref<DetailedActivity | null>(null);
const comparisonVersionActivity = ref<DetailedActivity | null>(null);
const activityVersion = ref<"corrected" | "raw">("raw");
const loadError = ref<string | null>(null);
const loadWarning = ref<string | null>(null);
const similarEffortExpanded = ref(false);

const map = ref<L.Map>();
const mapContainerRef = ref<HTMLElement | null>(null);
const basePolyline = ref<L.Polyline | null>(null);
const selectedPolyline = ref<L.Polyline | null>(null);
const hoverMarker = ref<L.Marker | null>(null);
const lastHoveredPointIndex = ref<number | null>(null);

const chartSeriesVisibility = reactive({
  speed: true,
  altitude: true,
  power: false,
  heartrate: false,
  cadence: false,
});

const hasSpeedData = computed(() => {
  const stream = activity.value?.stream;
  return Boolean(stream?.velocitySmooth?.length && stream.distance?.length);
});

const hasAltitudeData = computed(() => {
  const stream = activity.value?.stream;
  return Boolean(stream?.altitude?.length && stream.distance?.length);
});

const hasPowerData = computed(() => {
  const stream = activity.value?.stream;
  return Boolean(stream?.watts?.length && stream.distance?.length);
});

const hasHeartRateData = computed(() => {
  const stream = activity.value?.stream;
  return Boolean(stream?.heartrate?.length && stream.distance?.length);
});

const hasCadenceData = computed(() => {
  const stream = activity.value?.stream;
  return Boolean(stream?.cadence?.length && stream.distance?.length);
});

type EffortPanelTab = "computed" | "strava";
type SegmentFilter = "all" | "pr" | "starred" | "climbs" | "descents";
type SegmentSort = "default" | "name" | "duration" | "power" | "grade";

type RouteEffortOption = {
  id: string;
  label: string;
  description: string;
  distance: number;
  seconds: number;
  idxStart: number;
  idxEnd: number;
  deltaAltitude?: number | null;
  elevationGain?: number | null;
  elevationLoss?: number | null;
  averagePower?: number | null;
  averageHeartrate?: number | null;
  averageCadence?: number | null;
  grade?: number | null;
  badges: string[];
  source: EffortPanelTab;
};

type RouteEffortDescriptionInput = {
  distance: number;
  seconds: number;
  deltaAltitude?: number | null;
  elevationGain?: number | null;
  elevationLoss?: number | null;
  averagePower?: number | null;
  grade?: number | null;
};

const effortPanelTab = ref<EffortPanelTab>("computed");
const selectedOption = ref<string | null>(null);
const segmentSearch = ref("");
const segmentFilter = ref<SegmentFilter>("all");
const segmentSort = ref<SegmentSort>("default");

const radioLabels = ref<HTMLElement[]>([]); // Ref to hold radio labels

type SelectedEffortSummary = {
  duration: string;
  distance: string;
  speed: string;
  gradient: string;
  elevation: string;
  power?: string;
};

const computedEffortOptions = computed<RouteEffortOption[]>(() => {
  return (activity.value?.activityEfforts ?? []).map((effort) => ({
    id: effort.id,
    label: effort.label,
    description: formatRouteEffortDescription(effort),
    distance: effort.distance,
    seconds: effort.seconds,
    idxStart: effort.idxStart,
    idxEnd: effort.idxEnd,
    deltaAltitude: effort.deltaAltitude,
    elevationGain: effort.elevationGain,
    elevationLoss: effort.elevationLoss,
    averagePower: effort.averagePower,
    badges: [],
    source: "computed",
  }));
});

const stravaSegmentOptions = computed<RouteEffortOption[]>(() => {
  return (activity.value?.stravaSegmentEfforts ?? []).map((effort) => {
    const badges = [
      effort.segment.starred ? "Starred" : null,
      effort.prRank ? `PR #${effort.prRank}` : null,
      effort.komRank ? `KOM #${effort.komRank}` : null,
      effort.hidden ? "Hidden" : null,
    ].filter((badge): badge is string => Boolean(badge));

    return {
      id: `strava-${effort.id}`,
      label: effort.segment.name || effort.name,
      description: formatStravaSegmentDescription(effort),
      distance: effort.distance,
      seconds: effort.elapsedTime,
      idxStart: effort.startIndex,
      idxEnd: effort.endIndex,
      deltaAltitude: effort.segment.elevationHigh - effort.segment.elevationLow,
      averagePower: effort.averageWatts > 0 ? effort.averageWatts : null,
      averageHeartrate: effort.averageHeartRate > 0 ? effort.averageHeartRate : null,
      averageCadence: effort.averageCadence > 0 ? effort.averageCadence : null,
      grade: Number.isFinite(effort.segment.averageGrade) ? effort.segment.averageGrade : null,
      badges,
      source: "strava",
    };
  });
});

const filteredStravaSegmentOptions = computed<RouteEffortOption[]>(() => {
  const search = segmentSearch.value.trim().toLowerCase();
  let options = [...stravaSegmentOptions.value];

  if (search) {
    options = options.filter((option) =>
      option.label.toLowerCase().includes(search) ||
      option.description.toLowerCase().includes(search)
    );
  }

  options = options.filter((option) => {
    switch (segmentFilter.value) {
      case "pr":
        return option.badges.some((badge) => badge.startsWith("PR #"));
      case "starred":
        return option.badges.includes("Starred");
      case "climbs":
        return (option.grade ?? 0) > 0.5;
      case "descents":
        return (option.grade ?? 0) < -0.5;
      default:
        return true;
    }
  });

  return options.sort((left, right) => {
    switch (segmentSort.value) {
      case "name":
        return left.label.localeCompare(right.label);
      case "duration":
        return right.seconds - left.seconds;
      case "power":
        return (right.averagePower ?? 0) - (left.averagePower ?? 0);
      case "grade":
        return Math.abs(right.grade ?? 0) - Math.abs(left.grade ?? 0);
      default:
        return 0;
    }
  });
});

const visibleEffortOptions = computed<RouteEffortOption[]>(() => {
  return effortPanelTab.value === "strava"
    ? filteredStravaSegmentOptions.value
    : computedEffortOptions.value;
});

const selectedEffort = computed<RouteEffortOption | null>(() => {
  if (!selectedOption.value) {
    return null;
  }

  return [...computedEffortOptions.value, ...stravaSegmentOptions.value].find(
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
  const gradient = resolveEffortGradient(effort) ?? 0;
  return {
    duration: formatTime(effort.seconds),
    distance: `${distanceInKm.toFixed(2)} km`,
    speed: formatSpeedWithUnit(speed, effectiveActivityType.value),
    gradient: `Grade ${gradient.toFixed(1)}%`,
    elevation: resolveEffortElevationLabel(effort) ?? "D+ 0 m",
    power: effort.averagePower && effort.averagePower > 0
      ? `Power ${Math.round(effort.averagePower)} W`
      : undefined,
  };
});

const stravaActivityUrl = computed(() => `https://www.strava.com/activities/${activity.value?.id ?? activityId ?? ""}`);
const effectiveActivityType = computed(() => resolveEffectiveActivityType(activity.value));
const activityTypeLabel = computed(() => formatActivityTypeLabel(effectiveActivityType.value));
const activityVersionLabel = computed(() => {
  if (activityVersion.value === "raw") {
    return "Raw";
  }

  return "Corrected";
});
const comparisonVersionLabel = computed(() => activityVersion.value === "corrected" ? "Raw" : "Corrected");

const activityDateLabel = computed(() => {
  const rawDate = activity.value?.startDateLocal ?? activity.value?.startDate;
  if (!rawDate) {
    return "Date unavailable";
  }

  const parsedDate = new Date(rawDate);
  if (Number.isNaN(parsedDate.getTime())) {
    return rawDate.substring(0, 16);
  }

  return parsedDate.toLocaleString("en-US", {
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
      label: "D+",
      value: `${currentActivity.totalElevationGain.toFixed(0)} m`,
    },
    {
      label: "D-",
      value: `${currentActivity.totalDescent.toFixed(0)} m`,
    },
    {
      label: "Average speed",
      value: formatSpeedWithUnit(currentActivity.averageSpeed, currentActivity.sportType || currentActivity.type),
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

type DetailMetricRow = {
  label: string;
  value: string;
  hint?: string;
  tone?: "muted" | "good" | "warn";
};

const cadenceUnit = computed(() => effectiveActivityType.value.endsWith("Run") ? "spm" : "rpm");

type PowerAnalysis = {
  averagePower: number | null;
  maxPower: number | null;
  best20MinutePower: number | null;
  best60MinutePower: number | null;
  normalizedPower: number | null;
  ftp: number | null;
  ftpSource: string | null;
  ftpSourceKind: "manual" | "strava" | "estimated" | null;
  weightKg: number | null;
  weightSource: string | null;
  intensityFactor: number | null;
  trainingStressScore: number | null;
  workKilojoules: number | null;
};

const summaryRows = computed<DetailMetricRow[]>(() => {
  const currentActivity = activity.value;
  if (!currentActivity) {
    return [];
  }

  const rows: DetailMetricRow[] = [
    { label: "Sport", value: activityTypeLabel.value },
    { label: "Strava base type", value: currentActivity.type || "N/A" },
    { label: "Date", value: activityDateLabel.value },
    { label: "Distance", value: `${(currentActivity.distance / 1000).toFixed(1)} km` },
    { label: "D+ / D-", value: `${Math.round(currentActivity.totalElevationGain)} m / ${Math.round(currentActivity.totalDescent)} m` },
    { label: "Moving time", value: formatTime(currentActivity.movingTime), hint: `Elapsed ${formatTime(currentActivity.elapsedTime)}` },
    { label: "Average speed", value: formatSpeedWithUnit(currentActivity.averageSpeed, effectiveActivityType.value) },
  ];

  if (currentActivity.maxSpeed > 0) {
    rows.push({
      label: "Max speed",
      value: formatSpeedWithUnit(currentActivity.maxSpeed, effectiveActivityType.value),
    });
  }

  if (currentActivity.averageCadence > 0) {
    rows.push({
      label: "Average cadence",
      value: formatCadenceValue(currentActivity.averageCadence),
    });
  }

  return rows;
});

const streamAvailabilityRows = computed<DetailMetricRow[]>(() => {
  const stream = activity.value?.stream;
  const rows: DetailMetricRow[] = [
    {
      label: "GPS",
      value: stream?.latlng?.length ? `${stream.latlng.length} points` : "Missing",
      tone: stream?.latlng?.length ? "good" : "warn",
    },
    {
      label: "Altitude",
      value: stream?.altitude?.length ? `${stream.altitude.length} points` : "Missing",
      tone: stream?.altitude?.length ? "good" : "muted",
    },
    {
      label: "Speed",
      value: stream?.velocitySmooth?.length ? `${stream.velocitySmooth.length} points` : "Missing",
      tone: stream?.velocitySmooth?.length ? "good" : "muted",
    },
    {
      label: "Heart rate",
      value: stream?.heartrate?.length ? `${stream.heartrate.length} points` : "Missing",
      tone: stream?.heartrate?.length ? "good" : "muted",
    },
    {
      label: "Cadence",
      value: stream?.cadence?.length ? `${stream.cadence.length} points` : "Missing",
      tone: stream?.cadence?.length ? "good" : "muted",
    },
    {
      label: "Watts",
      value: stream?.watts?.length ? `${stream.watts.length} points` : "Missing",
      tone: stream?.watts?.length ? "good" : "muted",
    },
  ];

  return rows;
});

const availableStreamSummary = computed(() => {
  const available = streamAvailabilityRows.value
    .filter((row) => row.value !== "Missing")
    .map((row) => row.label);

  return available.length > 0 ? available.join(", ") : "No detailed streams";
});

const powerSourceLabel = computed(() => {
  const currentActivity = activity.value;
  if (!currentActivity || (!hasPowerData.value && currentActivity.averageWatts <= 0)) {
    return "None";
  }

  return currentActivity.deviceWatts ? "Power meter" : "Strava estimate";
});

const correctionSummary = computed(() => {
  if (!comparisonVersionActivity.value) {
    return "Comparison unavailable";
  }

  if (versionDifferenceRows.value.length === 0) {
    return "No correction applied";
  }

  const count = versionDifferenceRows.value.length;
  return activityVersion.value === "corrected"
    ? `${count} corrected field${count > 1 ? "s" : ""}`
    : `${count} corrected field${count > 1 ? "s" : ""} available`;
});

const dataSourceRows = computed<DetailMetricRow[]>(() => {
  const streamCount = streamAvailabilityRows.value.filter((row) => row.value !== "Missing").length;
  return [
    { label: "Displayed view", value: activityVersionLabel.value },
    {
      label: "Correction status",
      value: correctionSummary.value,
      tone: versionDifferenceRows.value.length > 0 ? "good" : "muted",
    },
    {
      label: "Power source",
      value: powerSourceLabel.value,
      hint: hasPowerData.value ? "Power stream available" : undefined,
    },
    {
      label: "Streams",
      value: availableStreamSummary.value,
      hint: `${streamCount}/6 available`,
    },
  ];
});

const versionDifferenceRows = computed<DetailMetricRow[]>(() => {
  const currentActivity = activity.value;
  const comparisonActivity = comparisonVersionActivity.value;
  if (!currentActivity || !comparisonActivity) {
    return [];
  }

  return buildVersionDifferenceRows(currentActivity, comparisonActivity, effectiveActivityType.value);
});

const canSelectCorrectedVersion = computed(() =>
  Boolean(comparisonVersionActivity.value && versionDifferenceRows.value.length > 0)
);

function buildVersionDifferenceRows(
  currentActivity: DetailedActivity,
  comparisonActivity: DetailedActivity,
  activityType: string,
): DetailMetricRow[] {
  const rows: DetailMetricRow[] = [];

  pushVersionDifference(rows, "Distance", currentActivity.distance, comparisonActivity.distance, 1, (value) => `${(value / 1000).toFixed(2)} km`, (delta) => formatSignedDistance(delta));
  pushVersionDifference(rows, "D+", currentActivity.totalElevationGain, comparisonActivity.totalElevationGain, 0.5, (value) => `${Math.round(value)} m`, (delta) => formatSignedMeters(delta));
  pushVersionDifference(rows, "D-", currentActivity.totalDescent, comparisonActivity.totalDescent, 0.5, (value) => `${Math.round(value)} m`, (delta) => formatSignedMeters(delta));
  pushVersionDifference(rows, "Moving time", currentActivity.movingTime, comparisonActivity.movingTime, 1, (value) => formatTime(value), (delta) => formatSignedTime(delta));
  pushVersionDifference(rows, "Average speed", currentActivity.averageSpeed, comparisonActivity.averageSpeed, 0.01, (value) => formatSpeedWithUnit(value, activityType), (delta) => formatSignedSpeed(delta));
  pushVersionDifference(rows, "Avg power", currentActivity.averageWatts, comparisonActivity.averageWatts, 0.5, (value) => `${Math.round(value)} W`, (delta) => formatSignedNumber(delta, " W", 0));
  pushVersionDifference(rows, "Avg HR", currentActivity.averageHeartrate, comparisonActivity.averageHeartrate, 0.5, (value) => `${Math.round(value)} bpm`, (delta) => formatSignedNumber(delta, " bpm", 0));

  return rows;
}

const powerAnalysis = computed<PowerAnalysis>(() =>
  buildPowerAnalysis(
    activity.value,
    athleteStore.athleteFtp,
    athleteStore.athleteWeight,
    athleteStore.performanceSettings,
  )
);

const powerRows = computed<DetailMetricRow[]>(() => {
  const currentActivity = activity.value;
  if (!currentActivity) {
    return [];
  }

  const rows: DetailMetricRow[] = [];
  const analysis = powerAnalysis.value;

  if (analysis.averagePower !== null) {
    rows.push({
      label: "Average power",
      value: `${Math.round(analysis.averagePower)} W`,
      hint: hasPowerData.value ? "Power stream average" : undefined,
    });
  }
  if (analysis.averagePower !== null && analysis.weightKg !== null) {
    rows.push({
      label: "Average W/kg",
      value: `${(analysis.averagePower / analysis.weightKg).toFixed(2)} W/kg`,
      hint: analysis.weightSource ?? undefined,
    });
  }
  if (analysis.maxPower !== null) {
    rows.push({ label: "Max power", value: `${Math.round(analysis.maxPower)} W` });
  }
  if (analysis.best20MinutePower !== null) {
    rows.push({ label: "Max avg power (20 min)", value: `${Math.round(analysis.best20MinutePower)} W` });
  }
  if (analysis.normalizedPower !== null) {
    rows.push({
      label: "Normalized Power (NP)",
      value: `${Math.round(analysis.normalizedPower)} W`,
      hint: "30 s rolling average, 4th-power weighted",
    });
  } else if (currentActivity.weightedAverageWatts > 0) {
    rows.push({
      label: "Weighted avg power",
      value: `${Math.round(currentActivity.weightedAverageWatts)} W`,
      hint: "Provided by Strava",
    });
  }
  if (analysis.intensityFactor !== null) {
    rows.push({ label: "Intensity Factor (IF)", value: analysis.intensityFactor.toFixed(3) });
  }
  if (analysis.trainingStressScore !== null) {
    rows.push({ label: "Training Stress Score (TSS)", value: analysis.trainingStressScore.toFixed(1) });
  }
  if (analysis.ftp !== null) {
    rows.push({
      label: analysis.ftpSourceKind === "estimated" ? "Estimated FTP" : "FTP setting",
      value: `${Math.round(analysis.ftp)} W`,
      hint: analysis.ftpSource ?? undefined,
    });
  }
  if (analysis.ftp !== null && analysis.weightKg !== null) {
    rows.push({
      label: "FTP / kg",
      value: `${(analysis.ftp / analysis.weightKg).toFixed(2)} W/kg`,
      hint: analysis.weightSource ?? undefined,
    });
  }
  if (analysis.workKilojoules !== null) {
    rows.push({ label: "Work", value: `${Math.round(analysis.workKilojoules)} kJ` });
  }
  if ((currentActivity.calories ?? 0) > 0) {
    rows.push({ label: "Calories", value: `${Math.round(currentActivity.calories ?? 0)} kcal` });
  }
  if (rows.length > 0 || hasPowerData.value) {
    rows.push({
      label: "Source",
      value: powerSourceLabel.value,
      hint: hasPowerData.value ? "Available in the profile chart" : undefined,
    });
  }

  return rows;
});

const bestPowerRows = computed<DetailMetricRow[]>(() => {
  const watts = activity.value?.stream?.watts ?? [];
  if (!watts.length) {
    return [];
  }

  return [
    { label: "5 s", seconds: 5 },
    { label: "30 s", seconds: 30 },
    { label: "1 min", seconds: 60 },
    { label: "5 min", seconds: 5 * 60 },
    { label: "60 min", seconds: 60 * 60 },
  ]
    .map(({ label, seconds }) => {
      const value = bestAveragePower(watts, seconds);
      return value !== null ? { label, value: `${Math.round(value)} W` } : null;
    })
    .filter((row): row is DetailMetricRow => row !== null);
});

function buildPowerAnalysis(
  currentActivity: DetailedActivity | null,
  athleteFtp: number,
  athleteWeight: number,
  performanceSettings: AthletePerformanceSettings,
): PowerAnalysis {
  if (!currentActivity) {
    return emptyPowerAnalysis();
  }

  const watts = sanitizePowerSamples(currentActivity.stream?.watts ?? []);
  const durationSeconds = resolvePowerDurationSeconds(currentActivity);
  const averagePower = watts.length > 0
    ? watts.reduce((sum, value) => sum + value, 0) / watts.length
    : currentActivity.averageWatts > 0
      ? currentActivity.averageWatts
      : null;
  const maxPower = watts.length > 0
    ? Math.max(...watts)
    : currentActivity.maxWatts > 0
      ? currentActivity.maxWatts
      : null;
  const best20MinutePower = bestAveragePower(watts, 20 * 60);
  const best60MinutePower = bestAveragePower(watts, 60 * 60);
  const normalizedPower = normalizedPowerFromWatts(watts);
  const manualFtp = resolveManualFtpForDate(
    performanceSettings,
    currentActivity.startDateLocal || currentActivity.startDate,
  );
  const ftpDetails = resolveFtpDetails(manualFtp, athleteFtp, best60MinutePower, best20MinutePower);
  const weightDetails = resolveWeightDetails(performanceSettings.weightKg, athleteWeight);
  const intensityFactor =
    normalizedPower !== null && ftpDetails.ftp !== null
      ? normalizedPower / ftpDetails.ftp
      : null;
  const trainingStressScore =
    normalizedPower !== null &&
    intensityFactor !== null &&
    ftpDetails.ftp !== null &&
    durationSeconds > 0
      ? (durationSeconds * normalizedPower * intensityFactor) / (ftpDetails.ftp * 3600) * 100
      : null;
  const workKilojoules = currentActivity.kilojoules > 0
    ? currentActivity.kilojoules
    : averagePower !== null && durationSeconds > 0
      ? (averagePower * durationSeconds) / 1000
      : null;

  return {
    averagePower,
    maxPower,
    best20MinutePower,
    best60MinutePower,
    normalizedPower,
    ftp: ftpDetails.ftp,
    ftpSource: ftpDetails.source,
    ftpSourceKind: ftpDetails.sourceKind,
    weightKg: weightDetails.weightKg,
    weightSource: weightDetails.source,
    intensityFactor,
    trainingStressScore,
    workKilojoules,
  };
}

function emptyPowerAnalysis(): PowerAnalysis {
  return {
    averagePower: null,
    maxPower: null,
    best20MinutePower: null,
    best60MinutePower: null,
    normalizedPower: null,
    ftp: null,
    ftpSource: null,
    ftpSourceKind: null,
    weightKg: null,
    weightSource: null,
    intensityFactor: null,
    trainingStressScore: null,
    workKilojoules: null,
  };
}

function sanitizePowerSamples(watts: number[]): number[] {
  return watts.map((value) => Number.isFinite(value) && value > 0 ? value : 0);
}

function normalizedPowerFromWatts(watts: number[]): number | null {
  if (watts.length < 30) {
    return null;
  }

  const rollingAverages = rollingAverage(watts, 30);
  if (!rollingAverages.length) {
    return null;
  }

  const fourthPowerAverage = rollingAverages.reduce(
    (sum, value) => sum + Math.pow(value, 4),
    0,
  ) / rollingAverages.length;

  return Math.pow(fourthPowerAverage, 0.25);
}

function rollingAverage(values: number[], windowSize: number): number[] {
  if (windowSize <= 0 || values.length < windowSize) {
    return [];
  }

  const result: number[] = [];
  let windowSum = 0;
  for (let index = 0; index < values.length; index += 1) {
    windowSum += values[index] ?? 0;
    if (index >= windowSize) {
      windowSum -= values[index - windowSize] ?? 0;
    }
    if (index >= windowSize - 1) {
      result.push(windowSum / windowSize);
    }
  }
  return result;
}

function resolveFtpDetails(
  manualFtp: ResolvedManualFtp | null,
  athleteFtp: number,
  best60MinutePower: number | null,
  best20MinutePower: number | null,
): { ftp: number | null; source: string | null; sourceKind: "manual" | "strava" | "estimated" | null } {
  if (manualFtp !== null && manualFtp.ftp > 0) {
    return {
      ftp: manualFtp.ftp,
      source: `Manual setting since ${manualFtp.effectiveFrom}`,
      sourceKind: "manual",
    };
  }
  if (Number.isFinite(athleteFtp) && athleteFtp > 0) {
    return { ftp: athleteFtp, source: "Strava athlete profile", sourceKind: "strava" };
  }
  if (best60MinutePower !== null && best60MinutePower > 0) {
    return { ftp: best60MinutePower, source: "Estimated from best 60 min power", sourceKind: "estimated" };
  }
  if (best20MinutePower !== null && best20MinutePower > 0) {
    return { ftp: best20MinutePower * 0.95, source: "Estimated as 95% of best 20 min power", sourceKind: "estimated" };
  }
  return { ftp: null, source: null, sourceKind: null };
}

function resolveWeightDetails(
  manualWeightKg: number | null | undefined,
  athleteWeight: number,
): { weightKg: number | null; source: string | null } {
  if (typeof manualWeightKg === "number" && Number.isFinite(manualWeightKg) && manualWeightKg > 0) {
    return { weightKg: manualWeightKg, source: "Manual weight setting" };
  }
  if (Number.isFinite(athleteWeight) && athleteWeight > 0) {
    return { weightKg: athleteWeight, source: "Strava athlete profile" };
  }
  return { weightKg: null, source: null };
}

function resolvePowerDurationSeconds(currentActivity: DetailedActivity): number {
  const time = currentActivity.stream?.time ?? [];
  const lastTime = time.length > 0 ? time[time.length - 1] : null;
  if (lastTime !== null && Number.isFinite(lastTime) && lastTime > 0) {
    return lastTime;
  }
  return currentActivity.elapsedTime > 0 ? currentActivity.elapsedTime : currentActivity.movingTime;
}

const resolvedHeartRateSettings = computed(() => {
  return (
    statisticsStore.heartRateZoneAnalysis?.resolvedSettings ??
    resolveHeartRateZoneSettings(
      athleteStore.heartRateZoneSettings,
      Math.trunc(activity.value?.maxHeartrate ?? 0) || null,
    )
  );
});

const activityHeartRateZones = computed(() => {
  const stream = activity.value?.stream;
  if (!stream) {
    return null;
  }

  return computeHeartRateZoneDistribution(
    stream.heartrate ?? null,
    stream.time ?? null,
    resolvedHeartRateSettings.value ?? null,
  );
});

const heartRateRows = computed<DetailMetricRow[]>(() => {
  const currentActivity = activity.value;
  if (!currentActivity) {
    return [];
  }

  const rows: DetailMetricRow[] = [];
  if (currentActivity.averageHeartrate > 0) {
    rows.push({ label: "Average HR", value: `${Math.round(currentActivity.averageHeartrate)} bpm` });
  }
  if (currentActivity.maxHeartrate > 0) {
    rows.push({ label: "Max HR", value: `${Math.round(currentActivity.maxHeartrate)} bpm` });
  }
  if (activityHeartRateZones.value) {
    rows.push({
      label: "Tracked HR time",
      value: formatTime(activityHeartRateZones.value.totalTrackedSeconds),
    });
    if (activityHeartRateZones.value.easyHardRatio !== null && activityHeartRateZones.value.easyHardRatio !== undefined) {
      rows.push({
        label: "Easy/hard ratio",
        value: `${activityHeartRateZones.value.easyHardRatio.toFixed(2)} : 1`,
      });
    }
  }
  if ((currentActivity.sufferScore ?? 0) > 0) {
    rows.push({ label: "Suffer score", value: `${Math.round(currentActivity.sufferScore ?? 0)}` });
  }

  return rows;
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

const activityComparison = computed(() => activity.value?.activityComparison ?? null);

const comparisonScopeLabel = computed(() => {
  const comparison = activityComparison.value;
  if (!comparison) {
    return "";
  }
  const sample = comparison.criteria.sampleSize;
  const activityLabel = formatActivityTypeLabel(comparison.criteria.activityType);
  return `${sample} similar ${activityLabel} activit${sample > 1 ? "ies" : "y"} in ${comparison.criteria.year}`;
});

const comparisonStatusClass = computed(() => {
  const status = activityComparison.value?.status ?? "insufficient-data";
  return `detail-comparison__status--${status}`;
});

const activityComparisonDisplayLabel = computed(() => {
  const comparison = activityComparison.value;
  if (!comparison) {
    return "";
  }

  const labels: Record<string, string> = {
    "typical": "In line with similar activities",
    "faster": "Faster than similar activities",
    "slower": "Slower than similar activities",
    "atypical": "Atypical activity",
    "insufficient-data": "Not enough data",
  };

  return labels[comparison.status] ?? comparison.label;
});

type ComparisonMetricRow = {
  label: string;
  current: string;
  baseline: string;
  delta: string;
  deltaClass: string;
};

const comparisonMetricRows = computed<ComparisonMetricRow[]>(() => {
  const currentActivity = activity.value;
  const comparison = activityComparison.value;
  if (!currentActivity || !comparison || comparison.criteria.sampleSize === 0) {
    return [];
  }

  const baseline = comparison.baseline;
  const deltas = comparison.deltas;
  const rows: ComparisonMetricRow[] = [
    {
      label: "Speed",
      current: formatSpeedWithUnit(currentActivity.averageSpeed, currentActivity.sportType || currentActivity.type),
      baseline: formatSpeedWithUnit(baseline.averageSpeed, currentActivity.sportType || currentActivity.type),
      delta: formatSignedSpeed(deltas.averageSpeed),
      deltaClass: comparisonDeltaClass(deltas.averageSpeed, true),
    },
    {
      label: "Moving time",
      current: formatTime(currentActivity.movingTime),
      baseline: formatTime(baseline.movingTime),
      delta: formatSignedTime(deltas.movingTime),
      deltaClass: comparisonDeltaClass(deltas.movingTime, false),
    },
    {
      label: "Distance",
      current: `${(currentActivity.distance / 1000).toFixed(1)} km`,
      baseline: `${(baseline.distance / 1000).toFixed(1)} km`,
      delta: formatSignedDistance(deltas.distance),
      deltaClass: comparisonDeltaClass(-Math.abs(deltas.distance), true),
    },
    {
      label: "D+",
      current: `${Math.round(currentActivity.totalElevationGain)} m`,
      baseline: `${Math.round(baseline.elevationGain)} m`,
      delta: formatSignedMeters(deltas.elevationGain),
      deltaClass: comparisonDeltaClass(-Math.abs(deltas.elevationGain), true),
    },
  ];

  if (currentActivity.averageHeartrate > 0 || baseline.averageHeartrate > 0) {
    rows.push({
      label: "Avg HR",
      current: `${Math.round(currentActivity.averageHeartrate)} bpm`,
      baseline: `${Math.round(baseline.averageHeartrate)} bpm`,
      delta: formatSignedNumber(deltas.averageHeartrate, " bpm", 0),
      deltaClass: comparisonDeltaClass(deltas.averageHeartrate, false),
    });
  }

  if (currentActivity.averageWatts > 0 || baseline.averageWatts > 0) {
    rows.push({
      label: "Power",
      current: `${Math.round(currentActivity.averageWatts)} W`,
      baseline: `${Math.round(baseline.averageWatts)} W`,
      delta: formatSignedNumber(deltas.averageWatts, " W", 0),
      deltaClass: comparisonDeltaClass(deltas.averageWatts, true),
    });
  }

  if (currentActivity.averageCadence > 0 || baseline.averageCadence > 0) {
    rows.push({
      label: "Cadence",
      current: `${Math.round(currentActivity.averageCadence)} rpm`,
      baseline: `${Math.round(baseline.averageCadence)} rpm`,
      delta: formatSignedNumber(deltas.averageCadence, " rpm", 0),
      deltaClass: comparisonDeltaClass(deltas.averageCadence, true),
    });
  }

  return rows;
});

function resolveEffectiveActivityType(currentActivity?: DetailedActivity | null): string {
  return currentActivity?.sportType || currentActivity?.type || "Ride";
}

function formatCadenceValue(cadence: number): string {
  const displayedCadence = effectiveActivityType.value.endsWith("Run")
    ? cadence * 2
    : cadence;

  return `${Math.round(displayedCadence)} ${cadenceUnit.value}`;
}

function pushVersionDifference(
  rows: DetailMetricRow[],
  label: string,
  currentValue: number,
  comparisonValue: number,
  threshold: number,
  formatter: (value: number) => string,
  deltaFormatter: (value: number) => string,
) {
  const delta = currentValue - comparisonValue;
  if (!Number.isFinite(delta) || Math.abs(delta) <= threshold) {
    return;
  }

  rows.push({
    label,
    value: `${formatter(currentValue)} (${deltaFormatter(delta)})`,
  });
}

function bestAveragePower(watts: number[], windowSamples: number): number | null {
  const finiteWatts = watts.map((value) => Number.isFinite(value) ? value : 0);
  if (finiteWatts.length < windowSamples || windowSamples <= 0) {
    return null;
  }

  let windowSum = 0;
  for (let index = 0; index < windowSamples; index += 1) {
    windowSum += finiteWatts[index] ?? 0;
  }

  let best = windowSum / windowSamples;
  for (let index = windowSamples; index < finiteWatts.length; index += 1) {
    windowSum += (finiteWatts[index] ?? 0) - (finiteWatts[index - windowSamples] ?? 0);
    best = Math.max(best, windowSum / windowSamples);
  }

  return best;
}

function comparisonDeltaClass(value: number, positiveIsGood: boolean): string {
  if (Math.abs(value) < 0.0001) {
    return "detail-comparison__delta detail-comparison__delta--flat";
  }
  const isGood = positiveIsGood ? value > 0 : value < 0;
  return `detail-comparison__delta ${isGood ? "detail-comparison__delta--good" : "detail-comparison__delta--warn"}`;
}

function formatSignedNumber(value: number, suffix: string, digits = 1): string {
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(digits)}${suffix}`;
}

function formatSignedDistance(value: number): string {
  return formatSignedNumber(value / 1000, " km", 1);
}

function formatSignedMeters(value: number): string {
  return formatSignedNumber(value, " m", 0);
}

function formatSignedSpeed(value: number): string {
  return formatSignedNumber(value * 3.6, " km/h", 1);
}

function formatSignedTime(value: number): string {
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${formatTime(Math.abs(value))}`;
}

function formatComparisonDate(value: string): string {
  if (!value) {
    return "";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value.substring(0, 10);
  }
  return parsed.toLocaleDateString("en-US", {
    day: "2-digit",
    month: "short",
  });
}

function formatStravaSegmentDescription(effort: StravaSegmentEffort): string {
  const parts = [
    formatRouteEffortDescription({
      distance: effort.distance,
      seconds: effort.elapsedTime,
      averagePower: effort.averageWatts,
      grade: effort.segment.averageGrade,
    }),
  ];

  if (effort.averageHeartRate > 0) {
    parts.push(`${Math.round(effort.averageHeartRate)} bpm`);
  }

  return parts.join(" · ");
}

function formatRouteEffortDescription(effort: RouteEffortDescriptionInput): string {
  const parts = [
    `${(effort.distance / 1000).toFixed(2)} km`,
    formatTime(effort.seconds),
  ];

  if (effort.seconds > 0 && effort.distance > 0) {
    parts.push(formatSpeedWithUnit(effort.distance / effort.seconds, effectiveActivityType.value));
  }

  const gradient = resolveEffortGradient(effort);

  if (gradient !== null && Number.isFinite(gradient)) {
    parts.push(`Grade ${gradient.toFixed(1)}%`);
  }

  const elevationLabel = resolveEffortElevationLabel(effort);
  if (elevationLabel) {
    parts.push(elevationLabel);
  }

  if (effort.averagePower && effort.averagePower > 0) {
    parts.push(`${Math.round(effort.averagePower)} W`);
  }

  return parts.join(" · ");
}

function resolveEffortGradient(effort: RouteEffortDescriptionInput): number | null {
  const explicitGrade = finiteNumberOrNull(effort.grade);
  if (explicitGrade !== null) {
    return explicitGrade;
  }

  if (effort.distance <= 0) {
    return null;
  }

  const deltaAltitude = finiteNumberOrNull(effort.deltaAltitude);
  const netGradient = deltaAltitude !== null ? (deltaAltitude / effort.distance) * 100 : null;
  if (netGradient !== null && Math.abs(netGradient) >= 0.05) {
    return netGradient;
  }

  const elevationGain = finiteNumberOrNull(effort.elevationGain);
  const elevationLoss = finiteNumberOrNull(effort.elevationLoss);
  if (elevationGain !== null || elevationLoss !== null) {
    const gain = elevationGain ?? 0;
    const loss = elevationLoss ?? 0;
    if (gain >= loss && gain >= 0.5) {
      return (gain / effort.distance) * 100;
    }
    if (loss > gain && loss >= 0.5) {
      return -(loss / effort.distance) * 100;
    }
  }

  return netGradient;
}

function resolveEffortElevationLabel(effort: RouteEffortDescriptionInput): string | null {
  const elevationGain = finiteNumberOrNull(effort.elevationGain);
  const elevationLoss = finiteNumberOrNull(effort.elevationLoss);
  const elevationParts: string[] = [];
  if (elevationGain !== null && elevationGain >= 0.5) {
    elevationParts.push(`D+ ${Math.round(elevationGain)} m`);
  }
  if (elevationLoss !== null && elevationLoss >= 0.5) {
    elevationParts.push(`D- ${Math.round(elevationLoss)} m`);
  }
  if (elevationParts.length > 0) {
    return elevationParts.join(" · ");
  }

  const deltaAltitude = finiteNumberOrNull(effort.deltaAltitude);
  if (deltaAltitude === null) {
    return null;
  }
  return `${deltaAltitude >= 0 ? "D+" : "D-"} ${Math.abs(Math.round(deltaAltitude))} m`;
}

function finiteNumberOrNull(value: number | null | undefined): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

async function fetchDetailedActivity(id: string, version: "corrected" | "raw" = activityVersion.value) {
  const detailed = await fetchDetailedActivityPayload(id, version);
  activity.value = detailed;
  activityVersion.value = version;
  similarEffortExpanded.value = false;
  loadError.value = null;
  loadWarning.value = getDetailedActivityWarning(detailed);
  comparisonVersionActivity.value = await fetchComparisonVersionActivity(id, version);
}

async function fetchInitialDetailedActivity(id: string) {
  const rawActivity = await fetchDetailedActivityPayload(id, "raw");
  const correctedActivity = await fetchComparisonVersionActivity(id, "raw");
  const hasCorrection =
    correctedActivity !== null &&
    buildVersionDifferenceRows(
      correctedActivity,
      rawActivity,
      resolveEffectiveActivityType(correctedActivity),
    ).length > 0;

  const selectedActivity = hasCorrection && correctedActivity ? correctedActivity : rawActivity;
  activity.value = selectedActivity;
  activityVersion.value = hasCorrection ? "corrected" : "raw";
  comparisonVersionActivity.value = hasCorrection ? rawActivity : correctedActivity;
  similarEffortExpanded.value = false;
  loadError.value = null;
  loadWarning.value = getDetailedActivityWarning(selectedActivity);
}

async function fetchDetailedActivityPayload(
  id: string,
  version: "corrected" | "raw",
  emitToast = true,
): Promise<DetailedActivity> {
  const url = version === "raw" ? `/api/activities/${id}?version=raw` : `/api/activities/${id}`;
  const response = await fetch(url);
  if (!response.ok) {
    const apiMessage = await extractApiErrorMessage(response.clone());
    if (emitToast) {
      try {
        await ErrorService.catchError(response);
      } catch {
        // The toast has already been emitted by ErrorService.
      }
    }
    throw new Error(apiMessage);
  }
  return (await response.json()) as DetailedActivity;
}

async function fetchComparisonVersionActivity(id: string, currentVersion: "corrected" | "raw"): Promise<DetailedActivity | null> {
  const nextVersion = currentVersion === "corrected" ? "raw" : "corrected";

  try {
    return await fetchDetailedActivityPayload(id, nextVersion, false);
  } catch {
    return null;
  }
}

async function switchActivityVersion(version: "corrected" | "raw") {
  if (version === activityVersion.value || !activityId) {
    return;
  }
  if (version === "corrected" && !canSelectCorrectedVersion.value) {
    return;
  }
  try {
    await fetchDetailedActivity(activityId, version);
    clearSelectedEffort();
    await nextTick();
    updateMap();
    initChart();
  } catch (error) {
    loadError.value = error instanceof Error && error.message
      ? error.message
      : "Unable to load this activity.";
  }
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
  if (mapContainerRef.value) {
    map.value = L.map(mapContainerRef.value);
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
          return formatSpeedWithUnit(this.value, effectiveActivityType.value);
        },
        style: {
          color: speedCurveColor,
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
          color: altitudeCurveColor,
        },
      },
      opposite: true,
    },
    {
      title: {
        text: "Power",
        style: {
          color: powerCurveColor,
        },
      },
      labels: {
        format: "{value} W",
        style: {
          color: powerCurveColor,
        },
      },
    },
    {
      title: {
        text: "Heart rate",
        style: {
          color: heartRateCurveColor,
        },
      },
      labels: {
        format: "{value} bpm",
        style: {
          color: heartRateCurveColor,
        },
      },
      opposite: true,
    },
    {
      title: {
        text: "Cadence",
        style: {
          color: cadenceCurveColor,
        },
      },
      labels: {
        formatter: function (this: any): string {
          return `${this.value} ${cadenceUnit.value}`;
        },
        style: {
          color: cadenceCurveColor,
        },
      },
      opposite: true,
    },
  ],
  tooltip: {
    formatter: function (this: any): string {
      const x = typeof this.x === "number" ? this.x : this.point?.x ?? 0;
      const lines = [`Distance: ${x.toFixed(1)} km`];
      for (const point of this.points ?? []) {
        lines.push(`${point.series.name}: <b>${formatChartPoint(point.series.name, point.y ?? 0)}</b>`);
      }
      return lines.join("<br/>");
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
      color: speedCurveColor,
    },
    {
      name: "Altitude",
      type: "area",
      data: [],
      color: altitudeCurveColor,
      yAxis: 1,
    },
    {
      name: "Selected segment",
      type: "area",
      data: [],
      color: "blue",
      yAxis: 1,
    },
    {
      name: "Power",
      type: "line",
      data: [],
      color: powerCurveColor,
      dashStyle: "ShortDash",
      yAxis: 2,
    },
    {
      name: "Heart rate",
      type: "line",
      data: [],
      color: heartRateCurveColor,
      yAxis: 3,
    },
    {
      name: "Cadence",
      type: "line",
      data: [],
      color: cadenceCurveColor,
      yAxis: 4,
    },
  ],
});

let chartInstance: Highcharts.Chart | null = null;
let chartMouseMoveHandler: ((e: MouseEvent) => void) | null = null;

const initChart = () => {
  destroyChartInstance();
  syncChartSeries(false);
  const chartContainer = document.getElementById("chart-container");
  if (chartContainer) {
    chartInstance = Highcharts.chart(chartContainer, chartOptions);

    chartMouseMoveHandler = (e: MouseEvent) => {
      if (!chartInstance || !map.value) return;

      const event: Highcharts.PointerEventObject = chartInstance.pointer.normalize(e);
      let point: Highcharts.Point | undefined = undefined;
      const hoverSeries = chartInstance.series.find((series) => series.visible && series.points.length > 0);
      point = hoverSeries?.searchPoint(event, true);

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

function destroyChartInstance() {
  const chartContainer = document.getElementById("chart-container");
  if (chartContainer && chartMouseMoveHandler) {
    chartContainer.removeEventListener("mousemove", chartMouseMoveHandler);
  }
  chartMouseMoveHandler = null;

  if (chartInstance) {
    chartInstance.destroy();
    chartInstance = null;
  }
}

function syncChartSeries(redraw = true) {
  if (!chartOptions.series) {
    return;
  }

  const stream = activity.value?.stream;
  const distanceStream = stream?.distance ?? [];

  (chartOptions.series[0] as SeriesLineOptions).data =
    chartSeriesVisibility.speed && stream?.velocitySmooth
      ? buildDistanceSeries(stream.velocitySmooth, distanceStream)
      : [];
  (chartOptions.series[0] as SeriesLineOptions).visible =
    chartSeriesVisibility.speed && hasSpeedData.value;

  (chartOptions.series[1] as SeriesAreaOptions).data =
    chartSeriesVisibility.altitude && stream?.altitude
      ? buildDistanceSeries(stream.altitude, distanceStream)
      : [];
  (chartOptions.series[1] as SeriesAreaOptions).visible =
    chartSeriesVisibility.altitude && hasAltitudeData.value;

  const altitudeStream = stream?.altitude ?? [];
  if (altitudeStream.length > 0 && Array.isArray(chartOptions.yAxis) && chartOptions.yAxis[1]) {
    const minAltitude = Math.min(...altitudeStream);
    chartOptions.yAxis[1].min = minAltitude * 0.95;
  }

  (chartOptions.series[3] as SeriesLineOptions).data =
    chartSeriesVisibility.power && stream?.watts
      ? buildDistanceSeries(stream.watts, distanceStream)
      : [];
  (chartOptions.series[3] as SeriesLineOptions).visible =
    chartSeriesVisibility.power && hasPowerData.value;

  (chartOptions.series[4] as SeriesLineOptions).data =
    chartSeriesVisibility.heartrate && stream?.heartrate
      ? buildDistanceSeries(stream.heartrate, distanceStream)
      : [];
  (chartOptions.series[4] as SeriesLineOptions).visible =
    chartSeriesVisibility.heartrate && hasHeartRateData.value;

  (chartOptions.series[5] as SeriesLineOptions).data =
    chartSeriesVisibility.cadence && stream?.cadence
      ? buildDistanceSeries(stream.cadence, distanceStream)
      : [];
  (chartOptions.series[5] as SeriesLineOptions).visible =
    chartSeriesVisibility.cadence && hasCadenceData.value;

  if (chartInstance) {
    chartInstance.update({
      series: chartOptions.series,
      yAxis: chartOptions.yAxis,
    }, redraw, true);
  }
}

function buildDistanceSeries(values: number[], distances: number[]) {
  const size = Math.min(values.length, distances.length);
  return Array.from({ length: size }, (_, index) => ({
    x: (distances[index] ?? 0) / 1000,
    y: Number.isFinite(values[index]) ? values[index] : 0,
  }));
}

function formatChartPoint(seriesName: string, value: number): string {
  switch (seriesName) {
    case "Speed":
      return formatSpeedWithUnit(value, effectiveActivityType.value);
    case "Altitude":
    case "Selected segment":
      return `${Math.round(value)} m`;
    case "Power":
      return `${Math.round(value)} W`;
    case "Heart rate":
      return `${Math.round(value)} bpm`;
    case "Cadence":
      return formatCadenceValue(value);
    default:
      return value.toFixed(1);
  }
}

const handleRouteEffortClick = (key: string) => {
  selectedOption.value = key;

  const effort = selectedEffort.value;
  if (!effort) {
    console.error(`No effort found for value: ${key}`);
    return;
  }

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

    // Force the chart update after replacing the selected segment overlay.
    if (chartInstance) {
      chartInstance.update({
        series: chartOptions.series,
      });
    }
  }
};

onMounted(async () => {
  contextStore.updateCurrentView("activity");
  initMap();
  try {
    await Promise.allSettled([
      athleteStore.fetchAthlete(),
      athleteStore.fetchPerformanceSettings(),
      athleteStore.fetchHeartRateZoneSettings(),
      statisticsStore.fetchHeartRateZoneAnalysis(),
    ]);
    await fetchInitialDetailedActivity(activityId ?? "");
    updateMap();
    initChart();

    // Ensure DOM is updated before initializing tooltips
    await nextTick();

    // Initialize tooltips for radio labels
    radioLabels.value.forEach((label) => {
      new Tooltip(label, {
        title: label.getAttribute("title") || "",
        html: true,
        customClass: "detailed-activity-tooltip",
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
  destroyChartInstance();
  if (map.value) {
    map.value.remove();
    map.value = undefined;
  }
  basePolyline.value = null;
  selectedPolyline.value = null;
  hoverMarker.value = null;
  lastHoveredPointIndex.value = null;
});

watch(effortPanelTab, () => {
  clearSelectedEffort();
});

watch([segmentSearch, segmentFilter, segmentSort], () => {
  if (selectedEffort.value && !visibleEffortOptions.value.some((option) => option.id === selectedEffort.value?.id)) {
    clearSelectedEffort();
  }
});

watch([
  () => chartSeriesVisibility.speed,
  () => chartSeriesVisibility.altitude,
  () => chartSeriesVisibility.power,
  () => chartSeriesVisibility.heartrate,
  () => chartSeriesVisibility.cadence,
], () => {
  syncChartSeries();
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

.detail-version-toggle {
  display: inline-flex;
  gap: 4px;
}

.detail-version-toggle .btn {
  min-height: 38px;
  font-weight: 800;
}

.detail-btn-strava {
  background: var(--ms-primary);
  border-color: var(--ms-primary);
}

.detail-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.detail-highlights {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 10px;
}

.detail-insight-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  align-items: stretch;
}

.detail-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.detail-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.detail-panel__header h2 {
  margin: 0;
  font-size: 0.96rem;
  font-weight: 900;
}

.detail-metric-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
}

.detail-metric-row {
  display: grid;
  grid-template-columns: minmax(96px, 0.85fr) minmax(0, 1.15fr);
  gap: 8px;
  align-items: baseline;
  min-width: 0;
  border-top: 1px solid #edf0f5;
  padding-top: 7px;
}

.detail-metric-row:first-child {
  border-top: 0;
  padding-top: 0;
}

.detail-metric-row dt {
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.detail-metric-row dd {
  display: grid;
  gap: 2px;
  min-width: 0;
  margin: 0;
  text-align: right;
}

.detail-metric-row dd strong {
  overflow-wrap: anywhere;
  color: var(--ms-text);
  font-size: 0.88rem;
}

.detail-metric-row dd small {
  color: var(--ms-text-muted);
  font-size: 0.72rem;
}

.detail-metric-row--good dd strong {
  color: #166534;
}

.detail-metric-row--warn dd strong {
  color: #b45309;
}

.detail-metric-row--muted dd strong {
  color: #64748b;
}

.detail-version-diff {
  border-top: 1px solid #edf0f5;
  padding-top: 8px;
}

.detail-version-diff h3 {
  margin: 0 0 6px;
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.detail-version-diff ul {
  display: grid;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.detail-version-diff li {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.detail-version-diff strong {
  color: var(--ms-text);
  text-align: right;
}

.detail-best-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(76px, 1fr));
  gap: 6px;
}

.detail-best-grid__item {
  border: 1px solid #edf0f5;
  border-radius: 8px;
  padding: 6px;
  background: #fbfdff;
  display: grid;
  gap: 2px;
}

.detail-best-grid__item span {
  color: var(--ms-text-muted);
  font-size: 0.68rem;
  font-weight: 800;
  text-transform: uppercase;
}

.detail-best-grid__item strong {
  color: var(--ms-text);
  font-size: 0.88rem;
}

.detail-zone-bars {
  display: grid;
  gap: 6px;
}

.detail-zone-bar {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) 38px;
  gap: 7px;
  align-items: center;
  font-size: 0.72rem;
  color: var(--ms-text-muted);
  font-weight: 800;
}

.detail-zone-bar div {
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: #eef2f7;
}

.detail-zone-bar i {
  display: block;
  height: 100%;
  min-width: 2px;
  border-radius: inherit;
  background: #ef4444;
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

.detail-comparison {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-comparison__subtitle {
  margin: 3px 0 0;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.detail-comparison__status {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--ms-border);
  border-radius: 999px;
  padding: 4px 10px;
  background: #f8fafc;
  color: #475569;
  font-size: 0.78rem;
  font-weight: 800;
}

.detail-comparison__status--faster,
.detail-comparison__status--typical {
  color: #166534;
  border-color: #bbf7d0;
  background: #f0fdf4;
}

.detail-comparison__status--slower,
.detail-comparison__status--atypical {
  color: #92400e;
  border-color: #fde68a;
  background: #fffbeb;
}

.detail-collapse-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.detail-comparison__collapsible {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-comparison__metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 8px;
}

.detail-comparison__metric {
  border-top: 1px solid var(--ms-border);
  padding-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.detail-comparison__metric-label {
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.detail-comparison__metric strong {
  color: var(--ms-text);
}

.detail-comparison__metric small {
  color: var(--ms-text-muted);
}

.detail-comparison__delta {
  margin-left: 4px;
  font-weight: 800;
}

.detail-comparison__delta--good {
  color: #15803d;
}

.detail-comparison__delta--warn {
  color: #b45309;
}

.detail-comparison__delta--flat {
  color: #64748b;
}

.detail-comparison__body {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(220px, 1fr);
  gap: 12px;
  align-items: start;
}

.detail-comparison__table-wrap,
.detail-comparison__segments {
  min-width: 0;
}

.detail-comparison__table-wrap {
  overflow-x: auto;
}

.detail-comparison__body h3 {
  margin: 0 0 6px;
  font-size: 0.86rem;
  font-weight: 800;
}

.detail-comparison__table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.82rem;
}

.detail-comparison__table th,
.detail-comparison__table td {
  border-top: 1px solid var(--ms-border);
  padding: 6px 5px;
  text-align: left;
  vertical-align: top;
}

.detail-comparison__table th {
  color: var(--ms-text-muted);
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.detail-comparison__table a {
  color: var(--ms-text);
  font-weight: 800;
  text-decoration: none;
}

.detail-comparison__table small {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.72rem;
}

.detail-comparison__segments ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-comparison__segments li {
  border-top: 1px solid var(--ms-border);
  padding-top: 6px;
  display: flex;
  justify-content: space-between;
  gap: 8px;
}

.detail-comparison__segments strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-comparison__segments span,
.detail-comparison__empty {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.detail-comparison__empty {
  margin: 0;
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

.detail-effort-tabs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  margin-bottom: 10px;
}

.detail-effort-tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-muted);
  font-size: 0.84rem;
  font-weight: 700;
  cursor: pointer;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.detail-effort-tab span {
  min-width: 20px;
  border-radius: 999px;
  background: #f1f5f9;
  color: #475569;
  padding: 1px 6px;
  font-size: 0.72rem;
}

.detail-effort-tab--active {
  border-color: #cfdcff;
  background: #eff5ff;
  color: #1d4ed8;
}

.detail-effort-tab--active span {
  background: #dbe8ff;
  color: #1d4ed8;
}

.detail-effort-filters {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr) minmax(0, 1fr);
  gap: 6px;
  margin-bottom: 10px;
}

.detail-effort-filters label {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.detail-effort-filters span {
  color: var(--ms-text-muted);
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.detail-effort-filters input,
.detail-effort-filters select {
  width: 100%;
  min-height: 32px;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text);
  font-size: 0.78rem;
  padding: 5px 8px;
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
  display: grid;
  gap: 2px;
  flex: 1;
  min-width: 0;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  color: var(--ms-text);
  font-size: 0.9rem;
}

.radio-label > span:first-child,
.radio-label small {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.radio-label small {
  color: var(--ms-muted);
  font-size: 0.74rem;
  font-weight: 600;
}

.radio-label--active {
  color: #1d4ed8;
}

.effort-option__badges {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.effort-option__badges span {
  border-radius: 999px;
  background: #fef3c7;
  color: #92400e;
  padding: 1px 6px;
  font-size: 0.68rem;
  font-weight: 800;
}

.detail-empty-state {
  margin: 10px 0 0;
  color: var(--ms-muted);
  font-size: 0.9rem;
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

.detail-chart-controls {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.detail-chart-controls label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 30px;
  border: 1px solid var(--ms-border);
  border-radius: 999px;
  background: #ffffff;
  color: var(--ms-text);
  padding: 3px 9px;
  font-size: 0.76rem;
  font-weight: 800;
}

.detail-chart-controls input {
  margin: 0;
}

.detail-chart-toggle--disabled {
  color: var(--ms-text-muted) !important;
  background: #f8fafc !important;
}

.detail-chart {
  width: 100%;
  height: 420px;
}

.switch-container {
  display: flex;
  justify-content: flex-end;
}

:global(.detailed-activity-tooltip .tooltip-inner) {
  text-align: left;
}

:global(.detailed-activity-tooltip .tooltip-inner ul),
:global(.detailed-activity-tooltip .tooltip-inner li) {
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
  .detail-kpi-grid,
  .detail-insight-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-map-layout,
  .detail-comparison__body {
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
  .detail-kpi-grid,
  .detail-highlights,
  .detail-insight-grid,
  .detail-effort-filters {
    grid-template-columns: 1fr;
  }

  .detail-hero {
    flex-direction: column;
  }

  .detail-hero__actions {
    width: 100%;
    justify-content: flex-start;
  }

  .detail-comparison__status {
    align-self: flex-start;
  }

  .detail-comparison__table {
    min-width: 520px;
  }

  .detail-card__header--chart {
    align-items: flex-start;
    flex-direction: column;
  }

  .detail-chart-controls {
    justify-content: flex-start;
  }
}
</style>

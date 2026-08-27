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
      <ActivityDetailHero
        :activity-name="activity?.name ?? ''"
        :activity-type-label="activityTypeLabel"
        :activity-date-label="activityDateLabel"
        :commute="activity?.commute ?? false"
        :activity-version="activityVersion"
        :activity-version-label="activityVersionLabel"
        :effort-count-label="effortCountLabel"
        :can-select-corrected-version="canSelectCorrectedVersion"
        :strava-activity-url="stravaActivityUrl"
        @version-change="switchActivityVersion"
      />

      <section class="detail-kpi-grid">
        <article
          v-for="kpi in kpis"
          :key="kpi.label"
          class="detail-kpi-card"
        >
          <span class="detail-kpi-card__label detail-label-with-tooltip">
            <span>{{ kpi.label }}</span>
            <TooltipHint
              v-if="metricTooltip(kpi)"
              :text="metricTooltip(kpi) ?? ''"
            />
          </span>
          <strong class="detail-kpi-card__value">{{ kpi.value }}</strong>
          <small v-if="kpi.hint" class="detail-kpi-card__hint">{{ kpi.hint }}</small>
        </article>
      </section>

      <section
        v-if="hikingInsightRows.length > 0"
        class="detail-card detail-hiking-insights"
      >
        <header class="detail-card__header">
          <h2>Hiking Insights</h2>
        </header>
        <div class="detail-hiking-insights__grid">
          <div
            v-for="row in hikingInsightRows"
            :key="row.label"
            class="detail-hiking-insight"
            :class="row.tone ? `detail-hiking-insight--${row.tone}` : undefined"
          >
            <span class="detail-hiking-insight__label detail-label-with-tooltip">
              <span>{{ row.label }}</span>
              <TooltipHint
                v-if="metricTooltip(row)"
                :text="metricTooltip(row) ?? ''"
              />
            </span>
            <strong>{{ row.value }}</strong>
            <small v-if="row.hint">{{ row.hint }}</small>
          </div>
        </div>
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
              <dt>
                <span class="detail-label-with-tooltip">
                  <span>{{ row.label }}</span>
                  <TooltipHint
                    v-if="metricTooltip(row)"
                    :text="metricTooltip(row) ?? ''"
                  />
                </span>
              </dt>
              <dd>
                <strong>{{ row.value }}</strong>
                <small v-if="row.hint">{{ row.hint }}</small>
              </dd>
            </div>
          </dl>
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
                <dt>
                  <span class="detail-label-with-tooltip">
                    <span>{{ row.label }}</span>
                    <TooltipHint
                      v-if="metricTooltip(row)"
                      :text="metricTooltip(row) ?? ''"
                    />
                  </span>
                </dt>
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
              <dt>
                <span class="detail-label-with-tooltip">
                  <span>{{ row.label }}</span>
                  <TooltipHint
                    v-if="metricTooltip(row)"
                    :text="metricTooltip(row) ?? ''"
                  />
                </span>
              </dt>
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
              <dt>
                <span class="detail-label-with-tooltip">
                  <span>{{ row.label }}</span>
                  <TooltipHint
                    v-if="metricTooltip(row)"
                    :text="metricTooltip(row) ?? ''"
                  />
                </span>
              </dt>
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
import type { ActivitySourceConflict, ActivitySourceRef, DetailedActivity, StravaSegmentEffort } from "@/models/activity.model";
import { formatActivityTypeLabel, formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import { useContextStore } from "@/stores/context.js";
import { useAthleteStore } from "@/stores/athlete";
import { useStatisticsStore } from "@/stores/statistics";
import {
  computeHeartRateZoneDistribution,
  resolveHeartRateZoneSettings,
} from "@/utils/heart-rate-zones";
import {
} from "@/models/athlete-performance-settings.model";
import { bestAveragePower, buildPowerAnalysis, formatOptionalDecimal, formatPowerZoneTime, type PowerAnalysis } from "@/services/activity-power-analysis";
import { ErrorService } from "@/services/error.service";
import { fetchResponse } from "@/services/http-client";
import { apiUrl } from "@/services/api-url";
import TooltipHint from "@/components/TooltipHint.vue";
import { getMetricTooltip } from "@/utils/metric-tooltips";
import { buildHikingInsights, type HikingDifficultyLabel } from "@/utils/hiking-insights";
import type { Options, SeriesAreaOptions, SeriesLineOptions } from "highcharts";
import Highcharts from "highcharts";
import { Chart } from "highcharts-vue";
import PowerDistributionChart from "@/components/charts/PowerDistributionChart.vue";
import PowerCurveDetailsChart from "@/components/charts/PowerCurveDetailsChart.vue";
import ActivityDetailHero from "@/components/activity-detail/ActivityDetailHero.vue";

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

const stravaActivityUrl = computed(() => activity.value?.link?.trim() ?? "");
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
  if (activity.value?.startDateLocal) {
    return formatLocalActivityDate(rawDate);
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
  tooltip?: string;
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
  tooltip?: string;
  tone?: "muted" | "good" | "warn";
};

const cadenceUnit = computed(() => effectiveActivityType.value.endsWith("Run") ? "spm" : "rpm");
const hikingInsights = computed(() => buildHikingInsights(activity.value));

const hikingInsightRows = computed<DetailMetricRow[]>(() => {
  const insights = hikingInsights.value;
  if (!insights) {
    return [];
  }

  const hasAltitudeStream = Boolean(activity.value?.stream?.altitude?.length);

  return [
    {
      label: "Hiking Difficulty",
      value: insights.difficultyLabel,
      hint: `Score ${insights.difficultyScore.toFixed(1)}`,
      tone: hikingDifficultyTone(insights.difficultyLabel),
    },
    {
      label: "Elevation per km",
      value: formatOptionalDecimal(insights.elevationPerKm, "m/km", 0),
    },
    {
      label: "Max continuous climb",
      value: formatOptionalDecimal(insights.maxContinuousClimbMeters, "m", 0),
      hint: hasAltitudeStream ? "From altitude stream" : "Altitude stream missing",
      tone: insights.maxContinuousClimbMeters === null ? "muted" : undefined,
    },
    {
      label: "Highest point",
      value: formatOptionalDecimal(insights.highestPointMeters, "m", 0),
      hint: hasAltitudeStream ? "From altitude stream" : "From activity summary",
    },
    {
      label: "Vertical speed",
      value: formatOptionalDecimal(insights.verticalSpeedMetersPerHour, "m/h", 0),
      hint: "D+ per moving hour",
    },
    {
      label: "Pause ratio",
      value: insights.pauseRatio !== null ? `${(insights.pauseRatio * 100).toFixed(0)}%` : "n/a",
      hint: insights.pausedSeconds > 0 ? `${formatTime(insights.pausedSeconds)} stopped` : "No stopped time",
    },
    {
      label: "Moving vs elapsed",
      value: `${formatTime(insights.movingTimeSeconds)} / ${formatTime(insights.elapsedTimeSeconds)}`,
    },
  ];
});


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
  const currentActivity = activity.value;
  const streamCount = streamAvailabilityRows.value.filter((row) => row.value !== "Missing").length;
  const source = currentActivity?.source;
  const rows: DetailMetricRow[] = [
    { label: "Displayed view", value: activityVersionLabel.value },
  ];

  if (source) {
    rows.push(
      {
        label: "Primary data",
        value: formatProviderLabel(source.primaryProvider),
        hint: `Activity ID ${source.primaryId}`,
      },
      {
        label: "Detailed stream",
        value: formatProviderLabel(source.streamProvider || source.fieldSources?.detailedStream || source.primaryProvider),
        hint: "GPS, altitude, speed, heart rate, cadence and power streams",
      },
      {
        label: "Matched sources",
        value: formatActivitySourceRefs(source.sources),
        hint: source.mergeConfidence ? `Match confidence ${source.mergeConfidence}` : undefined,
      },
    );
    if ((source.conflicts?.length ?? 0) > 0) {
      rows.push({
        label: "Source conflicts",
        value: `${source.conflicts?.length ?? 0} field${(source.conflicts?.length ?? 0) > 1 ? "s" : ""}`,
        hint: formatSourceConflictSample(source.conflicts),
        tone: "warn",
      });
    }
  } else {
    rows.push({
      label: "Primary data",
      value: "Single source",
      hint: "No composite provenance reported",
      tone: "muted",
    });
  }

  rows.push(
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
  );

  return rows;
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
  if (analysis.powerZoneEstimate !== null) {
    rows.push({
      label: "Aerobic power-zone time",
      value: formatPowerZoneTime(
        analysis.powerZoneEstimate.aerobicSeconds,
        analysis.powerZoneEstimate.trackedSeconds,
      ),
    });
    rows.push({
      label: "Threshold / VO2 time",
      value: formatPowerZoneTime(
        analysis.powerZoneEstimate.thresholdVo2Seconds,
        analysis.powerZoneEstimate.trackedSeconds,
      ),
    });
    rows.push({
      label: "Anaerobic exposure",
      value: formatPowerZoneTime(
        analysis.powerZoneEstimate.anaerobicSeconds,
        analysis.powerZoneEstimate.trackedSeconds,
      ),
    });
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


function hikingDifficultyTone(label: HikingDifficultyLabel): DetailMetricRow["tone"] {
  if (label === "Easy" || label === "Moderate") {
    return "good";
  }
  if (label === "Epic" || label === "Very hard") {
    return "warn";
  }
  return undefined;
}


function metricTooltip(metric: { label: string; tooltip?: string }): string | null {
  return metric.tooltip ?? getMetricTooltip(metric.label);
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

function formatLocalActivityDate(value: string): string {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})[T\s](\d{2}):(\d{2})/);
  if (!match) {
    return value.substring(0, 16);
  }
  const [, year, month, day, hour, minute] = match;
  const localDate = new Date(Number(year), Number(month) - 1, Number(day), Number(hour), Number(minute));
  return localDate.toLocaleString("en-US", {
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatProviderLabel(provider: string | undefined): string {
  const normalized = (provider ?? "").trim().toLowerCase();
  if (normalized === "strava") return "Strava";
  if (normalized === "fit") return "FIT";
  if (normalized === "gpx") return "GPX";
  if (normalized === "ridewithgps") return "RideWithGPS";
  return provider || "Unknown";
}

function formatActivitySourceRefs(sources: ActivitySourceRef[] | undefined): string {
  if (!Array.isArray(sources) || sources.length === 0) {
    return "n/a";
  }
  return sources
    .map((source) => `${formatProviderLabel(source.provider)} #${source.activityId}${source.hasStream ? " + stream" : ""}`)
    .join(" · ");
}

function formatSourceConflictSample(conflicts: ActivitySourceConflict[] | undefined): string | undefined {
  if (!Array.isArray(conflicts) || conflicts.length === 0) {
    return undefined;
  }
  const conflict = conflicts[0];
  return `${conflict.field}: ${formatProviderLabel(conflict.source)} ${conflict.primary} -> ${conflict.other}`;
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
  const url = apiUrl("getActivity", {
    path: { activityId: id },
    query: { version: version === "raw" ? "raw" : undefined },
  });
  const response = await fetchResponse(url);
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
    const rawLatlngs = activity.value?.stream?.latlng;
    if (rawLatlngs) {
      const segments = splitMapTraceAtRecordingGaps(rawLatlngs, activity.value?.stream?.time ?? []);
      const filteredLatlngs = segments.flat();
      if (filteredLatlngs.length === 0) {
        if (basePolyline.value) {
          basePolyline.value.remove();
          basePolyline.value = null;
        }
        return;
      }

      if (basePolyline.value) {
        basePolyline.value.setLatLngs(segments);
      } else {
        basePolyline.value = L.polyline(segments, {
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

function splitMapTraceAtRecordingGaps(rawLatlngs: number[][], times: number[]): L.LatLng[][] {
  const validDeltas = times
    .slice(1)
    .map((time, index) => time - times[index])
    .filter((delta) => Number.isFinite(delta) && delta > 0)
    .sort((left, right) => left - right);
  const medianCadence = validDeltas.length > 0 ? validDeltas[Math.floor((validDeltas.length - 1) / 2)] : 1;
  const gapThresholdSeconds = Math.max(30, medianCadence * 10);
  const segments: L.LatLng[][] = [];
  let currentSegment: L.LatLng[] = [];

  rawLatlngs.forEach((rawPoint, index) => {
    const latitude = rawPoint[0];
    const longitude = rawPoint[1];
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude) || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180) {
      if (currentSegment.length > 0) segments.push(currentSegment);
      currentSegment = [];
      return;
    }
    const point = L.latLng(latitude, longitude);
    if (index > 0 && currentSegment.length > 0 && index < times.length) {
      const deltaSeconds = times[index] - times[index - 1];
      const previousPoint = currentSegment[currentSegment.length - 1];
      if (deltaSeconds >= gapThresholdSeconds && previousPoint.distanceTo(point) >= 200) {
        segments.push(currentSegment);
        currentSegment = [];
      }
    }
    currentSegment.push(point);
  });
  if (currentSegment.length > 0) segments.push(currentSegment);
  return segments;
}

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

<style src="../assets/views/detailed-activity-view.css"></style>

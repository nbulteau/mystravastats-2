<script setup lang="ts">
import { Chart } from "highcharts-vue";
import type { Options } from "highcharts";
import { computed, ref, watch, onMounted } from "vue";
import { RouterLink } from "vue-router";
import type {
  SegmentEffort,
  SegmentTargetSummary,
} from "@/models/segment-analysis.model";
import { useContextStore } from "@/stores/context";
import { useSegmentsStore } from "@/stores/segments";
import { formatTime } from "@/utils/formatters";

type SegmentMetric = "TIME" | "SPEED";
type SegmentKindFilter = "ALL" | "CLIMB" | "SEGMENT";
type SegmentSort = "ATTEMPTS" | "CLOSE_TO_PR" | "NAME";
type ChartMode = "PERFORMANCE" | "PHYSIOLOGY";

const contextStore = useContextStore();
const segmentsStore = useSegmentsStore();
onMounted(() => contextStore.updateCurrentView("segments"));

const filterMetric = ref<SegmentMetric>(segmentsStore.metric);
const filterQuery = ref<string>(segmentsStore.query);
const filterFrom = ref<string>(segmentsStore.from);
const filterTo = ref<string>(segmentsStore.to);
const filterKind = ref<SegmentKindFilter>("ALL");
const filterMinAttempts = ref<number>(2);
const filterSort = ref<SegmentSort>("ATTEMPTS");
const chartMode = ref<ChartMode>("PERFORMANCE");

const isLoading = ref<boolean>(false);

const segments = computed(() => segmentsStore.segments);
const efforts = computed(() => segmentsStore.efforts);
const summary = computed(() => segmentsStore.summary);
const selectedMetric = computed(() => segmentsStore.metric);

function toSortableTimestamp(value: string): number {
  const normalized = value.includes("T") ? value : value.replace(" ", "T");
  const parsed = Date.parse(normalized);
  if (!Number.isNaN(parsed)) {
    return parsed;
  }
  const dayPrefix = value.slice(0, 10);
  if (/^\d{4}-\d{2}-\d{2}$/.test(dayPrefix)) {
    const dayParsed = Date.parse(`${dayPrefix}T00:00:00Z`);
    if (!Number.isNaN(dayParsed)) {
      return dayParsed;
    }
  }
  return 0;
}

function segmentKind(segment: SegmentTargetSummary): SegmentKindFilter {
  return segment.climbCategory > 0 ? "CLIMB" : "SEGMENT";
}

const filteredSegments = computed(() => {
  const minimumAttempts = Math.max(1, filterMinAttempts.value || 1);

  const visible = segments.value.filter((segment) => {
    if (filterKind.value !== "ALL" && segmentKind(segment) !== filterKind.value) {
      return false;
    }
    return segment.attemptsCount >= minimumAttempts;
  });

  return visible.sort((left, right) => {
    switch (filterSort.value) {
      case "NAME":
        return left.targetName.localeCompare(right.targetName);
      case "CLOSE_TO_PR":
        if (left.closeToPrCount !== right.closeToPrCount) {
          return right.closeToPrCount - left.closeToPrCount;
        }
        if (left.attemptsCount !== right.attemptsCount) {
          return right.attemptsCount - left.attemptsCount;
        }
        return left.targetName.localeCompare(right.targetName);
      case "ATTEMPTS":
      default:
        if (left.attemptsCount !== right.attemptsCount) {
          return right.attemptsCount - left.attemptsCount;
        }
        return left.targetName.localeCompare(right.targetName);
    }
  });
});

watch(
  filteredSegments,
  (nextSegments) => {
    if (nextSegments.length === 0) {
      return;
    }
    const selectedId = segmentsStore.selectedSegmentId;
    if (!selectedId || !nextSegments.some((segment) => segment.targetId === selectedId)) {
      void selectSegment(nextSegments[0].targetId);
    }
  },
  { immediate: true },
);

const selectedSegment = computed(() =>
  filteredSegments.value.find((segment) => segment.targetId === segmentsStore.selectedSegmentId),
);

const orderedEfforts = computed(() => {
  return [...efforts.value].sort((left, right) => {
    const leftTs = toSortableTimestamp(left.activityDate);
    const rightTs = toSortableTimestamp(right.activityDate);
    if (leftTs === rightTs) {
      return left.activityDate.localeCompare(right.activityDate);
    }
    return leftTs - rightTs;
  });
});

const topEfforts = computed(() => {
  if (summary.value?.topEfforts?.length) {
    return summary.value.topEfforts;
  }
  return orderedEfforts.value
    .slice()
    .sort((left, right) => {
      if (selectedMetric.value === "TIME") {
        return left.elapsedTimeSeconds - right.elapsedTimeSeconds;
      }
      return right.speedKph - left.speedKph;
    })
    .slice(0, 3);
});

const prEventsCount = computed(() =>
  orderedEfforts.value.filter((attempt) => attempt.setsNewPr).length,
);

function valueForMetric(attempt: SegmentEffort): number {
  if (selectedMetric.value === "TIME") {
    return attempt.elapsedTimeSeconds;
  }
  return attempt.speedKph;
}

function average(values: number[]): number | null {
  if (values.length === 0) {
    return null;
  }
  return values.reduce((total, value) => total + value, 0) / values.length;
}

function computeRecentTrendLabel(series: number[]): string {
  if (series.length < 6) {
    return "Not enough data";
  }
  const recent = average(series.slice(-3));
  const previous = average(series.slice(-6, -3));
  if (recent === null || previous === null || previous === 0) {
    return "Stable";
  }

  const ratio = (recent - previous) / previous;
  const isTime = selectedMetric.value === "TIME";
  const improving = isTime ? ratio < 0 : ratio > 0;
  const changePercent = Math.abs(ratio * 100);

  if (changePercent < 0.8) {
    return "Stable";
  }
  return `${improving ? "Improving" : "Declining"} ${changePercent.toFixed(1)}%`;
}

const performanceInsights = computed(() => {
  const attempts = orderedEfforts.value;
  if (attempts.length === 0) {
    return {
      firstAttemptLabel: "-",
      latestAttemptLabel: "-",
      progressionLabel: "-",
      recentTrendLabel: "Not enough data",
      averagePowerLabel: "-",
      averageHeartRateLabel: "-",
    };
  }

  const firstAttempt = attempts[0];
  const latestAttempt = attempts[attempts.length - 1];
  const personalBest = topEfforts.value[0] ?? firstAttempt;

  let progressionLabel = "-";
  if (selectedMetric.value === "TIME") {
    const savedSeconds = firstAttempt.elapsedTimeSeconds - personalBest.elapsedTimeSeconds;
    progressionLabel = savedSeconds > 0
      ? `${formatTime(savedSeconds)} faster`
      : "No gain yet";
  } else {
    const gainKph = personalBest.speedKph - firstAttempt.speedKph;
    progressionLabel = gainKph > 0
      ? `+${gainKph.toFixed(2)} km/h`
      : "No gain yet";
  }

  const trendSeries = attempts.map((attempt) => valueForMetric(attempt));
  const trendLabel = computeRecentTrendLabel(trendSeries);

  const powerValues = attempts
    .map((attempt) => attempt.averagePowerWatts)
    .filter((value) => value > 0);
  const heartRateValues = attempts
    .map((attempt) => attempt.averageHeartRate)
    .filter((value) => value > 0);

  return {
    firstAttemptLabel: formatEffortValue(firstAttempt),
    latestAttemptLabel: formatEffortValue(latestAttempt),
    progressionLabel,
    recentTrendLabel: trendLabel,
    averagePowerLabel: powerValues.length > 0
      ? `${Math.round(average(powerValues) ?? 0)} W`
      : "-",
    averageHeartRateLabel: heartRateValues.length > 0
      ? `${Math.round(average(heartRateValues) ?? 0)} bpm`
      : "-",
  };
});

function normalizeDateRange() {
  let from = filterFrom.value.trim();
  let to = filterTo.value.trim();
  if (from && to && from > to) {
    const nextFrom = to;
    const nextTo = from;
    from = nextFrom;
    to = nextTo;
  }
  filterFrom.value = from;
  filterTo.value = to;
}

async function applyFilters(force = false) {
  normalizeDateRange();
  segmentsStore.updateFilters({
    metric: filterMetric.value,
    query: filterQuery.value,
    from: filterFrom.value,
    to: filterTo.value,
  });
  if (force) {
    segmentsStore.invalidateCache();
  }
  isLoading.value = true;
  try {
    await segmentsStore.ensureLoaded(force);
  } finally {
    isLoading.value = false;
  }
}

async function resetFilters() {
  filterQuery.value = "";
  filterFrom.value = "";
  filterTo.value = "";
  filterKind.value = "ALL";
  filterMinAttempts.value = 2;
  filterSort.value = "ATTEMPTS";
  chartMode.value = "PERFORMANCE";
  await applyFilters();
}

async function refreshData() {
  await applyFilters(true);
}

async function selectSegment(segmentId: number) {
  isLoading.value = true;
  try {
    await segmentsStore.selectSegment(segmentId);
  } finally {
    isLoading.value = false;
  }
}

function formatDate(value: string): string {
  return value.slice(0, 10);
}

function formatSpeed(value: number): string {
  return `${value.toFixed(2)} km/h`;
}

function formatDistance(valueMeters: number): string {
  return `${(valueMeters / 1000).toFixed(2)} km`;
}

function formatPower(value: number): string {
  if (value <= 0) {
    return "-";
  }
  return `${Math.round(value)} W`;
}

function formatHeartRate(value: number): string {
  if (value <= 0) {
    return "-";
  }
  return `${Math.round(value)} bpm`;
}

function formatCategory(category: number): string {
  if (category <= 0) {
    return "Segment";
  }
  if (category === 5) {
    return "Cat. HC";
  }
  return `Cat. ${6 - category}`;
}

function formatEffortValue(attempt: SegmentEffort): string {
  if (selectedMetric.value === "TIME") {
    return formatTime(attempt.elapsedTimeSeconds);
  }
  return formatSpeed(attempt.speedKph);
}

const chartOptions = computed<Options>(() => {
  const categories = orderedEfforts.value.map((attempt) => formatDate(attempt.activityDate));
  const timeData = orderedEfforts.value.map((attempt) => attempt.elapsedTimeSeconds);
  const speedData = orderedEfforts.value.map((attempt) => Number(attempt.speedKph.toFixed(2)));
  const powerData = orderedEfforts.value.map((attempt) =>
    attempt.averagePowerWatts > 0 ? Number(attempt.averagePowerWatts.toFixed(1)) : null,
  );
  const heartRateData = orderedEfforts.value.map((attempt) =>
    attempt.averageHeartRate > 0 ? Number(attempt.averageHeartRate.toFixed(1)) : null,
  );

  const sharedTooltipFormatter = function (this: any): string {
    const pointIndex = this.points?.[0]?.point?.index as number | undefined;
    const attempt = pointIndex !== undefined ? orderedEfforts.value[pointIndex] : undefined;
    if (!attempt) {
      return "";
    }
    const lines = [
      `<b>${formatDate(attempt.activityDate)}</b>`,
      `Activity: ${attempt.activity.name}`,
      `Time: <b>${formatTime(attempt.elapsedTimeSeconds)}</b>`,
      `Speed: <b>${formatSpeed(attempt.speedKph)}</b>`,
    ];

    if (attempt.averagePowerWatts > 0) {
      lines.push(`Power: <b>${formatPower(attempt.averagePowerWatts)}</b>`);
    }
    if (attempt.averageHeartRate > 0) {
      lines.push(`HR: <b>${formatHeartRate(attempt.averageHeartRate)}</b>`);
    }
    if (attempt.personalRank) {
      lines.push(`Personal rank: <b>#${attempt.personalRank}</b>`);
    }
    if (attempt.setsNewPr) {
      lines.push(`<span style="color:#fc4c02;font-weight:700;">New PR</span>`);
    }
    return lines.join("<br/>");
  };

  if (chartMode.value === "PHYSIOLOGY") {
    return {
      chart: {
        type: "line",
        height: 300,
        animation: false,
      },
      title: {
        text: "Power & heart-rate trend",
      },
      xAxis: {
        categories,
        tickLength: 0,
      },
      yAxis: [
        {
          title: { text: "Power (W)" },
        },
        {
          title: { text: "Heart rate (bpm)" },
          opposite: true,
        },
      ],
      tooltip: {
        shared: true,
        formatter: sharedTooltipFormatter,
      },
      series: [
        {
          type: "line",
          name: "Power",
          yAxis: 0,
          data: powerData,
          color: "#f28e2b",
        },
        {
          type: "line",
          name: "Heart rate",
          yAxis: 1,
          data: heartRateData,
          color: "#e15759",
        },
      ],
      credits: { enabled: false },
    };
  }

  const timeColor = selectedMetric.value === "TIME" ? "#fc4c02" : "#4e79a7";
  const speedColor = selectedMetric.value === "SPEED" ? "#fc4c02" : "#59a14f";

  return {
    chart: {
      type: "line",
      height: 300,
      animation: false,
    },
    title: {
      text: "Performance trend",
    },
    xAxis: {
      categories,
      tickLength: 0,
      title: {
        text: "Date",
      },
    },
    yAxis: [
      {
        title: {
          text: "Elapsed time",
        },
        reversed: true,
      },
      {
        title: {
          text: "Speed (km/h)",
        },
        opposite: true,
      },
    ],
    tooltip: {
      shared: true,
      formatter: sharedTooltipFormatter,
    },
    series: [
      {
        type: "line",
        name: "Elapsed time",
        yAxis: 0,
        data: timeData,
        color: timeColor,
      },
      {
        type: "line",
        name: "Speed",
        yAxis: 1,
        data: speedData,
        color: speedColor,
      },
    ],
    credits: { enabled: false },
  };
});
</script>

<template>
  <div class="segments-page">
    <section class="segments-toolbar">
      <div class="segments-toolbar-row">
        <div class="segments-field">
          <label for="segmentMetric">Metric</label>
          <select
            id="segmentMetric"
            v-model="filterMetric"
            class="form-select form-select-sm"
          >
            <option value="TIME">Time</option>
            <option value="SPEED">Speed</option>
          </select>
        </div>
        <div class="segments-field segments-field--grow">
          <label for="segmentQuery">Segment name</label>
          <input
            id="segmentQuery"
            v-model="filterQuery"
            type="text"
            class="form-control form-control-sm"
            placeholder="Filter by segment name"
            @keyup.enter="applyFilters()"
          >
        </div>
        <div class="segments-field">
          <label for="segmentFrom">From</label>
          <input
            id="segmentFrom"
            v-model="filterFrom"
            type="date"
            class="form-control form-control-sm"
          >
        </div>
        <div class="segments-field">
          <label for="segmentTo">To</label>
          <input
            id="segmentTo"
            v-model="filterTo"
            type="date"
            class="form-control form-control-sm"
          >
        </div>
      </div>

      <div class="segments-toolbar-row segments-toolbar-row--secondary">
        <div class="segments-field">
          <label for="segmentKind">Target type</label>
          <select
            id="segmentKind"
            v-model="filterKind"
            class="form-select form-select-sm"
          >
            <option value="ALL">All</option>
            <option value="CLIMB">Climbs</option>
            <option value="SEGMENT">Flat segments</option>
          </select>
        </div>
        <div class="segments-field">
          <label for="segmentMinAttempts">Min attempts</label>
          <input
            id="segmentMinAttempts"
            v-model.number="filterMinAttempts"
            type="number"
            min="1"
            max="100"
            class="form-control form-control-sm"
          >
        </div>
        <div class="segments-field">
          <label for="segmentSort">Sort by</label>
          <select
            id="segmentSort"
            v-model="filterSort"
            class="form-select form-select-sm"
          >
            <option value="ATTEMPTS">Attempts</option>
            <option value="CLOSE_TO_PR">Close to PR</option>
            <option value="NAME">Name</option>
          </select>
        </div>
        <div class="segments-actions">
          <button
            type="button"
            class="btn btn-sm btn-outline-secondary"
            :disabled="isLoading"
            @click="resetFilters"
          >
            Reset
          </button>
          <button
            type="button"
            class="btn btn-sm btn-outline-secondary"
            :disabled="isLoading"
            @click="refreshData"
          >
            Refresh
          </button>
          <button
            type="button"
            class="btn btn-sm btn-primary"
            :disabled="isLoading"
            @click="applyFilters()"
          >
            Apply
          </button>
        </div>
      </div>
    </section>

    <section
      v-if="filteredSegments.length === 0"
      class="segments-empty"
    >
      No repeated segments found for the selected filters. Try widening the date range, lowering min attempts, or switching filters.
    </section>

    <section
      v-else
      class="segments-layout"
    >
      <aside class="segments-list">
        <button
          v-for="segment in filteredSegments"
          :key="segment.targetId"
          type="button"
          class="segments-list-item"
          :class="{ 'segments-list-item--active': segment.targetId === selectedSegment?.targetId }"
          @click="selectSegment(segment.targetId)"
        >
          <span class="segments-list-item__title">{{ segment.targetName }}</span>
          <span class="segments-list-item__meta">
            {{ formatCategory(segment.climbCategory) }} · {{ segment.attemptsCount }} attempts
          </span>
          <span class="segments-list-item__meta">
            Best: {{ segment.bestValue }}
          </span>
          <span class="segments-list-item__meta">
            Trend: {{ segment.recentTrend }}
          </span>
        </button>
      </aside>

      <article
        v-if="selectedSegment"
        class="segments-content"
      >
        <header class="segments-header">
          <h3>{{ selectedSegment.targetName }}</h3>
          <p>
            {{ formatCategory(selectedSegment.climbCategory) }} ·
            {{ formatDistance(selectedSegment.distance) }} ·
            {{ selectedSegment.averageGrade.toFixed(1) }}% avg grade
          </p>
        </header>

        <div class="segments-kpis">
          <div class="segments-kpi">
            <span class="segments-kpi__label">Attempts</span>
            <strong>{{ selectedSegment.attemptsCount }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">PR events</span>
            <strong>{{ prEventsCount }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">Personal record</span>
            <strong>{{ summary?.personalRecord ? formatEffortValue(summary.personalRecord) : selectedSegment.bestValue }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">Latest effort</span>
            <strong>{{ performanceInsights.latestAttemptLabel }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">First → PR gain</span>
            <strong>{{ performanceInsights.progressionLabel }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">Recent form (3 vs 3)</span>
            <strong>{{ performanceInsights.recentTrendLabel }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">Average power</span>
            <strong>{{ performanceInsights.averagePowerLabel }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">Average heart rate</span>
            <strong>{{ performanceInsights.averageHeartRateLabel }}</strong>
          </div>
          <div class="segments-kpi">
            <span class="segments-kpi__label">Close to PR</span>
            <strong>{{ selectedSegment.closeToPrCount }}</strong>
          </div>
        </div>

        <div class="segments-chart-toolbar">
          <div class="btn-group btn-group-sm">
            <button
              type="button"
              class="btn"
              :class="chartMode === 'PERFORMANCE' ? 'btn-primary' : 'btn-outline-secondary'"
              @click="chartMode = 'PERFORMANCE'"
            >
              Performance
            </button>
            <button
              type="button"
              class="btn"
              :class="chartMode === 'PHYSIOLOGY' ? 'btn-primary' : 'btn-outline-secondary'"
              @click="chartMode = 'PHYSIOLOGY'"
            >
              Physiology
            </button>
          </div>
        </div>

        <div class="segments-chart">
          <Chart :options="chartOptions" />
        </div>

        <div
          v-if="topEfforts.length > 0"
          class="segments-top"
        >
          <h4>Top efforts</h4>
          <div class="segments-top-grid">
            <div
              v-for="(attempt, index) in topEfforts"
              :key="`${attempt.activity.id}-${index}`"
              class="segments-top-item"
            >
              <span class="segments-top-item__rank">#{{ index + 1 }}</span>
              <strong>{{ formatEffortValue(attempt) }}</strong>
              <span>{{ formatDate(attempt.activityDate) }}</span>
              <span>{{ attempt.activity.name }}</span>
            </div>
          </div>
        </div>

        <div class="segments-table">
          <table class="table table-sm align-middle mb-0">
            <thead>
              <tr>
                <th>Date</th>
                <th>Value</th>
                <th>Moving</th>
                <th>Speed</th>
                <th>Power</th>
                <th>HR</th>
                <th>Rank</th>
                <th>Delta</th>
                <th>Activity</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="attempt in orderedEfforts"
                :key="`${attempt.activity.id}-${attempt.activityDate}-${attempt.elapsedTimeSeconds}`"
              >
                <td>{{ formatDate(attempt.activityDate) }}</td>
                <td>
                  <strong>{{ formatEffortValue(attempt) }}</strong>
                </td>
                <td>{{ formatTime(attempt.movingTimeSeconds) }}</td>
                <td>{{ formatSpeed(attempt.speedKph) }}</td>
                <td>{{ formatPower(attempt.averagePowerWatts) }}</td>
                <td>{{ formatHeartRate(attempt.averageHeartRate) }}</td>
                <td>{{ attempt.personalRank ? `#${attempt.personalRank}` : "-" }}</td>
                <td>
                  <span
                    class="segments-delta"
                    :class="{ 'segments-delta--pr': attempt.setsNewPr }"
                  >
                    {{ attempt.setsNewPr ? "New PR" : attempt.deltaToPr }}
                  </span>
                </td>
                <td>
                  <RouterLink :to="`/activities/${attempt.activity.id}`">
                    {{ attempt.activity.name }}
                  </RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </section>
  </div>
</template>

<style scoped>
.segments-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.segments-toolbar,
.segments-empty,
.segments-list,
.segments-content {
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
}

.segments-toolbar {
  padding: 10px;
}

.segments-toolbar-row {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.segments-toolbar-row--secondary {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--ms-border);
}

.segments-field {
  min-width: 120px;
}

.segments-field--grow {
  flex: 1;
  min-width: 220px;
}

.segments-field label {
  display: block;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--ms-text-muted);
  margin-bottom: 3px;
}

.segments-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.segments-empty {
  padding: 16px;
  color: var(--ms-text-muted);
}

.segments-layout {
  display: grid;
  grid-template-columns: 310px 1fr;
  gap: 12px;
}

.segments-list {
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 78vh;
  overflow-y: auto;
}

.segments-list-item {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  background: #ffffff;
  padding: 9px;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 2px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.segments-list-item:hover {
  border-color: #ffd2bf;
  box-shadow: 0 4px 10px rgba(252, 76, 2, 0.08);
  background: #fff9f6;
}

.segments-list-item--active {
  border-color: #ffb89a;
  background: #fff5f0;
}

.segments-list-item__title {
  font-weight: 700;
  color: var(--ms-text);
}

.segments-list-item__meta {
  color: var(--ms-text-muted);
  font-size: 0.8rem;
}

.segments-content {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.segments-header h3 {
  margin: 0;
  color: var(--ms-text);
}

.segments-header p {
  margin: 4px 0 0;
  color: var(--ms-text-muted);
}

.segments-kpis {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.segments-kpi {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: #ffffff;
}

.segments-kpi__label {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.02em;
  color: var(--ms-text-muted);
  font-weight: 700;
}

.segments-kpi strong {
  color: var(--ms-text);
}

.segments-chart-toolbar {
  display: flex;
  justify-content: flex-end;
}

.segments-chart {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  background: #ffffff;
  padding: 8px;
}

.segments-top {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  background: #ffffff;
  padding: 10px;
}

.segments-top h4 {
  margin: 0 0 8px;
  font-size: 1rem;
}

.segments-top-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.segments-top-item {
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  background: #fafbfd;
  gap: 2px;
}

.segments-top-item__rank {
  font-weight: 700;
  color: var(--ms-primary);
}

.segments-table {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  background: #ffffff;
  overflow-x: auto;
}

.segments-table th {
  white-space: nowrap;
  color: var(--ms-text-muted);
  font-size: 0.8rem;
  text-transform: uppercase;
}

.segments-table td {
  white-space: nowrap;
}

.segments-delta {
  font-weight: 600;
  color: var(--ms-text-muted);
}

.segments-delta--pr {
  color: var(--ms-primary);
}

@media (max-width: 1260px) {
  .segments-kpis {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1200px) {
  .segments-layout {
    grid-template-columns: 1fr;
  }

  .segments-list {
    max-height: none;
  }
}

@media (max-width: 900px) {
  .segments-kpis {
    grid-template-columns: 1fr;
  }

  .segments-top-grid {
    grid-template-columns: 1fr;
  }

  .segments-actions {
    margin-left: 0;
    width: 100%;
    justify-content: flex-end;
  }
}
</style>

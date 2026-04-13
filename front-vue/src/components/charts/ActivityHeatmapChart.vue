<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { Chart } from "highcharts-vue";
import { formatTime } from "@/utils/formatters";
import iconRide from "@/assets/buttons/road-bike.png";
import iconMountainBikeRide from "@/assets/buttons/mountain-bike.png";
import iconCommute from "@/assets/buttons/city-bike.png";
import iconGravelRide from "@/assets/buttons/touring-bike.png";
import iconVirtualRide from "@/assets/buttons/virtual-bike.png";
import iconRun from "@/assets/buttons/run.png";
import iconTrailRun from "@/assets/buttons/trail-run.png";
import iconHike from "@/assets/buttons/hike.png";
import iconAlpineSki from "@/assets/buttons/alpine-ski.png";
import type {
  ActivityHeatmap,
  HeatmapDayData,
  HeatmapMetricKey,
} from "@/models/activity-heatmap.model";

const MONTH_NAMES = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const DAY_LABELS = Array.from({ length: 31 }, (_, i) => String(i + 1));
const FIXED_MAX_BY_METRIC: Record<HeatmapMetricKey, number> = {
  distanceKm: 80,
  elevationGainM: 2200,
  durationSec: 4 * 3600,
};

const METRIC_OPTIONS: Array<{ key: HeatmapMetricKey; label: string; legendUnit: string }> = [
  { key: "distanceKm", label: "Distance", legendUnit: "km" },
  { key: "elevationGainM", label: "Elevation", legendUnit: "m" },
  { key: "durationSec", label: "Duration", legendUnit: "time" },
];
const ICON_BY_ACTIVITY_TYPE: Record<string, string> = {
  Ride: iconRide,
  MountainBikeRide: iconMountainBikeRide,
  Commute: iconCommute,
  GravelRide: iconGravelRide,
  VirtualRide: iconVirtualRide,
  Run: iconRun,
  TrailRun: iconTrailRun,
  Hike: iconHike,
  AlpineSki: iconAlpineSki,
};

const props = defineProps<{
  activityHeatmap: ActivityHeatmap;
  selectedYear?: string;
}>();

const router = useRouter();
const selectedMetric = ref<HeatmapMetricKey>("distanceKm");
const colorScaleMode = ref<"auto" | "fixed">("auto");
const selectedDayKey = ref<string | null>(null);

const availableYears = computed(() =>
  Object.keys(props.activityHeatmap).sort((a, b) => parseInt(b, 10) - parseInt(a, 10))
);

const hasData = computed(() => availableYears.value.length > 0);

const displayYear = computed(() => {
  if (!hasData.value) {
    return "";
  }
  if (
    props.selectedYear &&
    props.selectedYear !== "All years" &&
    availableYears.value.includes(props.selectedYear)
  ) {
    return props.selectedYear;
  }
  return availableYears.value[0] ?? "";
});

watch(displayYear, () => {
  selectedDayKey.value = null;
});

const isAllYearsSelected = computed(() => props.selectedYear === "All years");
const isFallbackYear = computed(
  () =>
    !!props.selectedYear &&
    props.selectedYear !== "All years" &&
    props.selectedYear !== displayYear.value
);

const currentYearData = computed<Record<string, HeatmapDayData>>(() => {
  if (!displayYear.value) {
    return {};
  }
  return props.activityHeatmap[displayYear.value] ?? {};
});

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

function dayKeyFromIndexes(monthIndex: number, dayIndex: number): string {
  return `${pad2(monthIndex + 1)}-${pad2(dayIndex + 1)}`;
}

function metricValue(day: HeatmapDayData, metric: HeatmapMetricKey): number {
  if (metric === "durationSec") {
    return day.durationSec ?? 0;
  }
  if (metric === "elevationGainM") {
    return day.elevationGainM ?? 0;
  }
  return day.distanceKm ?? 0;
}

type HeatmapPoint = {
  x: number;
  y: number;
  value: number;
  custom: {
    dayKey: string;
    activityTypes: string[];
    iconUrls: string[];
  };
};

function resolveDayActivityTypes(day: HeatmapDayData): string[] {
  if (!day.activities || day.activities.length === 0) {
    return [];
  }

  const uniqueTypes: string[] = [];
  const seenTypes = new Set<string>();
  day.activities.forEach((activity) => {
    const type = (activity.type ?? "").trim();
    if (!type || seenTypes.has(type)) {
      return;
    }
    seenTypes.add(type);
    uniqueTypes.push(type);
  });
  return uniqueTypes;
}

function resolveActivityTypeIcon(activityType: string | null): string | null {
  if (!activityType) {
    return null;
  }
  if (ICON_BY_ACTIVITY_TYPE[activityType]) {
    return ICON_BY_ACTIVITY_TYPE[activityType];
  }
  if (activityType.endsWith("Run")) {
    return iconRun;
  }
  if (activityType.includes("Ski")) {
    return iconAlpineSki;
  }
  if (activityType.includes("Ride") || activityType === "Commute") {
    return iconRide;
  }
  return null;
}

const heatmapPoints = computed<HeatmapPoint[]>(() => {
  const points: HeatmapPoint[] = [];
  Object.entries(currentYearData.value).forEach(([dayKey, day]) => {
    const [monthStr, dayStr] = dayKey.split("-");
    const month = parseInt(monthStr, 10) - 1;
    const dayIndex = parseInt(dayStr, 10) - 1;
    const activityTypes = resolveDayActivityTypes(day);
    const iconUrls = activityTypes
      .map((type) => resolveActivityTypeIcon(type))
      .filter((iconUrl): iconUrl is string => !!iconUrl);
    if (Number.isNaN(month) || Number.isNaN(dayIndex)) {
      return;
    }
    points.push({
      x: month,
      y: dayIndex,
      value: metricValue(day, selectedMetric.value),
      custom: {
        dayKey,
        activityTypes,
        iconUrls,
      },
    });
  });
  return points;
});

const maxMetricValue = computed(() => {
  const values = heatmapPoints.value.map((point) => point.value);
  const max = Math.max(...values, 0);
  return max > 0 ? max : 1;
});

const colorAxisMax = computed(() => {
  if (colorScaleMode.value === "fixed") {
    return FIXED_MAX_BY_METRIC[selectedMetric.value];
  }
  return maxMetricValue.value;
});

const selectedMetricMeta = computed(() =>
  METRIC_OPTIONS.find((option) => option.key === selectedMetric.value) ?? METRIC_OPTIONS[0]
);

function formatDistance(value: number): string {
  return `${(value ?? 0).toFixed(1)} km`;
}

function formatElevation(value: number): string {
  return `${Math.round(value ?? 0)} m`;
}

function formatDuration(value: number): string {
  return formatTime(Math.round(value ?? 0));
}

function formatMetric(value: number, metric: HeatmapMetricKey): string {
  if (metric === "durationSec") {
    return formatDuration(value);
  }
  if (metric === "elevationGainM") {
    return formatElevation(value);
  }
  return formatDistance(value);
}

function dayLabel(dayKey: string): string {
  if (!displayYear.value) {
    return dayKey;
  }
  const [monthStr, dayStr] = dayKey.split("-");
  const year = Number(displayYear.value);
  const month = Number(monthStr);
  const day = Number(dayStr);
  const date = new Date(year, month - 1, day);
  return new Intl.DateTimeFormat(navigator.language, {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(date);
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

const monthlySummary = computed(() =>
  MONTH_NAMES.map((label, index) => {
    const monthPrefix = `${pad2(index + 1)}-`;
    let distanceKm = 0;
    let elevationGainM = 0;
    let durationSec = 0;
    let activityCount = 0;

    Object.entries(currentYearData.value).forEach(([dayKey, day]) => {
      if (!dayKey.startsWith(monthPrefix)) {
        return;
      }
      distanceKm += day.distanceKm ?? 0;
      elevationGainM += day.elevationGainM ?? 0;
      durationSec += day.durationSec ?? 0;
      activityCount += day.activityCount ?? 0;
    });

    return {
      month: label,
      distanceKm,
      elevationGainM,
      durationSec,
      activityCount,
    };
  })
);

const selectedDayData = computed<HeatmapDayData | null>(() => {
  if (!selectedDayKey.value) {
    return null;
  }
  return currentYearData.value[selectedDayKey.value] ?? null;
});

const selectedDayLabel = computed(() => (selectedDayKey.value ? dayLabel(selectedDayKey.value) : ""));

function onHeatmapPointClick(this: any): void {
  const dayKey =
    this?.point?.custom?.dayKey ??
    dayKeyFromIndexes(Number(this?.point?.x ?? 0), Number(this?.point?.y ?? 0));
  selectedDayKey.value = dayKey;
}

function openActivityDetails(activityId: number): void {
  void router.push({ name: "activity", params: { id: String(activityId) } });
}

const chartOptions = computed((): any => ({
  chart: {
    type: "heatmap",
    height: 560,
    marginTop: 56,
    marginBottom: 56,
  },
  title: {
    text: `Activity heatmap${displayYear.value ? ` - ${displayYear.value}` : ""}`,
  },
  xAxis: {
    categories: MONTH_NAMES,
    labels: { style: { fontSize: "11px" } },
  },
  yAxis: {
    title: { text: null },
    categories: DAY_LABELS,
    reversed: false,
  },
  colorAxis: {
    min: 0,
    max: colorAxisMax.value,
    stops: [
      [0, "#eff3f8"],
      [0.15, "#c8e6c9"],
      [0.4, "#7bc47f"],
      [0.7, "#3ca55c"],
      [1, "#1f6f43"],
    ],
  },
  legend: {
    align: "right",
    layout: "vertical",
    verticalAlign: "middle",
    symbolHeight: 180,
    title: { text: selectedMetricMeta.value.legendUnit },
  },
  plotOptions: {
    series: {
      cursor: "pointer",
      point: {
        events: {
          click: onHeatmapPointClick,
        },
      },
    },
  },
  tooltip: {
    outside: true,
    useHTML: true,
    style: {
      pointerEvents: "none",
      zIndex: "9999",
    },
    formatter: function (this: any): string {
      const dayKey =
        this?.point?.custom?.dayKey ?? dayKeyFromIndexes(Number(this.point.x), Number(this.point.y));
      const dayData = currentYearData.value[dayKey];
      const metric = selectedMetric.value;

      if (!dayData || dayData.activityCount === 0) {
        return `<b>${dayLabel(dayKey)}</b><br/>No activity`;
      }

      const activitiesPreview = dayData.activities
        .slice(0, 3)
        .map((activity) => `• ${escapeHtml(activity.name)}`)
        .join("<br/>");
      const moreCount = Math.max(0, dayData.activities.length - 3);

      return [
        `<b>${dayLabel(dayKey)}</b>`,
        `${selectedMetricMeta.value.label}: <b>${formatMetric(metricValue(dayData, metric), metric)}</b>`,
        metric !== "distanceKm" ? `Distance: ${formatDistance(dayData.distanceKm)}` : "",
        metric !== "elevationGainM" ? `Elevation: ${formatElevation(dayData.elevationGainM)}` : "",
        metric !== "durationSec" ? `Duration: ${formatDuration(dayData.durationSec)}` : "",
        dayData.activities.length > 0
          ? `Types: ${resolveDayActivityTypes(dayData).map(escapeHtml).join(" · ")}`
          : "",
        `Activities: ${dayData.activityCount}`,
        activitiesPreview,
        moreCount > 0 ? `+${moreCount} more` : "",
      ]
        .filter(Boolean)
        .join("<br/>");
    },
  },
  series: [
    {
      type: "heatmap",
      name: selectedMetricMeta.value.label,
      borderWidth: 1,
      borderColor: "#ffffff",
      data: heatmapPoints.value,
      dataLabels: {
        enabled: true,
        useHTML: true,
        allowOverlap: true,
        crop: false,
        overflow: "none",
        formatter: function (this: any): string {
          const iconUrls: string[] = this?.point?.custom?.iconUrls ?? [];
          if (iconUrls.length === 0) {
            return "";
          }
          const icons = iconUrls
            .map(
              (iconUrl) =>
                `<img src="${iconUrl}" alt="" style="width:10px;height:10px;object-fit:contain;display:block;pointer-events:none;" />`
            )
            .join("");
          return `<span style="display:inline-flex;align-items:center;justify-content:center;gap:1px;padding:1px 2px;border-radius:999px;background:rgba(255,255,255,0.82);box-shadow:0 1px 2px rgba(26,32,44,0.2);max-width:100%;pointer-events:none;">
            ${icons}
          </span>`;
        },
        style: {
          textOutline: "none",
        },
        zIndex: 1,
      },
    },
  ],
  credits: { enabled: false },
}));
</script>

<template>
  <div class="heatmap-wrapper">
    <div v-if="!hasData" class="chart-empty">
      No heatmap data available for the selected filters.
    </div>

    <template v-else>
      <div class="heatmap-toolbar">
        <div class="toolbar-group">
          <span class="toolbar-label">Metric</span>
          <div class="metric-toggle">
            <button
              v-for="option in METRIC_OPTIONS"
              :key="option.key"
              type="button"
              class="metric-toggle__btn"
              :class="{ 'metric-toggle__btn--active': selectedMetric === option.key }"
              @click="selectedMetric = option.key"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <div class="toolbar-group toolbar-group--compact">
          <label for="color-scale-mode" class="toolbar-label">Color scale</label>
          <select id="color-scale-mode" v-model="colorScaleMode" class="form-select form-select-sm scale-select">
            <option value="auto">Auto</option>
            <option value="fixed">Fixed</option>
          </select>
        </div>
      </div>

      <div v-if="isAllYearsSelected" class="heatmap-note">
        All years selected: showing latest available year ({{ displayYear }}).
      </div>
      <div v-else-if="isFallbackYear" class="heatmap-note">
        No heatmap data for {{ selectedYear }}. Showing {{ displayYear }}.
      </div>

      <div class="monthly-summary">
        <article v-for="summary in monthlySummary" :key="summary.month" class="month-card">
          <header class="month-card__title">{{ summary.month }}</header>
          <div class="month-card__value">{{ formatDistance(summary.distanceKm) }}</div>
          <div class="month-card__meta">D+ {{ formatElevation(summary.elevationGainM) }}</div>
          <div class="month-card__meta">{{ formatDuration(summary.durationSec) }}</div>
          <div class="month-card__meta">{{ summary.activityCount }} activities</div>
        </article>
      </div>

      <div class="heatmap-chart">
        <Chart :options="chartOptions" />
      </div>

      <div class="day-inspector">
        <div class="day-inspector__title">Day details</div>
        <template v-if="selectedDayData && selectedDayKey">
          <div class="day-inspector__subtitle">
            {{ selectedDayLabel }} · {{ selectedDayData.activityCount }} activities
          </div>
          <div class="day-inspector__totals">
            <span>{{ formatDistance(selectedDayData.distanceKm) }}</span>
            <span>{{ formatElevation(selectedDayData.elevationGainM) }}</span>
            <span>{{ formatDuration(selectedDayData.durationSec) }}</span>
          </div>
          <div v-if="selectedDayData.activityCount === 0" class="day-inspector__empty">
            No activity on this day.
          </div>
          <ul v-else class="day-inspector__list">
            <li v-for="activity in selectedDayData.activities" :key="activity.id" class="day-inspector__item">
              <button type="button" class="day-inspector__link" @click="openActivityDetails(activity.id)">
                {{ activity.name }}
              </button>
              <span class="day-inspector__item-meta">
                {{ formatDistance(activity.distanceKm) }} · D+ {{ formatElevation(activity.elevationGainM) }} ·
                {{ formatDuration(activity.durationSec) }}
              </span>
            </li>
          </ul>
        </template>
        <div v-else class="day-inspector__empty">
          Click a heatmap day to inspect activities.
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.heatmap-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}

.heatmap-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.toolbar-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.toolbar-group--compact {
  min-width: 150px;
}

.toolbar-label {
  font-size: 0.78rem;
  color: #5f6a78;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

.metric-toggle {
  display: inline-flex;
  border: 1px solid #d8dee7;
  border-radius: 10px;
  overflow: hidden;
  background: #ffffff;
}

.metric-toggle__btn {
  border: 0;
  background: transparent;
  padding: 0.45rem 0.7rem;
  font-weight: 600;
  color: #465266;
  cursor: pointer;
}

.metric-toggle__btn + .metric-toggle__btn {
  border-left: 1px solid #d8dee7;
}

.metric-toggle__btn--active {
  background: #ffede5;
  color: #cc4f16;
}

.scale-select {
  min-width: 140px;
}

.heatmap-note {
  color: #4c617b;
  font-size: 0.88rem;
  background: #f3f8ff;
  border: 1px solid #d9e6f5;
  border-radius: 8px;
  padding: 0.45rem 0.65rem;
}

.monthly-summary {
  display: grid;
  grid-template-columns: repeat(6, minmax(135px, 1fr));
  gap: 0.55rem;
}

.month-card {
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  background: #ffffff;
  padding: 0.45rem 0.55rem;
}

.month-card__title {
  font-size: 0.78rem;
  color: #6e7a8d;
  font-weight: 700;
  text-transform: uppercase;
}

.month-card__value {
  font-weight: 700;
  font-size: 0.98rem;
  color: #28313f;
}

.month-card__meta {
  font-size: 0.8rem;
  color: #5f6a78;
}

.heatmap-chart {
  width: 100%;
}

.day-inspector {
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  padding: 0.75rem;
  background: #fbfcfe;
}

.day-inspector__title {
  font-weight: 700;
  color: #2f3b4b;
}

.day-inspector__subtitle {
  color: #5f6a78;
  margin-top: 0.15rem;
}

.day-inspector__totals {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 0.35rem;
  font-weight: 600;
  color: #2f3b4b;
}

.day-inspector__list {
  list-style: none;
  margin: 0.55rem 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.day-inspector__item {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: baseline;
}

.day-inspector__link {
  border: 0;
  background: transparent;
  color: #e25824;
  text-decoration: underline;
  cursor: pointer;
  font-weight: 600;
  padding: 0;
}

.day-inspector__item-meta {
  color: #566275;
  font-size: 0.9rem;
}

.day-inspector__empty {
  color: #5f6a78;
  margin-top: 0.45rem;
}

@media (max-width: 1200px) {
  .monthly-summary {
    grid-template-columns: repeat(4, minmax(135px, 1fr));
  }
}

@media (max-width: 840px) {
  .monthly-summary {
    grid-template-columns: repeat(2, minmax(135px, 1fr));
  }
}
</style>

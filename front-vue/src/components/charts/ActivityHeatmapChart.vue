<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { Chart } from "highcharts-vue";
import TooltipHint from "@/components/TooltipHint.vue";
import { formatTime } from "@/utils/formatters";
import { getMetricTooltip } from "@/utils/metric-tooltips";
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
const selectedComparisonYear = ref<string>("");

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

const comparisonYearOptions = computed(() =>
  availableYears.value.filter((year) => year !== displayYear.value)
);

const comparisonYearData = computed<Record<string, HeatmapDayData>>(() => {
  if (!selectedComparisonYear.value) {
    return {};
  }
  return props.activityHeatmap[selectedComparisonYear.value] ?? {};
});

watch(
  [displayYear, comparisonYearOptions],
  () => {
    if (comparisonYearOptions.value.length === 0) {
      selectedComparisonYear.value = "";
      return;
    }
    if (!comparisonYearOptions.value.includes(selectedComparisonYear.value)) {
      selectedComparisonYear.value = comparisonYearOptions.value[0] ?? "";
    }
  },
  { immediate: true }
);

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

function formatDelta(value: number, metric: HeatmapMetricKey): string {
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${formatMetric(Math.abs(value), metric)}`;
}

function formatDeltaPercent(value: number | null): string {
  if (value === null || Number.isNaN(value)) {
    return "n/a";
  }
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${Math.abs(value).toFixed(1)}%`;
}

function deltaClass(value: number): string {
  if (value > 0) {
    return "comparison-table__delta--up";
  }
  if (value < 0) {
    return "comparison-table__delta--down";
  }
  return "comparison-table__delta--flat";
}

function buildMonthlyMetricTotals(
  yearData: Record<string, HeatmapDayData>,
  metric: HeatmapMetricKey
): number[] {
  const totals = Array.from({ length: 12 }, () => 0);
  Object.entries(yearData).forEach(([dayKey, day]) => {
    const [monthStr] = dayKey.split("-");
    const monthIndex = parseInt(monthStr, 10) - 1;
    if (monthIndex < 0 || monthIndex > 11 || Number.isNaN(monthIndex)) {
      return;
    }
    totals[monthIndex] += metricValue(day, metric);
  });
  return totals;
}

function isLeapYear(year: number): boolean {
  return (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
}

function dayOfYear(date: Date): number {
  const start = new Date(date.getFullYear(), 0, 0);
  const diff = date.getTime() - start.getTime();
  return Math.floor(diff / 86_400_000);
}

function startOfIsoWeek(date: Date): Date {
  const weekStart = new Date(date);
  const dayIndex = (weekStart.getDay() + 6) % 7;
  weekStart.setDate(weekStart.getDate() - dayIndex);
  weekStart.setHours(0, 0, 0, 0);
  return weekStart;
}

function formatShortDate(date: Date): string {
  return new Intl.DateTimeFormat(navigator.language, {
    day: "2-digit",
    month: "short",
  }).format(date);
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

type ComparisonRow = {
  month: string;
  currentValue: number;
  previousValue: number;
  delta: number;
  deltaPercent: number | null;
};

const comparisonRows = computed<ComparisonRow[]>(() => {
  if (!selectedComparisonYear.value) {
    return [];
  }
  const currentTotals = buildMonthlyMetricTotals(currentYearData.value, selectedMetric.value);
  const previousTotals = buildMonthlyMetricTotals(comparisonYearData.value, selectedMetric.value);
  return MONTH_NAMES.map((month, index) => {
    const currentValue = currentTotals[index] ?? 0;
    const previousValue = previousTotals[index] ?? 0;
    const delta = currentValue - previousValue;
    const deltaPercent = previousValue > 0 ? (delta / previousValue) * 100 : null;
    return {
      month,
      currentValue,
      previousValue,
      delta,
      deltaPercent,
    };
  });
});

const comparisonSummary = computed(() => {
  if (comparisonRows.value.length === 0) {
    return null;
  }
  const currentTotal = comparisonRows.value.reduce((sum, row) => sum + row.currentValue, 0);
  const previousTotal = comparisonRows.value.reduce((sum, row) => sum + row.previousValue, 0);
  const deltaTotal = currentTotal - previousTotal;
  const deltaPercentTotal = previousTotal > 0 ? (deltaTotal / previousTotal) * 100 : null;

  const bestGain = [...comparisonRows.value]
    .filter((row) => row.delta > 0)
    .sort((a, b) => b.delta - a.delta)[0];
  const biggestDrop = [...comparisonRows.value]
    .filter((row) => row.delta < 0)
    .sort((a, b) => a.delta - b.delta)[0];

  return {
    currentTotal,
    previousTotal,
    deltaTotal,
    deltaPercentTotal,
    bestGain,
    biggestDrop,
  };
});

type DayInsightEntry = {
  dayKey: string;
  date: Date;
  metricValue: number;
  activityCount: number;
};

const currentYearDayEntries = computed<DayInsightEntry[]>(() => {
  const year = Number(displayYear.value);
  if (!Number.isFinite(year) || year <= 0) {
    return [];
  }
  return Object.entries(currentYearData.value)
    .map(([dayKey, dayData]) => {
      const [monthStr, dayStr] = dayKey.split("-");
      const month = Number(monthStr);
      const day = Number(dayStr);
      const date = new Date(year, month - 1, day);
      return {
        dayKey,
        date,
        metricValue: metricValue(dayData, selectedMetric.value),
        activityCount: dayData.activityCount ?? 0,
      };
    })
    .filter((entry) => entry.activityCount > 0)
    .filter((entry) => !Number.isNaN(entry.date.getTime()))
    .sort((a, b) => a.date.getTime() - b.date.getTime());
});

const advancedInsights = computed(() => {
  const year = Number(displayYear.value);
  if (!Number.isFinite(year) || year <= 0) {
    return null;
  }
  const entries = currentYearDayEntries.value;
  const today = new Date();
  const isCurrentYear = year === today.getFullYear();
  const totalDaysInScope = isCurrentYear ? dayOfYear(today) : isLeapYear(year) ? 366 : 365;
  const activeDays = entries.length;
  const consistency = totalDaysInScope > 0 ? (activeDays / totalDaysInScope) * 100 : 0;

  let longestStreak = 0;
  let currentStreak = 0;
  let previousDayNumber: number | null = null;
  const activeDayNumbers = entries.map((entry) => dayOfYear(entry.date));
  for (const dayNumber of activeDayNumbers) {
    if (previousDayNumber !== null && dayNumber === previousDayNumber + 1) {
      currentStreak += 1;
    } else {
      currentStreak = 1;
    }
    longestStreak = Math.max(longestStreak, currentStreak);
    previousDayNumber = dayNumber;
  }

  let longestBreak = 0;
  let previousActiveDay = 0;
  for (const dayNumber of activeDayNumbers) {
    const gap = dayNumber - previousActiveDay - 1;
    longestBreak = Math.max(longestBreak, gap);
    previousActiveDay = dayNumber;
  }
  longestBreak = Math.max(longestBreak, totalDaysInScope - previousActiveDay);

  const totalMetric = entries.reduce((sum, entry) => sum + entry.metricValue, 0);
  const avgPerActiveDay = activeDays > 0 ? totalMetric / activeDays : 0;
  const bestDay = [...entries].sort((a, b) => b.metricValue - a.metricValue)[0] ?? null;

  const weeklyMap = new Map<
    string,
    { startDate: Date; endDate: Date; metricTotal: number; activityDays: number; activities: number }
  >();
  entries.forEach((entry) => {
    const weekStart = startOfIsoWeek(entry.date);
    const key = weekStart.toISOString().slice(0, 10);
    const existing = weeklyMap.get(key);
    if (existing) {
      existing.metricTotal += entry.metricValue;
      existing.activityDays += 1;
      existing.activities += entry.activityCount;
      return;
    }
    const endDate = new Date(weekStart);
    endDate.setDate(endDate.getDate() + 6);
    weeklyMap.set(key, {
      startDate: weekStart,
      endDate,
      metricTotal: entry.metricValue,
      activityDays: 1,
      activities: entry.activityCount,
    });
  });
  const weeklyRows = [...weeklyMap.values()].sort((a, b) => a.startDate.getTime() - b.startDate.getTime());
  const bestWeek = [...weeklyRows].sort((a, b) => b.metricTotal - a.metricTotal)[0] ?? null;

  let momentum: {
    recentAverage: number;
    previousAverage: number;
    delta: number;
    deltaPercent: number | null;
  } | null = null;
  if (weeklyRows.length >= 4) {
    const recentWindowSize = Math.min(4, weeklyRows.length);
    const recentRows = weeklyRows.slice(weeklyRows.length - recentWindowSize);
    const previousRows = weeklyRows.slice(
      Math.max(0, weeklyRows.length - recentWindowSize * 2),
      weeklyRows.length - recentWindowSize
    );
    if (previousRows.length > 0) {
      const recentAverage =
        recentRows.reduce((sum, row) => sum + row.metricTotal, 0) / recentRows.length;
      const previousAverage =
        previousRows.reduce((sum, row) => sum + row.metricTotal, 0) / previousRows.length;
      const delta = recentAverage - previousAverage;
      momentum = {
        recentAverage,
        previousAverage,
        delta,
        deltaPercent: previousAverage > 0 ? (delta / previousAverage) * 100 : null,
      };
    }
  }

  const weekdayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  const weekdayBuckets = weekdayLabels.map((label) => ({
    label,
    metricTotal: 0,
    activityDays: 0,
    activities: 0,
    barPercent: 0,
  }));
  entries.forEach((entry) => {
    const weekdayIndex = (entry.date.getDay() + 6) % 7;
    const bucket = weekdayBuckets[weekdayIndex];
    bucket.metricTotal += entry.metricValue;
    bucket.activityDays += 1;
    bucket.activities += entry.activityCount;
  });
  const maxWeekdayMetric = Math.max(...weekdayBuckets.map((bucket) => bucket.metricTotal), 0);
  weekdayBuckets.forEach((bucket) => {
    bucket.barPercent = maxWeekdayMetric > 0 ? (bucket.metricTotal / maxWeekdayMetric) * 100 : 0;
  });

  const typeCounts = new Map<string, number>();
  Object.values(currentYearData.value).forEach((dayData) => {
    dayData.activities.forEach((activity) => {
      const type = (activity.type ?? "").trim() || "Other";
      typeCounts.set(type, (typeCounts.get(type) ?? 0) + 1);
    });
  });
  const typeRows = [...typeCounts.entries()]
    .map(([type, count]) => ({ type, count }))
    .sort((a, b) => b.count - a.count);
  const totalActivities = typeRows.reduce((sum, row) => sum + row.count, 0);

  const peakDays = [...entries]
    .sort((a, b) => b.metricValue - a.metricValue)
    .slice(0, 5);

  return {
    activeDays,
    totalDaysInScope,
    consistency,
    longestStreak,
    longestBreak,
    avgPerActiveDay,
    bestDay,
    bestWeek,
    momentum,
    weekdayBuckets,
    peakDays,
    typeRows,
    totalActivities,
  };
});

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

      <div v-if="selectedComparisonYear" class="comparison-panel">
        <div class="comparison-panel__header">
          <div class="comparison-panel__title">
            Comparative analysis
            <span class="comparison-panel__subtitle">
              {{ selectedMetricMeta.label }} · {{ displayYear }} vs {{ selectedComparisonYear }}
            </span>
          </div>
          <div class="comparison-panel__controls">
            <label for="comparison-year" class="toolbar-label">Compare with</label>
            <select
              id="comparison-year"
              v-model="selectedComparisonYear"
              class="form-select form-select-sm comparison-panel__select"
            >
              <option v-for="year in comparisonYearOptions" :key="year" :value="year">
                {{ year }}
              </option>
            </select>
          </div>
        </div>

        <div v-if="comparisonSummary" class="comparison-summary">
          <span class="comparison-summary__chip">
            {{ displayYear }}: {{ formatMetric(comparisonSummary.currentTotal, selectedMetric) }}
          </span>
          <span class="comparison-summary__chip">
            {{ selectedComparisonYear }}: {{ formatMetric(comparisonSummary.previousTotal, selectedMetric) }}
          </span>
          <span class="comparison-summary__chip" :class="deltaClass(comparisonSummary.deltaTotal)">
            Delta: {{ formatDelta(comparisonSummary.deltaTotal, selectedMetric) }}
            ({{ formatDeltaPercent(comparisonSummary.deltaPercentTotal) }})
          </span>
          <span v-if="comparisonSummary.bestGain" class="comparison-summary__chip comparison-summary__chip--best">
            Best gain: {{ comparisonSummary.bestGain.month }} ({{ formatDelta(comparisonSummary.bestGain.delta, selectedMetric) }})
          </span>
          <span v-if="comparisonSummary.biggestDrop" class="comparison-summary__chip comparison-summary__chip--drop">
            Biggest drop: {{ comparisonSummary.biggestDrop.month }} ({{ formatDelta(comparisonSummary.biggestDrop.delta, selectedMetric) }})
          </span>
        </div>

        <div class="comparison-table__wrap">
          <table class="comparison-table">
            <thead>
              <tr>
                <th>Month</th>
                <th>{{ displayYear }}</th>
                <th>{{ selectedComparisonYear }}</th>
                <th>Delta</th>
                <th>Delta %</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in comparisonRows" :key="row.month">
                <td>{{ row.month }}</td>
                <td>{{ formatMetric(row.currentValue, selectedMetric) }}</td>
                <td>{{ formatMetric(row.previousValue, selectedMetric) }}</td>
                <td :class="deltaClass(row.delta)">{{ formatDelta(row.delta, selectedMetric) }}</td>
                <td :class="deltaClass(row.delta)">{{ formatDeltaPercent(row.deltaPercent) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-if="advancedInsights" class="advanced-panel">
        <div class="advanced-panel__header">
          <div class="advanced-panel__title">
            Advanced insights
            <TooltipHint :text="getMetricTooltip('Heatmap Advanced Insights') ?? ''" />
            <span class="advanced-panel__subtitle">
              {{ selectedMetricMeta.label }} focused insights for {{ displayYear }}
            </span>
          </div>
        </div>

        <div class="advanced-kpis">
          <article class="advanced-kpi">
            <div class="advanced-kpi__label">
              Consistency
              <TooltipHint :text="getMetricTooltip('Heatmap Consistency') ?? ''" />
            </div>
            <div class="advanced-kpi__value">
              {{ advancedInsights.consistency.toFixed(1) }}%
            </div>
            <div class="advanced-kpi__meta">
              {{ advancedInsights.activeDays }} active days / {{ advancedInsights.totalDaysInScope }} days
            </div>
          </article>
          <article class="advanced-kpi">
            <div class="advanced-kpi__label">
              Longest streak
              <TooltipHint :text="getMetricTooltip('Heatmap Longest Streak') ?? ''" />
            </div>
            <div class="advanced-kpi__value">{{ advancedInsights.longestStreak }} days</div>
            <div class="advanced-kpi__meta">
              Longest break: {{ advancedInsights.longestBreak }} days
              <TooltipHint :text="getMetricTooltip('Heatmap Longest Break') ?? ''" />
            </div>
          </article>
          <article class="advanced-kpi">
            <div class="advanced-kpi__label">
              Average active day
              <TooltipHint :text="getMetricTooltip('Heatmap Average Active Day') ?? ''" />
            </div>
            <div class="advanced-kpi__value">
              {{ formatMetric(advancedInsights.avgPerActiveDay, selectedMetric) }}
            </div>
            <div class="advanced-kpi__meta">
              Best day:
              <template v-if="advancedInsights.bestDay">
                {{ formatShortDate(advancedInsights.bestDay.date) }}
              </template>
              <template v-else>
                n/a
              </template>
            </div>
          </article>
          <article class="advanced-kpi">
            <div class="advanced-kpi__label">
              Weekly momentum
              <TooltipHint :text="getMetricTooltip('Heatmap Weekly Momentum') ?? ''" />
            </div>
            <div
              class="advanced-kpi__value"
              :class="advancedInsights.momentum ? deltaClass(advancedInsights.momentum.delta) : 'comparison-table__delta--flat'"
            >
              <template v-if="advancedInsights.momentum">
                {{ formatDelta(advancedInsights.momentum.delta, selectedMetric) }}
              </template>
              <template v-else>
                n/a
              </template>
            </div>
            <div class="advanced-kpi__meta">
              <template v-if="advancedInsights.momentum">
                Last 4 weeks vs previous:
                {{ formatDeltaPercent(advancedInsights.momentum.deltaPercent) }}
              </template>
              <template v-else>
                Need at least 2 comparable week blocks
              </template>
            </div>
          </article>
        </div>

        <div class="advanced-grid">
          <section class="advanced-card">
            <h4 class="advanced-card__title">
              Weekday signature
              <TooltipHint :text="getMetricTooltip('Heatmap Weekday Signature') ?? ''" />
            </h4>
            <div class="weekday-bars">
              <div v-for="row in advancedInsights.weekdayBuckets" :key="row.label" class="weekday-row">
                <span class="weekday-row__label">{{ row.label }}</span>
                <div class="weekday-row__bar-wrap">
                  <span class="weekday-row__bar" :style="{ width: `${row.barPercent}%` }" />
                </div>
                <span class="weekday-row__value">{{ formatMetric(row.metricTotal, selectedMetric) }}</span>
              </div>
            </div>
          </section>

          <section class="advanced-card">
            <h4 class="advanced-card__title">
              Top days
              <TooltipHint :text="getMetricTooltip('Heatmap Top Days') ?? ''" />
            </h4>
            <div v-if="advancedInsights.peakDays.length === 0" class="advanced-card__empty">
              No activity day found.
            </div>
            <ul v-else class="peak-days-list">
              <li v-for="day in advancedInsights.peakDays" :key="day.dayKey" class="peak-days-list__item">
                <span class="peak-days-list__date">{{ formatShortDate(day.date) }}</span>
                <span class="peak-days-list__metric">{{ formatMetric(day.metricValue, selectedMetric) }}</span>
                <span class="peak-days-list__meta">{{ day.activityCount }} activities</span>
              </li>
            </ul>
          </section>

          <section class="advanced-card">
            <h4 class="advanced-card__title">
              Activity mix
              <TooltipHint :text="getMetricTooltip('Heatmap Activity Mix') ?? ''" />
            </h4>
            <div v-if="advancedInsights.typeRows.length === 0" class="advanced-card__empty">
              No activity types found.
            </div>
            <ul v-else class="type-mix-list">
              <li v-for="row in advancedInsights.typeRows.slice(0, 6)" :key="row.type" class="type-mix-list__item">
                <span class="type-mix-list__type">{{ row.type }}</span>
                <span class="type-mix-list__count">{{ row.count }}</span>
                <span class="type-mix-list__share">
                  {{
                    advancedInsights.totalActivities > 0
                      ? `${((row.count / advancedInsights.totalActivities) * 100).toFixed(1)}%`
                      : "0.0%"
                  }}
                </span>
              </li>
            </ul>
          </section>

          <section class="advanced-card">
            <h4 class="advanced-card__title">
              Best week
              <TooltipHint :text="getMetricTooltip('Heatmap Best Week') ?? ''" />
            </h4>
            <div v-if="advancedInsights.bestWeek" class="best-week">
              <div class="best-week__value">
                {{ formatMetric(advancedInsights.bestWeek.metricTotal, selectedMetric) }}
              </div>
              <div class="best-week__meta">
                {{ formatShortDate(advancedInsights.bestWeek.startDate) }} - {{ formatShortDate(advancedInsights.bestWeek.endDate) }}
              </div>
              <div class="best-week__meta">
                {{ advancedInsights.bestWeek.activityDays }} active days · {{ advancedInsights.bestWeek.activities }} activities
              </div>
            </div>
            <div v-else class="advanced-card__empty">
              No weekly data available.
            </div>
          </section>
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

.comparison-panel {
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  background: #ffffff;
  padding: 0.75rem;
}

.comparison-panel__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.8rem;
  flex-wrap: wrap;
}

.comparison-panel__title {
  display: flex;
  flex-direction: column;
  font-weight: 700;
  color: #2f3b4b;
}

.comparison-panel__subtitle {
  margin-top: 0.15rem;
  font-size: 0.85rem;
  color: #5f6a78;
  font-weight: 600;
}

.comparison-panel__controls {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.comparison-panel__select {
  min-width: 150px;
}

.comparison-summary {
  margin-top: 0.6rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.comparison-summary__chip {
  border: 1px solid #e1e5ee;
  background: #f6f8fb;
  color: #3f4a5a;
  border-radius: 999px;
  padding: 0.25rem 0.55rem;
  font-size: 0.82rem;
  font-weight: 600;
}

.comparison-summary__chip--best {
  background: #edf9f1;
  border-color: #ccead7;
  color: #1f6f43;
}

.comparison-summary__chip--drop {
  background: #fdf1f0;
  border-color: #f2d3d0;
  color: #9a3d34;
}

.comparison-table__wrap {
  margin-top: 0.6rem;
  overflow: auto;
}

.comparison-table {
  width: 100%;
  border-collapse: collapse;
  min-width: 580px;
  font-size: 0.87rem;
}

.comparison-table th,
.comparison-table td {
  border-bottom: 1px solid #eceff4;
  padding: 0.38rem 0.42rem;
  text-align: left;
  white-space: nowrap;
}

.comparison-table th {
  color: #4a5566;
  font-weight: 700;
}

.comparison-table__delta--up {
  color: #1f6f43;
  font-weight: 700;
}

.comparison-table__delta--down {
  color: #9a3d34;
  font-weight: 700;
}

.comparison-table__delta--flat {
  color: #5f6a78;
}

.advanced-panel {
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  background: linear-gradient(180deg, #ffffff 0%, #fbfcff 100%);
  box-shadow: 0 8px 24px rgba(24, 39, 75, 0.06);
  padding: 0.9rem;
}

.advanced-panel__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.8rem;
  flex-wrap: wrap;
}

.advanced-panel__title {
  display: flex;
  flex-direction: column;
  font-weight: 700;
  color: #2f3b4b;
  font-size: 1rem;
}

.advanced-panel__subtitle {
  margin-top: 0.15rem;
  font-size: 0.85rem;
  color: #5f6a78;
  font-weight: 600;
}

.advanced-kpis {
  margin-top: 0.7rem;
  display: grid;
  grid-template-columns: repeat(4, minmax(170px, 1fr));
  gap: 0.6rem;
}

.advanced-kpi {
  border: 1px solid #e7ebf2;
  border-radius: 10px;
  background: #f7f9fe;
  padding: 0.55rem 0.6rem;
}

.advanced-kpi__label {
  font-size: 0.78rem;
  text-transform: uppercase;
  letter-spacing: 0.02em;
  color: #5f6a78;
  font-weight: 700;
}

.advanced-kpi__value {
  margin-top: 0.1rem;
  font-size: 1.1rem;
  color: #263042;
  font-weight: 800;
  letter-spacing: 0.01em;
}

.advanced-kpi__meta {
  margin-top: 0.15rem;
  font-size: 0.8rem;
  color: #5f6a78;
}

.advanced-grid {
  margin-top: 0.7rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(260px, 1fr));
  gap: 0.6rem;
}

.advanced-card {
  border: 1px solid #e7ebf2;
  border-radius: 10px;
  background: #ffffff;
  padding: 0.55rem 0.6rem;
  box-shadow: 0 2px 10px rgba(24, 39, 75, 0.04);
}

.advanced-card__title {
  margin: 0;
  font-size: 0.9rem;
  color: #2f3b4b;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.advanced-card__empty {
  margin-top: 0.45rem;
  color: #5f6a78;
  font-size: 0.84rem;
}

.weekday-bars {
  margin-top: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.weekday-row {
  display: grid;
  grid-template-columns: 34px 1fr auto;
  align-items: center;
  gap: 0.45rem;
}

.weekday-row__label {
  font-size: 0.8rem;
  color: #4d596b;
  font-weight: 600;
}

.weekday-row__bar-wrap {
  height: 8px;
  border-radius: 999px;
  background: #edf1f7;
  overflow: hidden;
}

.weekday-row__bar {
  display: block;
  height: 100%;
  min-width: 2px;
  border-radius: 999px;
  background: linear-gradient(90deg, #ffb35e 0%, #fc4c02 100%);
}

.weekday-row__value {
  font-size: 0.8rem;
  color: #4d596b;
  font-weight: 600;
}

.peak-days-list,
.type-mix-list {
  margin: 0.45rem 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.peak-days-list__item,
.type-mix-list__item {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 0.45rem;
  align-items: baseline;
  font-size: 0.84rem;
}

.peak-days-list__date,
.type-mix-list__type {
  color: #2f3b4b;
  font-weight: 600;
}

.peak-days-list__metric,
.type-mix-list__count {
  color: #4d596b;
  font-weight: 700;
}

.peak-days-list__meta,
.type-mix-list__share {
  color: #6a7586;
}

.best-week {
  margin-top: 0.5rem;
}

.best-week__value {
  font-size: 1.02rem;
  font-weight: 800;
  color: #263042;
}

.best-week__meta {
  margin-top: 0.15rem;
  font-size: 0.82rem;
  color: #5f6a78;
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

  .advanced-kpis {
    grid-template-columns: repeat(2, minmax(170px, 1fr));
  }
}

@media (max-width: 840px) {
  .monthly-summary {
    grid-template-columns: repeat(2, minmax(135px, 1fr));
  }

  .advanced-kpis {
    grid-template-columns: 1fr;
  }

  .advanced-grid {
    grid-template-columns: 1fr;
  }
}
</style>

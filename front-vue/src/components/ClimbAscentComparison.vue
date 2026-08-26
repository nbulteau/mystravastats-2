<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { ClimbAscent } from "@/models/badge-check-result.model";
import {
  buildSectorComparisons,
  defaultComparedAscentIds,
  type ClimbComparisonMetric,
} from "@/utils/climb-comparison";
import { formatTime } from "@/utils/formatters";

const props = defineProps<{
  ascents: ClimbAscent[];
  bestAscentId: number | null;
  lengthKm: number;
}>();

const metric = ref<ClimbComparisonMetric>("elapsedSeconds");
const selectedIds = ref<number[]>([]);
const colors = ["#e85d2a", "#176d50", "#3768a6"];
const metricOptions: { value: ClimbComparisonMetric; label: string; suffix: string }[] = [
  { value: "elapsedSeconds", label: "Time", suffix: "" },
  { value: "speedKph", label: "Speed", suffix: " km/h" },
  { value: "vamMetersPerHour", label: "VAM", suffix: " m/h" },
  { value: "powerWatts", label: "Power", suffix: " W" },
  { value: "heartRateBpm", label: "Cardio", suffix: " bpm" },
];

const eligibleAscents = computed(() => props.ascents.filter((ascent) => (ascent.comparisonPoints?.length ?? 0) >= 2));
const selectedAscents = computed(() => selectedIds.value
  .map((id) => eligibleAscents.value.find((ascent) => ascent.activityId === id))
  .filter((ascent): ascent is ClimbAscent => Boolean(ascent)));

watch(
  () => [props.bestAscentId, ...props.ascents.map((ascent) => ascent.activityId)],
  () => {
    selectedIds.value = defaultComparedAscentIds(props.ascents, props.bestAscentId);
  },
  { immediate: true },
);

const chartSeries = computed(() => selectedAscents.value.map((ascent, index) => ({
  ascent,
  color: colors[index],
  values: (ascent.comparisonPoints ?? []).flatMap((point) => {
    const value = point[metric.value];
    return typeof value === "number" && Number.isFinite(value)
      ? [{ distanceKm: point.distanceKm, value }]
      : [];
  }),
})));
const chartValues = computed(() => chartSeries.value.flatMap((series) => series.values.map((point) => point.value)));
const chartMin = computed(() => chartValues.value.length ? Math.min(...chartValues.value) : 0);
const chartMax = computed(() => chartValues.value.length ? Math.max(...chartValues.value) : 1);
const chartDistance = computed(() => Math.max(
  props.lengthKm,
  ...chartSeries.value.flatMap((series) => series.values.map((point) => point.distanceKm)),
  1,
));
const chartPolylines = computed(() => chartSeries.value.map((series) => ({
  ...series,
  points: series.values.map((point) => {
    const x = 54 + (point.distanceKm / chartDistance.value) * 702;
    const range = Math.max(1, chartMax.value - chartMin.value);
    const y = 224 - ((point.value - chartMin.value) / range) * 178;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" "),
})));

const sectorRows = computed(() => {
  const [reference, ...candidates] = selectedAscents.value;
  if (!reference) return [];
  const candidateSectors = candidates.map((ascent) => ({
    ascent,
    sectors: buildSectorComparisons(ascent, reference, props.lengthKm),
  }));
  const referenceSectors = buildSectorComparisons(reference, reference, props.lengthKm);
  return referenceSectors.map((sector, index) => ({
    ...sector,
    comparisons: candidateSectors.map(({ ascent, sectors }) => ({ ascent, ...sectors[index] })),
  }));
});

function toggleAscent(activityId: number): void {
  if (selectedIds.value.includes(activityId)) {
    selectedIds.value = selectedIds.value.filter((id) => id !== activityId);
  } else if (selectedIds.value.length < 3) {
    selectedIds.value = [...selectedIds.value, activityId];
  }
}

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString("en-GB");
}

function formatMetric(value: number): string {
  if (metric.value === "elapsedSeconds") return formatTime(Math.round(value));
  const option = metricOptions.find((candidate) => candidate.value === metric.value);
  return `${Math.round(value).toLocaleString("en-US")}${option?.suffix ?? ""}`;
}

function formatDelta(value: number | null): string {
  if (value == null) return "—";
  const sign = value > 0 ? "+" : value < 0 ? "−" : "±";
  return `${sign}${formatTime(Math.abs(Math.round(value)))}`;
}

function warningLabel(code: string): string {
  const labels: Record<string, string> = {
    MISSING_STREAM: "GPS stream unavailable",
    INCOMPLETE_DISTANCE_STREAM: "Incomplete distance stream",
    INCOMPLETE_TIME_STREAM: "Incomplete time stream",
    DISTANCE_DIFFERENCE_OVER_10_PERCENT: "Detected distance differs from the catalogue",
    START_OFFSET_OVER_100_METERS: "Start point offset by more than 100 m",
    FINISH_OFFSET_OVER_100_METERS: "Finish point offset by more than 100 m",
    LOW_RESOLUTION_STREAM: "Low-resolution GPS stream",
  };
  return labels[code] ?? code;
}
</script>

<template>
  <section class="comparison-card" aria-labelledby="comparison-title">
    <div class="comparison-heading">
      <div>
        <p>Personal analysis</p>
        <h2 id="comparison-title">Compare my ascents</h2>
      </div>
      <span>2 to 3 ascents · same side</span>
    </div>

    <p v-if="eligibleAscents.length < 2" class="comparison-empty">
      Two ascents with complete distance and time streams are required for comparison.
    </p>
    <template v-else>
      <div class="ascent-picker" aria-label="Ascents to compare">
        <label v-for="ascent in eligibleAscents" :key="ascent.activityId">
          <input
            type="checkbox"
            :checked="selectedIds.includes(ascent.activityId)"
            :disabled="!selectedIds.includes(ascent.activityId) && selectedIds.length >= 3"
            @change="toggleAscent(ascent.activityId)"
          >
          <span>
            <strong>{{ formatDate(ascent.date) }}</strong>
            <small>{{ ascent.activityId === bestAscentId ? "Best time" : ascent.activityName }}</small>
          </span>
        </label>
      </div>

      <div v-if="selectedAscents.length >= 2" class="comparison-content">
        <div class="metric-tabs" role="group" aria-label="Comparison metric">
          <button
            v-for="option in metricOptions"
            :key="option.value"
            type="button"
            :class="{ active: metric === option.value }"
            @click="metric = option.value"
          >{{ option.label }}</button>
        </div>

        <div class="chart-wrap">
          <svg viewBox="0 0 800 260" role="img" :aria-label="`Comparison progress by ${metricOptions.find((option) => option.value === metric)?.label.toLowerCase()}`">
            <line v-for="index in 5" :key="index" x1="54" x2="756" :y1="46 + (index - 1) * 44.5" :y2="46 + (index - 1) * 44.5" />
            <polyline
              v-for="series in chartPolylines"
              :key="series.ascent.activityId"
              :points="series.points"
              :stroke="series.color"
            />
            <text x="54" y="246">0 km</text>
            <text x="756" y="246" text-anchor="end">{{ chartDistance.toLocaleString("en-US", { maximumFractionDigits: 1 }) }} km</text>
            <text x="54" y="35">{{ formatMetric(chartMax) }}</text>
            <text x="54" y="238">{{ formatMetric(chartMin) }}</text>
          </svg>
          <div class="chart-legend">
            <span v-for="(series, index) in chartSeries" :key="series.ascent.activityId">
              <i :style="{ background: colors[index] }" /> {{ formatDate(series.ascent.date) }}
            </span>
          </div>
          <p v-if="chartValues.length < 2" class="metric-missing">This metric is unavailable for the selected ascents.</p>
        </div>

        <div class="quality-grid">
          <article v-for="ascent in selectedAscents" :key="ascent.activityId">
            <strong>{{ formatDate(ascent.date) }} · {{ ascent.comparisonQuality?.precision === "high" ? "high" : "estimated" }} accuracy</strong>
            <span>
              Aligned using catalogue points and distance · start ±{{ ascent.comparisonQuality?.startOffsetMeters ?? 0 }} m ·
              finish ±{{ ascent.comparisonQuality?.finishOffsetMeters ?? 0 }} m
            </span>
            <ul v-if="ascent.comparisonQuality?.warnings.length">
              <li v-for="warning in ascent.comparisonQuality.warnings" :key="warning">{{ warningLabel(warning) }}</li>
            </ul>
          </article>
        </div>

        <div class="sector-table-wrap">
          <table>
            <thead>
              <tr>
                <th>Sector</th>
                <th>{{ formatDate(selectedAscents[0].date) }} (reference)</th>
                <th v-for="ascent in selectedAscents.slice(1)" :key="ascent.activityId">{{ formatDate(ascent.date) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="sector in sectorRows" :key="sector.startKm">
                <th>Km {{ sector.startKm.toLocaleString("en-US") }}–{{ sector.endKm.toLocaleString("en-US", { maximumFractionDigits: 1 }) }}</th>
                <td>{{ sector.sectorSeconds == null ? "—" : formatTime(Math.round(sector.sectorSeconds)) }}</td>
                <td
                  v-for="comparison in sector.comparisons"
                  :key="comparison.ascent.activityId"
                  :class="{ gain: comparison.deltaSeconds != null && comparison.deltaSeconds < 0, loss: comparison.deltaSeconds != null && comparison.deltaSeconds > 0 }"
                >{{ formatDelta(comparison.deltaSeconds) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="method-note">Gaps are interpolated by distance between selected GPS points. They show a sector trend, not an official time measurement.</p>
      </div>
      <p v-else class="comparison-empty">Select at least two ascents.</p>
    </template>
  </section>
</template>

<style scoped>
.comparison-card { margin-top: 16px; padding: 20px 22px; border: 1px solid #c8d6e5; border-radius: 17px; background: linear-gradient(180deg, #fff 0%, #f7faff 100%); box-shadow: var(--ms-shadow-soft); }
.comparison-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.comparison-heading p { margin: 0 0 6px; color: #3768a6; font-size: .7rem; font-weight: 850; letter-spacing: .09em; text-transform: uppercase; }
.comparison-heading h2 { margin: 0; font-size: 1.12rem; }
.comparison-heading > span { color: var(--ms-text-muted); font-size: .72rem; font-weight: 750; }
.ascent-picker { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 14px; }
.ascent-picker label { display: flex; align-items: center; gap: 8px; min-width: 175px; padding: 8px 10px; border: 1px solid #d7e0ea; border-radius: 10px; background: white; cursor: pointer; }
.ascent-picker span { display: flex; flex-direction: column; }.ascent-picker strong { font-size: .76rem; }.ascent-picker small { max-width: 180px; overflow: hidden; color: var(--ms-text-muted); font-size: .65rem; text-overflow: ellipsis; white-space: nowrap; }
.comparison-content { margin-top: 14px; }.metric-tabs { display: flex; flex-wrap: wrap; gap: 5px; }
.metric-tabs button { padding: 5px 10px; border: 1px solid #cad5e2; border-radius: 999px; color: #36536f; background: white; font-size: .7rem; font-weight: 750; }
.metric-tabs button.active { border-color: #3768a6; color: white; background: #3768a6; }
.chart-wrap { margin-top: 10px; padding: 10px; border: 1px solid #e0e6ed; border-radius: 12px; background: white; }
svg { display: block; width: 100%; height: auto; } svg line { stroke: #e5eaf0; stroke-width: 1; } svg polyline { fill: none; stroke-linecap: round; stroke-linejoin: round; stroke-width: 4; } svg text { fill: #667788; font-size: 11px; font-weight: 650; }
.chart-legend { display: flex; flex-wrap: wrap; justify-content: center; gap: 12px; }.chart-legend span { color: var(--ms-text-muted); font-size: .68rem; font-weight: 700; }.chart-legend i { display: inline-block; width: 14px; height: 3px; margin-right: 4px; vertical-align: middle; }
.metric-missing,.method-note,.comparison-empty { margin: 12px 0 0; color: var(--ms-text-muted); font-size: .74rem; }.comparison-empty { padding: 12px; border: 1px dashed #ced6df; border-radius: 9px; background: #f8fafc; }
.quality-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; margin-top: 10px; }.quality-grid article { display: flex; flex-direction: column; padding: 9px; border-left: 3px solid #8aa9ca; background: #f5f8fc; }.quality-grid strong { font-size: .69rem; }.quality-grid span,.quality-grid li { color: var(--ms-text-muted); font-size: .62rem; }.quality-grid ul { margin: 4px 0 0; padding-left: 16px; }
.sector-table-wrap { max-height: 360px; margin-top: 10px; overflow: auto; border: 1px solid #e0e6ed; border-radius: 10px; background: white; }.sector-table-wrap table { width: 100%; min-width: 560px; border-collapse: collapse; }.sector-table-wrap th,.sector-table-wrap td { padding: 7px 9px; border-bottom: 1px solid #edf0f4; font-size: .68rem; text-align: left; }.sector-table-wrap thead th { position: sticky; top: 0; z-index: 1; color: var(--ms-text-muted); background: #f7f9fb; text-transform: uppercase; }.sector-table-wrap .gain { color: #176d50; font-weight: 800; }.sector-table-wrap .loss { color: #a23a24; font-weight: 800; }
@media (max-width: 760px) { .comparison-heading { flex-direction: column; }.quality-grid { grid-template-columns: 1fr; }.comparison-card { padding: 15px 12px; } }
</style>

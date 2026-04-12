<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import TooltipHint from "@/components/TooltipHint.vue";
import type {
  HeartRateZoneAnalysis,
  HeartRateZoneSettings,
} from "@/models/heart-rate-zone.model";
import { formatTime } from "@/utils/formatters";
import { getMetricTooltip } from "@/utils/metric-tooltips";

const props = defineProps<{
  analysis: HeartRateZoneAnalysis;
  settings: HeartRateZoneSettings;
}>();

const emit = defineEmits<{
  (e: "save-settings", payload: HeartRateZoneSettings): void;
}>();

const draftSettings = reactive<HeartRateZoneSettings>({
  maxHr: null,
  thresholdHr: null,
  reserveHr: null,
});

watch(
  () => props.settings,
  (settings) => {
    draftSettings.maxHr = settings.maxHr ?? null;
    draftSettings.thresholdHr = settings.thresholdHr ?? null;
    draftSettings.reserveHr = settings.reserveHr ?? null;
  },
  { immediate: true, deep: true },
);

const recentActivities = computed(() =>
  [...props.analysis.activities].reverse().slice(0, 12),
);

const methodLabel = computed(() => {
  const method = props.analysis.resolvedSettings?.method;
  if (method === "THRESHOLD") return "Threshold HR";
  if (method === "RESERVE") return "HR Reserve";
  if (method === "MAX") return "Max HR";
  return "Unavailable";
});

const sourceLabel = computed(() => {
  const source = props.analysis.resolvedSettings?.source;
  if (source === "ATHLETE_SETTINGS") return "Athlete settings";
  if (source === "DERIVED_FROM_DATA") return "Derived from activities";
  return "Unavailable";
});

function normalizePositiveInt(value: number | null | undefined): number | null {
  if (value === null || value === undefined || Number.isNaN(value)) return null;
  const normalized = Math.trunc(value);
  return normalized > 0 ? normalized : null;
}

const derivedMaxHr = computed(() => {
  const resolved = props.analysis.resolvedSettings;
  const resolvedMaxHr = normalizePositiveInt(resolved?.maxHr);
  if (!resolved || resolvedMaxHr === null) return null;

  if (resolved.source === "DERIVED_FROM_DATA") {
    return resolvedMaxHr;
  }

  // Backward-compatible inference for older backend builds:
  // in THRESHOLD mode, max HR can still be derived when no explicit max HR is saved.
  const savedMaxHr = normalizePositiveInt(props.settings.maxHr);
  const thresholdHr = normalizePositiveInt(props.settings.thresholdHr);
  const inferredDerivedInThresholdMode =
    resolved.method === "THRESHOLD" &&
    savedMaxHr === null &&
    thresholdHr !== null &&
    resolvedMaxHr !== thresholdHr;

  return inferredDerivedInThresholdMode ? resolvedMaxHr : null;
});

const maxHrInputValue = computed<number | "">({
  get() {
    if (
      draftSettings.maxHr !== null &&
      draftSettings.maxHr !== undefined &&
      draftSettings.maxHr > 0
    ) {
      return draftSettings.maxHr;
    }
    return derivedMaxHr.value ?? "";
  },
  set(value) {
    if (value === "" || value === null || value === undefined) {
      draftSettings.maxHr = null;
      return;
    }

    const parsed = Number(value);
    if (Number.isNaN(parsed) || parsed <= 0) {
      draftSettings.maxHr = null;
      return;
    }

    draftSettings.maxHr = Math.trunc(parsed);
  },
});

const maxHrIsDerivedInTextbox = computed(
  () =>
    (draftSettings.maxHr === null || draftSettings.maxHr === undefined) &&
    derivedMaxHr.value !== null,
);

const resolvedMaxHrLabel = computed(() => {
  const maxHr = props.analysis.resolvedSettings?.maxHr;
  if (!maxHr || maxHr <= 0) return "-";
  return `${maxHr} bpm`;
});

const usesDerivedMaxHr = computed(
  () => props.analysis.resolvedSettings?.source === "DERIVED_FROM_DATA",
);

function saveSettings() {
  emit("save-settings", {
    maxHr: normalizePositiveInt(draftSettings.maxHr),
    thresholdHr: normalizePositiveInt(draftSettings.thresholdHr),
    reserveHr: normalizePositiveInt(draftSettings.reserveHr),
  });
}

function formatRatio(value: number | null | undefined): string {
  if (value === null || value === undefined) return "-";
  return `${value.toFixed(2)} : 1`;
}

function zoneTooltip(zone: string): string {
  const tooltips: Record<string, string> = {
    Z1: "Recovery zone. Very easy effort, mostly aerobic and low fatigue.",
    Z2: "Endurance zone. Easy to moderate effort used to build aerobic base.",
    Z3: "Tempo zone. Sustained moderate-hard effort.",
    Z4: "Threshold zone. Hard effort near lactate threshold.",
    Z5: "VO2 max zone. Very hard effort, typically short intervals.",
  };
  return tooltips[zone] ?? "Heart rate zone distribution.";
}
</script>

<template>
  <section class="hr-zone-card">
    <header class="hr-zone-header">
      <h4 class="mb-0">
        Heart Rate Zone Analysis
        <TooltipHint :text="'Zone-based training breakdown using heart rate streams and your settings.'" />
      </h4>
      <small class="text-muted hr-zone-method">
        Method: {{ methodLabel }}
        <TooltipHint :text="getMetricTooltip('HR Zone Method') ?? ''" />
        <span class="ms-2">
          Source: {{ sourceLabel }}
          <TooltipHint :text="getMetricTooltip('HR Zone Source') ?? ''" />
        </span>
      </small>
    </header>

    <div class="hr-zone-settings row g-2 align-items-end">
      <div class="col-sm-4">
        <label class="form-label">
          Max HR
          <TooltipHint :text="getMetricTooltip('Max HR') ?? ''" />
        </label>
        <input v-model="maxHrInputValue" type="number" min="1" class="form-control form-control-sm">
        <small v-if="maxHrIsDerivedInTextbox" class="text-muted">
          Derived from activities (effective value currently used).
        </small>
      </div>
      <div class="col-sm-4">
        <label class="form-label">
          Threshold HR
          <TooltipHint :text="getMetricTooltip('Threshold HR') ?? ''" />
        </label>
        <input v-model.number="draftSettings.thresholdHr" type="number" min="1" class="form-control form-control-sm">
      </div>
      <div class="col-sm-4">
        <label class="form-label">
          Reserve HR
          <TooltipHint :text="getMetricTooltip('Reserve HR') ?? ''" />
        </label>
        <input v-model.number="draftSettings.reserveHr" type="number" min="1" class="form-control form-control-sm">
      </div>
      <div class="col-12 d-flex justify-content-end">
        <button class="btn btn-primary btn-sm" type="button" @click="saveSettings">
          Save Zones
        </button>
      </div>
      <div class="col-12">
        <small class="text-muted">
          Settings are saved per athlete in local cache.
          <TooltipHint :text="getMetricTooltip('Heart Rate Zone Storage') ?? ''" />
        </small>
      </div>
    </div>

    <div class="hr-zone-summary row g-2">
      <div class="col-md-3">
        <div class="metric-tile">
          <div class="metric-label">
            Tracked HR Time
            <TooltipHint :text="getMetricTooltip('Tracked HR Time') ?? ''" />
          </div>
          <div class="metric-value">{{ formatTime(analysis.totalTrackedSeconds) }}</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="metric-tile">
          <div class="metric-label">
            Easy / Hard Ratio
            <TooltipHint :text="getMetricTooltip('Easy / Hard Ratio') ?? ''" />
          </div>
          <div class="metric-value">{{ formatRatio(analysis.easyHardRatio) }}</div>
        </div>
      </div>
      <div class="col-md-3">
        <div class="metric-tile">
          <div class="metric-label">
            Resolved Max HR
            <TooltipHint :text="getMetricTooltip('Resolved Max HR') ?? ''" />
          </div>
          <div class="metric-value" :class="{ 'metric-value--derived': usesDerivedMaxHr }">
            {{ resolvedMaxHrLabel }}
          </div>
          <small v-if="analysis.resolvedSettings" class="metric-subvalue">
            {{ sourceLabel }}
          </small>
        </div>
      </div>
      <div class="col-md-3">
        <div class="metric-tile">
          <div class="metric-label">
            HR Data Availability
            <TooltipHint :text="getMetricTooltip('HR Data Availability') ?? ''" />
          </div>
          <div class="metric-value">{{ analysis.hasHeartRateData ? "Available" : "Unavailable" }}</div>
        </div>
      </div>
    </div>

    <div v-if="!analysis.hasHeartRateData" class="hr-zone-empty">
      No heart rate streams available for the current filters.
    </div>

    <template v-else>
      <div class="zone-bars">
        <div
          v-for="zone in analysis.zones"
          :key="zone.zone"
          class="zone-row"
        >
          <div class="zone-label">
            {{ zone.zone }} - {{ zone.label }}
            <TooltipHint :text="zoneTooltip(zone.zone)" />
          </div>
          <div class="zone-progress">
            <div class="progress">
              <div
                class="progress-bar"
                role="progressbar"
                :style="{ width: `${zone.percentage}%` }"
                :aria-valuenow="zone.percentage"
                aria-valuemin="0"
                aria-valuemax="100"
              />
            </div>
          </div>
          <div class="zone-value">{{ formatTime(zone.seconds) }} ({{ zone.percentage.toFixed(1) }}%)</div>
        </div>
      </div>

      <div class="table-responsive mt-3">
        <table class="table table-sm table-striped">
          <thead>
            <tr>
              <th>Month</th>
              <th>
                Tracked
                <TooltipHint :text="getMetricTooltip('Tracked HR Time') ?? ''" />
              </th>
              <th>
                Easy/Hard
                <TooltipHint :text="getMetricTooltip('Easy / Hard Ratio') ?? ''" />
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="period in analysis.byMonth" :key="period.period">
              <td>{{ period.period }}</td>
              <td>{{ formatTime(period.totalTrackedSeconds) }}</td>
              <td>{{ formatRatio(period.easyHardRatio) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="table-responsive">
        <table class="table table-sm table-striped">
          <thead>
            <tr>
              <th>Recent Activity</th>
              <th>Date</th>
              <th>
                Tracked
                <TooltipHint :text="getMetricTooltip('Tracked HR Time') ?? ''" />
              </th>
              <th>
                Easy/Hard
                <TooltipHint :text="getMetricTooltip('Easy / Hard Ratio') ?? ''" />
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="summary in recentActivities" :key="summary.activity.id">
              <td>{{ summary.activity.name }}</td>
              <td>{{ summary.activityDate.split("T")[0] }}</td>
              <td>{{ formatTime(summary.totalTrackedSeconds) }}</td>
              <td>{{ formatRatio(summary.easyHardRatio) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>

<style scoped>
.hr-zone-card {
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 12px;
}

.hr-zone-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 12px;
}

.hr-zone-method {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.metric-tile {
  border: 1px solid #eceef4;
  border-radius: 10px;
  padding: 8px;
  background: #fafbfe;
}

.metric-label {
  color: var(--ms-text-muted);
  font-size: 0.85rem;
}

.metric-value {
  font-weight: 800;
  color: var(--ms-text);
}

.metric-value--derived {
  color: var(--ms-primary);
}

.metric-subvalue {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
}

.zone-bars {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.zone-row {
  display: grid;
  grid-template-columns: 170px 1fr 190px;
  align-items: center;
  gap: 10px;
}

.zone-label {
  font-size: 0.9rem;
  color: #2c2f36;
}

.zone-value {
  text-align: right;
  color: #2c2f36;
  font-size: 0.9rem;
}

.progress {
  height: 10px;
  border-radius: 999px;
  background-color: #f0f2f6;
}

.progress-bar {
  background: linear-gradient(135deg, #fc4c02, #ff7a31);
}

.hr-zone-empty {
  border: 1px dashed #e2e5ec;
  border-radius: 10px;
  color: var(--ms-text-muted);
  font-size: 0.95rem;
  padding: 16px;
  background: #fafbfe;
}

@media (max-width: 992px) {
  .zone-row {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .zone-value {
    text-align: left;
  }
}
</style>

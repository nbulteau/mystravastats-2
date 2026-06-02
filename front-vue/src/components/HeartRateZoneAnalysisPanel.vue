<script setup lang="ts">
import { computed } from "vue";
import TooltipHint from "@/components/TooltipHint.vue";
import type { HeartRateZoneAnalysis } from "@/models/heart-rate-zone.model";
import { formatTime } from "@/utils/formatters";
import { getMetricTooltip } from "@/utils/metric-tooltips";

const props = defineProps<{
  analysis: HeartRateZoneAnalysis;
}>();

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

const resolvedMaxHrLabel = computed(() => {
  const maxHr = props.analysis.resolvedSettings?.maxHr;
  if (!maxHr || maxHr <= 0) return "-";
  return `${maxHr} bpm`;
});

const usesDerivedMaxHr = computed(
  () => props.analysis.resolvedSettings?.source === "DERIVED_FROM_DATA",
);

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
      <div class="hr-zone-header__meta">
        <small class="text-muted hr-zone-method">
          Method: {{ methodLabel }}
          <TooltipHint :text="getMetricTooltip('HR Zone Method') ?? ''" />
          <span class="ms-2">
            Source: {{ sourceLabel }}
            <TooltipHint :text="getMetricTooltip('HR Zone Source') ?? ''" />
          </span>
        </small>
        <RouterLink
          class="btn btn-outline-secondary btn-sm hr-zone-edit-link"
          to="/settings"
        >
          <i class="fa-solid fa-sliders" aria-hidden="true" />
          Edit HR settings
        </RouterLink>
      </div>
    </header>

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
  gap: 12px;
  margin-bottom: 12px;
}

.hr-zone-header__meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

.hr-zone-method {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.hr-zone-edit-link {
  align-items: center;
  display: inline-flex;
  gap: 6px;
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
  .hr-zone-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .hr-zone-header__meta {
    justify-content: flex-start;
  }

  .zone-row {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .zone-value {
    text-align: left;
  }
}
</style>

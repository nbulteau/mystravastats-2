<script setup lang="ts">
import { computed } from "vue";
import VGrid, { VGridVueTemplate } from "@revolist/vue3-datagrid";
import ActivityCellRenderer from "@/components/cell-renderers/ActivityCellRenderer.vue";
import { useContextStore } from "@/stores/context.js";

const contextStore = useContextStore();
contextStore.updateCurrentView("segments");

const progression = computed(() => contextStore.segmentClimbProgression);
const targets = computed(() => progression.value.targets);
const attempts = computed(() => progression.value.attempts);
const selectedTarget = computed(() =>
  targets.value.find((target) => target.targetId === progression.value.selectedTargetId)
);

const selectedMetric = computed({
  get: () => contextStore.segmentProgressionMetric,
  set: (metric: "TIME" | "SPEED") => {
    void contextStore.updateSegmentProgressionMetric(metric);
  },
});

const selectedTargetTypeFilter = computed({
  get: () => contextStore.segmentProgressionTargetType,
  set: (targetType: "ALL" | "SEGMENT" | "CLIMB") => {
    void contextStore.updateSegmentProgressionTargetType(targetType);
  },
});

const rows = computed(() =>
  attempts.value.map((attempt) => ({
    ...attempt,
    activityDate: attempt.activityDate.split("T")[0],
    elapsedTime: formatSeconds(attempt.elapsedTimeSeconds),
    speedKphDisplay: `${attempt.speedKph.toFixed(1)} km/h`,
    distanceDisplay: `${(attempt.distance / 1000).toFixed(2)} km`,
    averageGradeDisplay: `${attempt.averageGrade.toFixed(1)} %`,
    elevationGainDisplay: `${attempt.elevationGain.toFixed(0)} m`,
    closeToPrDisplay: attempt.closeToPr ? "Yes" : "-",
    setsNewPrDisplay: attempt.setsNewPr ? "Yes" : "-",
    prRankDisplay: attempt.prRank ?? "-",
    weatherDisplay: attempt.weatherSummary ?? "-",
  }))
);

const columns = [
  { prop: "activityDate", name: "Date", size: 120 },
  { prop: "elapsedTime", name: "Time", size: 120 },
  { prop: "speedKphDisplay", name: "Speed", size: 120 },
  { prop: "distanceDisplay", name: "Distance", size: 120 },
  { prop: "averageGradeDisplay", name: "Grade", size: 110 },
  { prop: "elevationGainDisplay", name: "Elev+", size: 110 },
  { prop: "deltaToPr", name: "Delta PR", size: 120 },
  { prop: "closeToPrDisplay", name: "Close PR", size: 110 },
  { prop: "setsNewPrDisplay", name: "New PR", size: 110 },
  { prop: "prRankDisplay", name: "PR Rank", size: 100 },
  {
    prop: "activity",
    name: "Activity",
    size: 320,
    cellTemplate: VGridVueTemplate(ActivityCellRenderer),
  },
  { prop: "weatherDisplay", name: "Weather", size: 130 },
];

const selectTarget = (targetId: number) => {
  void contextStore.updateSegmentProgressionTarget(targetId);
};

function formatSeconds(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, "0")}:${seconds
      .toString()
      .padStart(2, "0")}`;
  }
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}
</script>

<template>
  <div class="segment-progression-page">
    <div class="toolbar">
      <div class="toolbar-item">
        <label for="segmentMetric" class="form-label mb-0">Metric</label>
        <select id="segmentMetric" v-model="selectedMetric" class="form-select form-select-sm">
          <option value="TIME">Time</option>
          <option value="SPEED">Speed</option>
        </select>
      </div>
      <div class="toolbar-item">
        <label for="segmentTargetType" class="form-label mb-0">Type</label>
        <select id="segmentTargetType" v-model="selectedTargetTypeFilter" class="form-select form-select-sm">
          <option value="ALL">All</option>
          <option value="CLIMB">Climbs</option>
          <option value="SEGMENT">Segments</option>
        </select>
      </div>
      <span class="toolbar-count">{{ targets.length }} favorites</span>
    </div>

    <div v-if="targets.length === 0" class="empty-state">
      No favorite climbs or segments were found for the selected sport/year.
      Open a few detailed activities to enrich the local segment cache, then refresh this page.
    </div>

    <div v-else class="content-grid">
      <aside class="targets-list">
        <button
          v-for="target in targets"
          :key="target.targetId"
          type="button"
          class="target-button"
          :class="{ selected: progression.selectedTargetId === target.targetId }"
          @click="selectTarget(target.targetId)"
        >
          <div class="target-name">{{ target.targetName }}</div>
          <div class="target-meta">
            {{ target.targetType }} · {{ target.attemptsCount }} attempts · best {{ target.bestValue }}
          </div>
        </button>
      </aside>

      <section class="details">
        <div v-if="selectedTarget" class="summary-card">
          <h5 class="summary-title">{{ selectedTarget.targetName }}</h5>
          <div class="summary-metrics">
            <span>{{ selectedTarget.targetType }}</span>
            <span>{{ (selectedTarget.distance / 1000).toFixed(2) }} km</span>
            <span>{{ selectedTarget.averageGrade.toFixed(1) }} %</span>
            <span>Best: {{ selectedTarget.bestValue }}</span>
            <span>Latest: {{ selectedTarget.latestValue }}</span>
            <span>Consistency: {{ selectedTarget.consistency }}</span>
            <span>Pacing: {{ selectedTarget.averagePacing }}</span>
            <span>Close PR: {{ selectedTarget.closeToPrCount }}</span>
            <span>Trend: {{ selectedTarget.recentTrend }}</span>
          </div>
          <small class="weather-note">
            Weather context is planned. Current dataset does not include weather enrichment yet.
          </small>
        </div>

        <VGrid
          name="segmentClimbProgressionGrid"
          theme="material"
          :columns="columns"
          :source="rows"
          :readonly="true"
          style="height: 56vh;"
        />
      </section>
    </div>
  </div>
</template>

<style scoped>
.segment-progression-page {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toolbar-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-count {
  color: #6b7280;
  font-size: 0.9rem;
}

.empty-state {
  border: 1px dashed #cfd6de;
  border-radius: 6px;
  color: #6b7280;
  padding: 14px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(300px, 360px) 1fr;
  gap: 10px;
}

.targets-list {
  max-height: 64vh;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.target-button {
  border: 1px solid #d8dee6;
  border-radius: 8px;
  background: white;
  text-align: left;
  padding: 10px;
}

.target-button.selected {
  border-color: #0d6efd;
  background: #eef5ff;
}

.target-name {
  font-weight: 600;
  margin-bottom: 4px;
}

.target-meta {
  color: #6b7280;
  font-size: 0.85rem;
}

.details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.summary-card {
  border: 1px solid #d8dee6;
  border-radius: 8px;
  padding: 10px;
  background: #fbfdff;
}

.summary-title {
  margin: 0 0 6px 0;
}

.summary-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 0.92rem;
}

.weather-note {
  display: block;
  margin-top: 8px;
  color: #6b7280;
}
</style>

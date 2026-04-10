<script setup lang="ts">
import { computed, ref, watch } from "vue";
import VGrid, { VGridVueTemplate } from "@revolist/vue3-datagrid";
import type { PersonalRecordTimeline } from "@/models/personal-record-timeline.model";
import ActivityCellRenderer from "./cell-renderers/ActivityCellRenderer.vue";

const props = defineProps<{
  timeline: PersonalRecordTimeline[];
}>();

const ALL_METRICS = "__ALL__";
const selectedMetric = ref(ALL_METRICS);

const metricOptions = computed(() => {
  const uniqueMetrics = Array.from(
    new Map(
      props.timeline.map((entry) => [entry.metricKey, entry.metricLabel])
    ).entries()
  );

  return [
    { key: ALL_METRICS, label: "All metrics" },
    ...uniqueMetrics
      .map(([key, label]) => ({ key, label }))
      .sort((left, right) => left.label.localeCompare(right.label)),
  ];
});

watch(
  metricOptions,
  (options) => {
    if (!options.some((option) => option.key === selectedMetric.value)) {
      selectedMetric.value = ALL_METRICS;
    }
  },
  { immediate: true }
);

const filteredTimeline = computed(() => {
  if (selectedMetric.value === ALL_METRICS) {
    return props.timeline;
  }

  return props.timeline.filter(
    (entry) => entry.metricKey === selectedMetric.value
  );
});

const rows = computed(() =>
  filteredTimeline.value.map((entry) => ({
    ...entry,
    activityDate: entry.activityDate.split("T")[0],
    previousValue: entry.previousValue ?? "-",
    improvement: entry.improvement ?? "Initial PR",
  }))
);

const columns = [
  { prop: "activityDate", name: "Date", size: 140 },
  { prop: "metricLabel", name: "Metric", size: 230 },
  { prop: "value", name: "PR", size: 220 },
  { prop: "previousValue", name: "Previous PR", size: 220 },
  { prop: "improvement", name: "Improvement", size: 220 },
  {
    prop: "activity",
    name: "Activity",
    size: 430,
    cellTemplate: VGridVueTemplate(ActivityCellRenderer),
  },
];
</script>

<template>
  <div class="timeline-wrapper">
    <div class="timeline-toolbar">
      <label for="timelineMetric" class="form-label mb-0">Metric</label>
      <select id="timelineMetric" v-model="selectedMetric" class="form-select form-select-sm metric-select">
        <option
          v-for="option in metricOptions"
          :key="option.key"
          :value="option.key"
        >
          {{ option.label }}
        </option>
      </select>
      <span class="timeline-count">{{ rows.length }} PR events</span>
    </div>

    <div v-if="rows.length === 0" class="timeline-empty">
      No PR events found for the selected filters.
    </div>

    <VGrid
      v-else
      name="personalRecordsTimelineGrid"
      theme="material"
      :columns="columns"
      :source="rows"
      :readonly="true"
      style="height: 42vh;"
    />
  </div>
</template>

<style scoped>
.timeline-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.timeline-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.metric-select {
  max-width: 320px;
}

.timeline-count {
  color: #6b7280;
  font-size: 0.9rem;
}

.timeline-empty {
  border: 1px dashed #cfd6de;
  border-radius: 6px;
  color: #6b7280;
  font-size: 0.95rem;
  padding: 16px;
}
</style>

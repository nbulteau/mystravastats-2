<script setup lang="ts">
import { computed, ref, watch } from "vue";
import VGrid, { VGridVueTemplate } from "@revolist/vue3-datagrid";
import type { PersonalRecordTimeline } from "@/models/personal-record-timeline.model";
import ActivityCellRenderer from "./cell-renderers/ActivityCellRenderer.vue";
import MetricLabelCellRenderer from "./cell-renderers/MetricLabelCellRenderer.vue";

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
  { prop: "metricLabel", name: "Metric", size: 230, cellTemplate: VGridVueTemplate(MetricLabelCellRenderer) },
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

    <section
      v-else
      class="grid-shell"
    >
      <VGrid
        name="personalRecordsTimelineGrid"
        theme="material"
        :columns="columns"
        :source="rows"
        :readonly="true"
        style="height: 42vh;"
      />
    </section>
  </div>
</template>

<style scoped>
.timeline-wrapper {
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid #d7e2ef;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.82);
  box-shadow: 0 12px 24px rgba(24, 39, 75, 0.07);
  padding: 10px;
}

.timeline-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.timeline-toolbar :deep(label) {
  color: #2f3c50;
  font-weight: 600;
}

.metric-select {
  max-width: 340px;
  min-width: 220px;
  border-radius: 10px;
  border: 1px solid #c7d7e8;
  background: #ffffff;
}

.timeline-count {
  color: #5d6979;
  font-size: 0.9rem;
  margin-left: auto;
}

.timeline-empty {
  border: 1px dashed #cfd6de;
  border-radius: 10px;
  color: #5d6979;
  font-size: 0.95rem;
  padding: 16px;
  background: #f8fbff;
}

@media (max-width: 992px) {
  .metric-select {
    min-width: 180px;
  }

  .timeline-count {
    width: 100%;
    margin-left: 0;
  }
}
</style>

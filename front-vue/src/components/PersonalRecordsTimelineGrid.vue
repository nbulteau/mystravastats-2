<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
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

function parseImprovementValue(improvement?: string): number | null {
  if (!improvement) return null;
  if (/initial\s+pr/i.test(improvement)) return null;
  const match = improvement.match(/[-+]?\d+(?:[.,]\d+)?/);
  if (!match) return null;
  const parsed = Number(match[0].replace(",", "."));
  if (!Number.isFinite(parsed)) return null;
  return Math.abs(parsed);
}

function toSortableDateMs(dateValue: string): number {
  const normalized = dateValue.includes("T")
    ? dateValue
    : dateValue.replace(" ", "T");
  const parsed = Date.parse(normalized);
  if (!Number.isNaN(parsed)) {
    return parsed;
  }

  // Support legacy timezone offsets without colon (for example +0200).
  const normalizedOffset = normalized.replace(/([+-]\d{2})(\d{2})$/, "$1:$2");
  const parsedWithNormalizedOffset = Date.parse(normalizedOffset);
  if (!Number.isNaN(parsedWithNormalizedOffset)) {
    return parsedWithNormalizedOffset;
  }

  // Fallback for non-ISO variants (for example offset without colon): use day precision.
  const dayPrefix = dateValue.slice(0, 10);
  if (/^\d{4}-\d{2}-\d{2}$/.test(dayPrefix)) {
    const dayParsed = Date.parse(`${dayPrefix}T00:00:00Z`);
    if (!Number.isNaN(dayParsed)) {
      return dayParsed;
    }
  }

  // Keep deterministic ordering even if backend sent an unexpected format.
  return -1;
}

const rows = computed(() => {
  const enriched = filteredTimeline.value.map((entry) => {
    const sortableDate = toSortableDateMs(entry.activityDate);
    return {
      ...entry,
      sortableDate,
      improvementValue: parseImprovementValue(entry.improvement),
      activityDate: entry.activityDate.split("T")[0],
      previousValue: entry.previousValue ?? "-",
      improvement: entry.improvement ?? "Initial PR",
    };
  });

  const sorted = [...enriched].sort((left, right) => {
    if (left.sortableDate === right.sortableDate) {
      return left.activityDate.localeCompare(right.activityDate);
    }
    return left.sortableDate - right.sortableDate;
  });

  return sorted;
});

const summary = computed(() => {
  const entries = rows.value;
  const latest = [...entries].sort((left, right) => right.sortableDate - left.sortableDate)[0] ?? null;
  const bestImprovement = [...entries]
    .filter((entry) => entry.improvementValue !== null)
    .sort((left, right) => (right.improvementValue ?? 0) - (left.improvementValue ?? 0))[0] ?? null;

  return {
    count: entries.length,
    latestDate: latest?.activityDate ?? "-",
    bestImprovement: bestImprovement?.improvement ?? "-",
  };
});

const viewportWidth = ref(
  typeof window === "undefined" ? 1600 : window.innerWidth
);

function updateViewportWidth(): void {
  if (typeof window !== "undefined") {
    viewportWidth.value = window.innerWidth;
  }
}

onMounted(() => {
  updateViewportWidth();
  if (typeof window !== "undefined") {
    window.addEventListener("resize", updateViewportWidth);
  }
});

onBeforeUnmount(() => {
  if (typeof window !== "undefined") {
    window.removeEventListener("resize", updateViewportWidth);
  }
});

const isCompactGrid = computed(() => viewportWidth.value <= 1450);

const columns = computed(() => {
  if (isCompactGrid.value) {
    return [
      { prop: "activityDate", name: "Date", size: 98 },
      {
        prop: "metricLabel",
        name: "Metric",
        size: 150,
        cellTemplate: VGridVueTemplate(MetricLabelCellRenderer),
      },
      { prop: "value", name: "PR", size: 160 },
      { prop: "previousValue", name: "Previous PR", size: 160 },
      { prop: "improvement", name: "Improvement", size: 150 },
      {
        prop: "activity",
        name: "Activity",
        size: 250,
        cellTemplate: VGridVueTemplate(ActivityCellRenderer),
      },
    ];
  }

  return [
    { prop: "activityDate", name: "Date", size: 125 },
    {
      prop: "metricLabel",
      name: "Metric",
      size: 185,
      cellTemplate: VGridVueTemplate(MetricLabelCellRenderer),
    },
    { prop: "value", name: "PR", size: 200 },
    { prop: "previousValue", name: "Previous PR", size: 200 },
    { prop: "improvement", name: "Improvement", size: 180 },
    {
      prop: "activity",
      name: "Activity",
      size: 320,
      cellTemplate: VGridVueTemplate(ActivityCellRenderer),
    },
  ];
});
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
      <span class="timeline-count">{{ summary.count }} PR events</span>
    </div>

    <div class="timeline-summary">
      <div class="timeline-summary-tile">
        <div class="timeline-summary-label">PR events</div>
        <div class="timeline-summary-value">{{ summary.count }}</div>
      </div>
      <div class="timeline-summary-tile">
        <div class="timeline-summary-label">Latest PR date</div>
        <div class="timeline-summary-value">{{ summary.latestDate }}</div>
      </div>
      <div class="timeline-summary-tile">
        <div class="timeline-summary-label">Best improvement</div>
        <div class="timeline-summary-value">{{ summary.bestImprovement }}</div>
      </div>
    </div>

    <div v-if="rows.length === 0" class="timeline-empty">
      No PR events found for the selected metric.
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
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 10px;
  min-width: 0;
}

.timeline-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.timeline-toolbar :deep(label) {
  color: var(--ms-text);
  font-weight: 600;
}

.metric-select {
  max-width: 340px;
  min-width: 220px;
  border-radius: 10px;
  border: 1px solid var(--ms-border);
  background: #ffffff;
}

.timeline-summary {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.timeline-summary-tile {
  border: 1px solid #eceef4;
  border-radius: 10px;
  padding: 8px 10px;
  background: #fafbfe;
}

.timeline-summary-label {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.timeline-summary-value {
  color: var(--ms-text);
  font-weight: 700;
}

.timeline-count {
  color: var(--ms-text-muted);
  font-size: 0.9rem;
  margin-left: auto;
}

.timeline-empty {
  border: 1px dashed var(--ms-border);
  border-radius: 10px;
  color: var(--ms-text-muted);
  font-size: 0.95rem;
  padding: 16px;
  background: #fafbfd;
}

.grid-shell {
  min-width: 0;
}

@media (max-width: 992px) {
  .metric-select {
    min-width: 180px;
  }

  .timeline-summary {
    grid-template-columns: 1fr;
  }

  .timeline-count {
    width: 100%;
    margin-left: 0;
  }
}
</style>

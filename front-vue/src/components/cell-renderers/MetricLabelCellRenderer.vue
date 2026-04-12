<script setup lang="ts">
import { computed } from "vue";
import { getMetricTooltip } from "@/utils/metric-tooltips";

const props = defineProps<{
  model: {
    label?: string;
    metricLabel?: string;
  };
}>();

const label = computed(() => props.model.metricLabel ?? props.model.label ?? "");
const tooltip = computed(() => getMetricTooltip(label.value));
</script>

<template>
  <span
    :class="['metric-label-cell', { 'has-tooltip': Boolean(tooltip) }]"
    :title="tooltip ?? undefined"
  >
    {{ label }}
  </span>
</template>

<style scoped>
.metric-label-cell {
  color: var(--ms-text);
}

.metric-label-cell.has-tooltip {
  border-bottom: 1px dotted #b46d4b;
  cursor: help;
}
</style>

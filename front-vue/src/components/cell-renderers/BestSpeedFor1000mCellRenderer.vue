<template>
  <div class="speed-cell">
    <span>{{ bestSpeedLabel }}</span>
  </div>
</template>

<script setup lang="ts">
import type {Activity} from "@/models/activity.model.ts";
import {formatSpeedWithUnit} from "@/utils/formatters.ts";
import { computed } from "vue";


const props = defineProps<{
  model: Activity;
}>();

const bestSpeedLabel = computed(() => {
  const bestSpeed = Number(props.model.bestSpeedForDistanceFor1000m);
  if (!Number.isFinite(bestSpeed) || bestSpeed <= 0) {
    return "Not available";
  }
  return formatSpeedWithUnit(bestSpeed, props.model.type);
});

</script>

<style scoped>
.speed-cell {
  text-align: right;
}
</style>

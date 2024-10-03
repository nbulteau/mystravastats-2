<template>
  <div class="speed-cell">
    <span>{{ formatSpeed(value, model.type) }}</span>
  </div>
</template>

<script setup lang="ts">
import type { Activity } from "@/models/activity.model";
import { defineProps } from "vue";


defineProps<{
  value: number;
  model: Activity; 
}>();

/**
 * Format average speed (m/s)
 */
function formatSpeed(speed: number, activityType: string): string {
  if (activityType === "Run") {
    return `${formatSeconds(1000 / speed)}/km`;
  } else {
    return `${(speed * 3.6).toFixed(2)} km/h`;
  }
}

/**
 * Format seconds to minutes and seconds
 */
 function formatSeconds(seconds: number): string {
  let min = Math.floor((seconds % 3600) / 60);
  let sec = Math.floor(seconds % 60);
  const hnd = Math.floor((seconds - min * 60 - sec) * 100 + 0.5);

  if (hnd === 100) {
    sec++;
    if (sec === 60) {
      sec = 0;
      min++;
    }
  }

  return `${min}'${sec < 10 ? '0' : ''}${sec}`;
}
</script>

<style scoped>
.speed-cell {
  text-align: right;
}
</style>

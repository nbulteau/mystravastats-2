<template>
  <div class="speed-cell">
    <span>{{ formatSpeedWithUnit(value, type) }}</span>
  </div>
</template>

<script setup lang="ts">
import {formatSpeedWithUnit} from "@/utils/formatters.ts";


defineProps<{
  value: number;
  type: string;
}>();


</script>

<style scoped>
.speed-cell {
  text-align: right;
}
</style>

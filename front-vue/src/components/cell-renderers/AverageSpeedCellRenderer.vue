<template>
  <div class="speed-cell">
    <span>{{ formatSpeedWithUnit(model.averageSpeed, model.type) }}</span>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from "vue";
import type {Activity} from "@/models/activity.model.ts";
import {formatSpeedWithUnit} from "@/utils/formatters.ts";


defineProps<{
  model: Activity;
}>();


</script>

<style scoped>
.speed-cell {
  text-align: right;
}
</style>

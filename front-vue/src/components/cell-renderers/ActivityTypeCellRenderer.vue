<template>
  <div class="activity-type-cell">
    <img
      v-if="iconUrl"
      :src="iconUrl"
      :alt="label"
      width="16"
      height="16"
    >
    <span>{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Activity } from "@/models/activity.model";
import iconRide from "@/assets/buttons/road-bike.png";
import iconMountainBikeRide from "@/assets/buttons/mountain-bike.png";
import iconCommute from "@/assets/buttons/city-bike.png";
import iconGravelRide from "@/assets/buttons/touring-bike.png";
import iconVirtualRide from "@/assets/buttons/virtual-bike.png";
import iconRun from "@/assets/buttons/run.png";
import iconTrailRun from "@/assets/buttons/trail-run.png";
import iconHike from "@/assets/buttons/hike.png";
import iconWalk from "@/assets/buttons/walk.png"
import iconAlpineSki from "@/assets/buttons/alpine-ski.png";

const props = defineProps<{
  model: Activity;
}>();

const ICON_BY_ACTIVITY_TYPE: Record<string, string> = {
  Ride: iconRide,
  MountainBikeRide: iconMountainBikeRide,
  Commute: iconCommute,
  GravelRide: iconGravelRide,
  VirtualRide: iconVirtualRide,
  Run: iconRun,
  TrailRun: iconTrailRun,
  Hike: iconHike,
  Walk: iconWalk,
  AlpineSki: iconAlpineSki,
};

const label = computed(() => props.model.type || "Unknown");
const iconUrl = computed(() => ICON_BY_ACTIVITY_TYPE[label.value] ?? null);
</script>

<style scoped>
.activity-type-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}
</style>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap/dist/js/bootstrap.bundle.min.js'; // Import Bootstrap JS
import { useContextStore } from "@/stores/context";
import { ToastTypeEnum } from "@/models/toast.model";

const props = defineProps<{
  badgeCheckResult: BadgeCheckResult;
}>();
const contextStore = useContextStore();

import runningBadge from "@/assets/badges/running.png";
import cyclingBadge from "@/assets/badges/cycling.png";
import racingBadge from "@/assets/badges/racing.png";
import hickingBadge from "@/assets/badges/hiking.png";
import stopwatchBadge from "@/assets/badges/stopwatch.png";
import badge from "@/assets/badges/badge.png";
import { Tooltip } from 'bootstrap';
import type { Activity } from '@/models/activity.model';

const buildBadgeImageUrl = (type: string) => {
  switch (type) {
    case "RunMovingTimeBadge":
    case "RideMovingTimeBadge":
    case "HikeMovingTimeBadge":
      return stopwatchBadge;
    case "RunDistanceBadge":
    case "RunElevationBadge":
      return runningBadge;
    case "RideDistanceBadge":
      return racingBadge;
    case "RideFamousClimbBadge":
    case "RideElevationBadge":
      return cyclingBadge;
    case "HikeDistanceBadge":
    case "HikeElevationBadge":
      return hickingBadge;
    default:
      return badge;
  }
};

const navigateToActivity = () => {
  if (props.badgeCheckResult.nbCheckedActivities <= 0 || !props.badgeCheckResult.activities?.length) {
    return;
  }

  const [latestActivity] = props.badgeCheckResult.activities;
  if (!latestActivity?.link) {
    return;
  }

  if (props.badgeCheckResult.activities.length > 1) {
    contextStore.showToast({
      id: `badge-toast-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: `Opening latest activity only (${props.badgeCheckResult.activities.length} linked to this badge).`,
      timeout: 3000,
    });
  }

  window.open(latestActivity.link, '_blank', 'noopener,noreferrer');
};

const badgeRef = ref<HTMLElement | null>(null);

const tooltipText = computed(() => {
  return `<strong>${props.badgeCheckResult.badge.label}</strong><br>
  A total of ${props.badgeCheckResult.nbCheckedActivities} activities<br>
  ${props.badgeCheckResult.activities && props.badgeCheckResult.activities.length > 0 ? 'Last activity is:<br>' : ''}
  ${props.badgeCheckResult.activities ? props.badgeCheckResult.activities.map((value: Activity) => `• ${value.name}`).join('<br>') : ''}
  `;  
});

function initTooltip() {
  if (badgeRef.value) {
    const tooltip = new Tooltip(badgeRef.value, {});
    tooltip.setContent({ '.tooltip-inner': tooltipText.value });
  }
}

function updateTooltip() {
  if (badgeRef.value) {
    const tooltip = Tooltip.getInstance(badgeRef.value);
    if (tooltip) {
      tooltip.setContent({ '.tooltip-inner': tooltipText.value });
    }
  }
}

watch(
  () => props.badgeCheckResult,
  () => {
    updateTooltip();
  },
  { immediate: true }
);

onMounted(() => {
  initTooltip();
});

</script>

<template>
  <div
    ref="badgeRef"
    class="badge-item card text-center p-2 border border-primary bg-light"
    data-bs-toggle="tooltip"
    data-bs-html="true"
    :title="tooltipText"
    @click="navigateToActivity" 
  >
    <div
      class="d-flex justify-content-center align-items-center"
      :class="{ 'badge-item--disabled': (props.badgeCheckResult.nbCheckedActivities <= 0) }"
    >
      <img
        :src="buildBadgeImageUrl(props.badgeCheckResult.badge.type)"
        class="badge-image card-img-top"
        :alt="props.badgeCheckResult.badge.label"
      >
    </div>
    <div>
      <span
        class="badge-label card-title text-center"
      >
        {{ props.badgeCheckResult.badge.label }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.badge-label {
  font-size: 1rem;
  line-height: 1.2;
  color: #2e3f56;
  font-weight: 700;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
}

.badge-item {
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
  width: 170px;
  min-height: 176px;
  border-radius: 16px;
  border: 1px solid #d3deed;
  background: linear-gradient(180deg, #ffffff, #f6faff);
  box-shadow: 0 10px 22px rgba(24, 39, 75, 0.08);
}

.badge-item:hover {
  transform: translateY(-4px);
  box-shadow: 0 16px 30px rgba(24, 39, 75, 0.16);
}

.badge-item--disabled {
  cursor: not-allowed;
  opacity: 0.5;
  pointer-events: none;
  background: #eef3f8;
  border-color: #c8d2df;
  color: #6c757d;
}

.badge-image {
  width: 92px;
  height: 92px;
  object-fit: cover;
  margin: auto;
}
</style>

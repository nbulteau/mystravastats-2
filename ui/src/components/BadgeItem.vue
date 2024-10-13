<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { defineProps } from 'vue';
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap/dist/js/bootstrap.bundle.min.js'; // Import Bootstrap JS

const props = defineProps<{
  badgeCheckResult: BadgeCheckResult;
}>();

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
  if (props.badgeCheckResult.nbCheckedActivities > 0 && props.badgeCheckResult.activities) {
    // Open all activities in a new tab
    props.badgeCheckResult.activities.forEach((activity: Activity) => {
      window.open(activity.link, '_blank');
    });
  }
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
  font-size: 1.2em;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%; /* Allow the text to take up the full width */
}

.badge-item {
  cursor: pointer;
  transition: transform 0.2s;
  width: 180px; /* Set a fixed width for the card */
  height: 180px; /* Set a fixed height for the card */
  background-color: #f8f9fa; /* Light background color for enabled state */
  border-color: #007bff; /* Primary border color for enabled state */
}

.badge-item:hover {
  transform: scale(1.05);
}

.badge-item--disabled {
  cursor: not-allowed;
  opacity: 0.5;
  pointer-events: none;
  background-color: #e9ecef;
  /* Darker background color for disabled state */
  border-color: #6c757d;
  /* Secondary border color for disabled state */
  color: #6c757d;
  /* Secondary text color for disabled state */
}

.badge-image {
  width: 100px;
  height: 100px;
  object-fit: cover;
  margin: auto;
  /* Center the image */
}

/* Custom Tooltip Styles */
.tooltip-inner {
  --bs-tooltip-max-width: 300px; /* Define the custom property for max-width */
  max-width: var(--bs-tooltip-max-width); /* Apply the custom property */
  background-color: #343a40; /* Dark background color */
  color: #ffffff; /* White text color */
  font-size: 1rem; /* Increase font size */
  padding: 10px; /* Add padding */
  border-radius: 5px; /* Rounded corners */
}
</style>
<script setup lang="ts">
import { ref, onBeforeUnmount, onMounted, computed, watch } from 'vue';
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

const isUnlocked = computed(() => props.badgeCheckResult.nbCheckedActivities > 0);
const linkedActivitiesCount = computed(() => props.badgeCheckResult.activities?.length ?? 0);
const statusLabel = computed(() => (isUnlocked.value ? 'Acquis' : 'À débloquer'));

const navigateToActivity = () => {
  if (!isUnlocked.value || !props.badgeCheckResult.activities?.length) {
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
  Statut : ${statusLabel.value}<br>
  Activités correspondantes : ${props.badgeCheckResult.nbCheckedActivities}<br>
  ${props.badgeCheckResult.activities && props.badgeCheckResult.activities.length > 0 ? 'Dernière activité liée :<br>' : ''}
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

onBeforeUnmount(() => {
  if (badgeRef.value) {
    const tooltip = Tooltip.getInstance(badgeRef.value);
    tooltip?.dispose();
  }
});

</script>

<template>
  <div
    ref="badgeRef"
    class="badge-item card text-center"
    :class="{ 'badge-item--earned': isUnlocked, 'badge-item--locked': !isUnlocked }"
    data-bs-toggle="tooltip"
    data-bs-html="true"
    :title="tooltipText"
    @click="navigateToActivity" 
  >
    <div
      class="badge-status-pill"
      :class="{ 'badge-status-pill--earned': isUnlocked, 'badge-status-pill--locked': !isUnlocked }"
    >
      {{ statusLabel }}
    </div>
    <div
      class="badge-media d-flex justify-content-center align-items-center"
    >
      <img
        :src="buildBadgeImageUrl(props.badgeCheckResult.badge.type)"
        class="badge-image card-img-top"
        :class="{ 'badge-image--locked': !isUnlocked }"
        :alt="props.badgeCheckResult.badge.label"
      >
    </div>
    <div class="badge-content">
      <span
        class="badge-label card-title text-center"
      >
        {{ props.badgeCheckResult.badge.label }}
      </span>
      <span class="badge-meta">
        {{ isUnlocked ? `${linkedActivitiesCount} activité${linkedActivitiesCount > 1 ? 's' : ''} liée${linkedActivitiesCount > 1 ? 's' : ''}` : 'Aucune activité liée pour le moment' }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.badge-label {
  font-size: 0.98rem;
  line-height: 1.25;
  color: #27384b;
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
  position: relative;
  transition: transform 0.2s, box-shadow 0.2s, border-color 0.2s;
  width: 188px;
  min-height: 232px;
  border-radius: 20px;
  border: 2px solid transparent;
  overflow: hidden;
  padding: 12px 12px 10px;
}

.badge-item:hover {
  transform: translateY(-4px);
}

.badge-item--earned {
  cursor: pointer;
  background: linear-gradient(180deg, #fffdf4 0%, #fff5cd 100%);
  border-color: #f2c24b;
  box-shadow: 0 14px 28px rgba(146, 101, 16, 0.22);
}

.badge-item--earned:hover {
  box-shadow: 0 18px 32px rgba(146, 101, 16, 0.28);
}

.badge-item--locked {
  cursor: default;
  background: linear-gradient(180deg, #f8fafc 0%, #eef3f8 100%);
  border-color: #c6d1de;
  box-shadow: 0 8px 18px rgba(24, 39, 75, 0.11);
}

.badge-status-pill {
  position: absolute;
  top: 10px;
  right: 10px;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  padding: 3px 9px;
  text-transform: uppercase;
}

.badge-status-pill--earned {
  background: #fff0b3;
  color: #6f4f00;
  border: 1px solid #e6bc4e;
}

.badge-status-pill--locked {
  background: #e9eef5;
  color: #586a7f;
  border: 1px solid #c3ceda;
}

.badge-media {
  min-height: 122px;
}

.badge-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.badge-meta {
  min-height: 18px;
  font-size: 0.76rem;
  line-height: 1.2;
  color: #5d7086;
  font-weight: 600;
}

.badge-image {
  width: 102px;
  height: 102px;
  object-fit: cover;
  margin: auto;
  border-radius: 50%;
  border: 3px solid rgba(255, 255, 255, 0.8);
  box-shadow: 0 8px 14px rgba(24, 39, 75, 0.2);
}

.badge-image--locked {
  filter: grayscale(1) contrast(0.9) saturate(0.65);
  opacity: 0.75;
}
</style>

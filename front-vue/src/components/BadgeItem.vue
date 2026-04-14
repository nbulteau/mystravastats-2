<script setup lang="ts">
import { ref, onBeforeUnmount, onMounted, computed, watch } from 'vue';
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap/dist/js/bootstrap.bundle.min.js'; // Import Bootstrap JS
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";

const props = defineProps<{
  badgeCheckResult: BadgeCheckResult;
}>();
const uiStore = useUiStore();

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
    uiStore.showToast({
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
  color: var(--ms-text);
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
  border-radius: 16px;
  border: 1px solid var(--ms-border);
  overflow: hidden;
  padding: 12px 12px 10px;
  background: var(--ms-surface-strong);
}

.badge-item:hover {
  transform: translateY(-3px);
}

.badge-item--earned {
  cursor: pointer;
  background: linear-gradient(180deg, #fff8f4 0%, #fff1e8 100%);
  border-color: #ffc8b1;
  box-shadow: 0 0 0 2px rgba(252, 76, 2, 0.12), 0 14px 24px rgba(198, 72, 14, 0.22);
}

.badge-item--earned:hover {
  box-shadow: 0 0 0 2px rgba(252, 76, 2, 0.16), 0 16px 28px rgba(198, 72, 14, 0.28);
}

.badge-item--locked {
  cursor: default;
  background: linear-gradient(180deg, #f7f8fb 0%, #f1f4f8 100%);
  border-color: #d9e0ea;
  box-shadow: 0 8px 16px rgba(26, 34, 48, 0.08);
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
  background: #fff1e8;
  color: #b43900;
  border: 1px solid #ffc8b1;
}

.badge-status-pill--locked {
  background: #ecf1f7;
  color: #616f80;
  border: 1px solid #ced8e4;
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
  color: var(--ms-text-muted);
  font-weight: 600;
}

.badge-image {
  width: 102px;
  height: 102px;
  object-fit: cover;
  margin: auto;
  border-radius: 50%;
  border: 3px solid rgba(255, 255, 255, 0.9);
  box-shadow: 0 8px 14px rgba(15, 23, 42, 0.18);
}

.badge-image--locked {
  filter: grayscale(1) contrast(0.86) saturate(0.6);
  opacity: 0.66;
}
</style>

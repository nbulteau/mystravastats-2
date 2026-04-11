<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import BadgeItem from "@/components/BadgeItem.vue"; // Import the new component
import type { BadgeCheckResult } from "@/models/badge-check-result.model";

const contextStore = useContextStore();
contextStore.updateCurrentView("badges");

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivityType);
const generalBadgesCheckResults = computed(() => contextStore.generalBadgesCheckResults);
const famousClimbBadgesCheckResults = computed(() => contextStore.famousClimbBadgesCheckResults);

const handleBadgeClick = (badgeCheckResult: BadgeCheckResult) => {
  console.log('Badge clicked:', badgeCheckResult);
  // TODO: Implement the badge click logic
};


</script>

<template>
  <div class="badges-page">
    <section
      v-if="generalBadgesCheckResults.length"
      class="badges-section"
    >
      <p class="badges-title">
        {{ currentActivity }} general badges for {{ currentYear }}
      </p>
      <div class="row g-3 justify-content-center">
        <div
          v-for="badge in generalBadgesCheckResults"
          :key="badge.badge.label"
          class="col-lg-2 col-md-3 col-sm-4 col-6 d-flex justify-content-center"
        >
          <BadgeItem
            :badge-check-result="badge"
            @badge-clicked="handleBadgeClick"
          />
        </div>
      </div>
    </section>

    <section
      v-if="famousClimbBadgesCheckResults.length"
      class="badges-section"
    >
      <p class="badges-title">
        Famous climbs {{ currentActivity }} badges for {{ currentYear }}
      </p>
      <div class="row g-3 justify-content-center">
        <div
          v-for="badge in famousClimbBadgesCheckResults"
          :key="badge.badge.label"
          class="col-lg-2 col-md-3 col-sm-4 col-6 d-flex justify-content-center"
        >
          <BadgeItem
            :badge-check-result="badge"
            @badge-clicked="handleBadgeClick"
          />
        </div>
      </div>
    </section>
    <div
      v-if="!generalBadgesCheckResults.length && !famousClimbBadgesCheckResults.length"
      class="chart-empty"
    >
      No badges found for the selected filters.
    </div>
  </div>
</template>

<style scoped>
.badges-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.badges-section {
  border: 1px solid #d7e2ef;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.84);
  box-shadow: 0 12px 26px rgba(24, 39, 75, 0.08);
  padding: 14px 12px;
}

.badges-title {
  margin-bottom: 12px;
  text-align: center;
  color: #2a3c52;
  font-size: 1.05rem;
  font-weight: 700;
}
</style>

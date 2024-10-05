<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import BadgeItem from "@/components/BadgeItem.vue"; // Import the new component
import type { BadgeCheckResult } from "@/models/badge-check-result.model";

const contextStore = useContextStore();
contextStore.updateCurrentView("badges");

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivity);
const generalBadgesCheckResults = computed(() => contextStore.generalBadgesCheckResults);
const famousClimbBadgesCheckResults = computed(() => contextStore.famousClimbBadgesCheckResults);

const handleBadgeClick = (badgeCheckResult: BadgeCheckResult) => {
  console.log('Badge clicked:', badgeCheckResult);
  // Add your custom logic here
};
</script>

<template>
  <main class="container">
    <div
      v-if="generalBadgesCheckResults.length"
      class="row"
    >
      <p class="text-center">
        {{ currentActivity }} general Badges for {{ currentYear }}
      </p>
      <div
        v-for="badge in generalBadgesCheckResults"
        :key="badge.badge.label"
        class="col-2 col-sm col-md-4 col-lg-2 mb-1 d-flex justify-content-center"
      >
        <BadgeItem
          :badge-check-result="badge"
          @badge-clicked="handleBadgeClick"
        /> 
      </div>
    </div>

    <div
      v-if="famousClimbBadgesCheckResults.length"
      class="row"
    >
      <p class="text-center">
        Famous Climb {{ currentActivity }} Badges for {{ currentYear }}
      </p>
      <div
        v-for="badge in famousClimbBadgesCheckResults"
        :key="badge.badge.label"
        class="col-2 col-sm col-md-4 col-lg-2 mb-1 d-flex justify-content-center"
      >
        <BadgeItem
          :badge-check-result="badge"
          @badge-clicked="handleBadgeClick"
        /> 
      </div>
    </div>
  </main>
</template>

<style scoped>
</style>
<script setup lang="ts">
import StatisticsGrid from "@/components/StatisticsGrid.vue";
import PersonalRecordsTimelineGrid from "@/components/PersonalRecordsTimelineGrid.vue";
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("statistics");

const currentYear = computed(() => contextStore.currentYear);
const statistics = computed(() => contextStore.statistics);
const personalRecordsTimeline = computed(() => contextStore.personalRecordsTimeline);
</script>

<template>
  <div class="statistics-page">
    <StatisticsGrid
      :year="currentYear"
      :statistics="statistics"
      height="40vh"
    />
    <PersonalRecordsTimelineGrid :timeline="personalRecordsTimeline" />
  </div>
</template>

<style scoped>
.statistics-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>

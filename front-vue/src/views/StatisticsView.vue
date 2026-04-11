<script setup lang="ts">
import StatisticsGrid from "@/components/StatisticsGrid.vue";
import PersonalRecordsTimelineGrid from "@/components/PersonalRecordsTimelineGrid.vue";
import HeartRateZoneAnalysisPanel from "@/components/HeartRateZoneAnalysisPanel.vue";
import type { HeartRateZoneSettings } from "@/models/heart-rate-zone.model";
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("statistics");

const currentYear = computed(() => contextStore.currentYear);
const statistics = computed(() => contextStore.statistics);
const personalRecordsTimeline = computed(() => contextStore.personalRecordsTimeline);
const heartRateZoneSettings = computed(() => contextStore.heartRateZoneSettings);
const heartRateZoneAnalysis = computed(() => contextStore.heartRateZoneAnalysis);

async function onSaveHeartRateZoneSettings(settings: HeartRateZoneSettings) {
  await contextStore.saveHeartRateZoneSettings(settings);
}
</script>

<template>
  <div class="statistics-page">
    <StatisticsGrid
      :year="currentYear"
      :statistics="statistics"
      height="40vh"
    />
    <PersonalRecordsTimelineGrid :timeline="personalRecordsTimeline" />
    <HeartRateZoneAnalysisPanel
      :analysis="heartRateZoneAnalysis"
      :settings="heartRateZoneSettings"
      @save-settings="onSaveHeartRateZoneSettings"
    />
  </div>
</template>

<style scoped>
.statistics-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>

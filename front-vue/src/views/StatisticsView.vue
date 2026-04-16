<script setup lang="ts">
import StatisticsGrid from "@/components/StatisticsGrid.vue";
import PersonalRecordsTimelineGrid from "@/components/PersonalRecordsTimelineGrid.vue";
import HeartRateZoneAnalysisPanel from "@/components/HeartRateZoneAnalysisPanel.vue";
import type { HeartRateZoneSettings } from "@/models/heart-rate-zone.model";
import { useContextStore } from "@/stores/context.js";
import { useStatisticsStore } from "@/stores/statistics";
import { useAthleteStore } from "@/stores/athlete";
import { computed, onMounted } from "vue";

const contextStore = useContextStore();
const statisticsStore = useStatisticsStore();
const athleteStore = useAthleteStore();
onMounted(() => contextStore.updateCurrentView("statistics"));

const currentYear = computed(() => contextStore.currentYear);
const statistics = computed(() => statisticsStore.statistics);
const personalRecordsTimeline = computed(() => statisticsStore.personalRecordsTimeline);
const heartRateZoneSettings = computed(() => athleteStore.heartRateZoneSettings);
const heartRateZoneAnalysis = computed(() => statisticsStore.heartRateZoneAnalysis);

async function onSaveHeartRateZoneSettings(settings: HeartRateZoneSettings) {
  await athleteStore.saveHeartRateZoneSettings(settings);
  statisticsStore.invalidateCache();
  await statisticsStore.fetchHeartRateZoneAnalysis();
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

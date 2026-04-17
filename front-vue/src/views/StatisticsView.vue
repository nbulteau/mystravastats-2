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
const isStatisticsLoading = computed(() => statisticsStore.isStatisticsLoading);
const isPersonalRecordsTimelineLoading = computed(() => statisticsStore.isPersonalRecordsTimelineLoading);
const isHeartRateZoneAnalysisLoading = computed(() => statisticsStore.isHeartRateZoneAnalysisLoading);
const statisticsError = computed(() => statisticsStore.statisticsError);
const personalRecordsTimelineError = computed(() => statisticsStore.personalRecordsTimelineError);
const heartRateZoneAnalysisError = computed(() => statisticsStore.heartRateZoneAnalysisError);

async function onSaveHeartRateZoneSettings(settings: HeartRateZoneSettings) {
  await athleteStore.saveHeartRateZoneSettings(settings);
  statisticsStore.invalidateCache();
  await statisticsStore.fetchHeartRateZoneAnalysis();
}
</script>

<template>
  <div class="statistics-page">
    <section class="statistics-block">
      <div v-if="isStatisticsLoading" class="block-state">
        Loading statistics...
      </div>
      <div v-else-if="statisticsError && !statistics.length" class="block-state block-state--error">
        {{ statisticsError }}
      </div>
      <template v-else>
        <StatisticsGrid
          :year="currentYear"
          :statistics="statistics"
          height="40vh"
        />
        <div v-if="statisticsError" class="block-state block-state--warning mt-2">
          Displaying cached statistics. {{ statisticsError }}
        </div>
      </template>
    </section>

    <section class="statistics-block">
      <div v-if="isPersonalRecordsTimelineLoading" class="block-state">
        Loading PR timeline...
      </div>
      <div
        v-else-if="personalRecordsTimelineError && !personalRecordsTimeline.length"
        class="block-state block-state--error"
      >
        {{ personalRecordsTimelineError }}
      </div>
      <template v-else>
        <PersonalRecordsTimelineGrid :timeline="personalRecordsTimeline" />
        <div v-if="personalRecordsTimelineError" class="block-state block-state--warning mt-2">
          Displaying cached timeline. {{ personalRecordsTimelineError }}
        </div>
      </template>
    </section>

    <section class="statistics-block">
      <div v-if="isHeartRateZoneAnalysisLoading" class="block-state">
        Loading heart rate zone analysis...
      </div>
      <div
        v-else-if="heartRateZoneAnalysisError && !heartRateZoneAnalysis.hasHeartRateData"
        class="block-state block-state--error"
      >
        {{ heartRateZoneAnalysisError }}
      </div>
      <template v-else>
        <HeartRateZoneAnalysisPanel
          :analysis="heartRateZoneAnalysis"
          :settings="heartRateZoneSettings"
          @save-settings="onSaveHeartRateZoneSettings"
        />
        <div v-if="heartRateZoneAnalysisError" class="block-state block-state--warning mt-2">
          Displaying cached HR zone analysis. {{ heartRateZoneAnalysisError }}
        </div>
      </template>
    </section>
  </div>
</template>

<style scoped>
.statistics-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.statistics-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.block-state {
  border: 1px dashed var(--ms-border);
  border-radius: 10px;
  color: var(--ms-text-muted);
  font-size: 0.95rem;
  padding: 14px;
  background: #fafbfd;
}

.block-state--error {
  border-color: #ef9a9a;
  background: #fff3f3;
  color: #9f2d2d;
}

.block-state--warning {
  border-color: #ffd59f;
  background: #fff9ef;
  color: #8a5b1d;
}
</style>

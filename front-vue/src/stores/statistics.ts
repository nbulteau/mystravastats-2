import { defineStore } from "pinia";
import type { Statistics } from "@/models/statistics.model";
import type { PersonalRecordTimeline } from "@/models/personal-record-timeline.model";
import {
  type HeartRateZoneAnalysis,
  emptyHeartRateZoneAnalysis,
} from "@/models/heart-rate-zone.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";
import { useAthleteStore } from "@/stores/athlete";

type StatisticsCacheEntry = {
  statistics: Statistics[];
  personalRecordsTimeline: PersonalRecordTimeline[];
  heartRateZoneAnalysis: HeartRateZoneAnalysis;
};

export const useStatisticsStore = defineStore("statistics", {
  state: () => ({
    statistics: [] as Statistics[],
    personalRecordsTimeline: [] as PersonalRecordTimeline[],
    heartRateZoneAnalysis: emptyHeartRateZoneAnalysis() as HeartRateZoneAnalysis,
    isStatisticsLoading: false,
    isPersonalRecordsTimelineLoading: false,
    isHeartRateZoneAnalysisLoading: false,
    statisticsError: null as string | null,
    personalRecordsTimelineError: null as string | null,
    heartRateZoneAnalysisError: null as string | null,
    cacheByKey: {} as Record<string, StatisticsCacheEntry>,
  }),
  actions: {
    currentFiltersKey(): string {
      return useContextStore().currentFiltersKey;
    },
    applyCacheEntry(entry: StatisticsCacheEntry) {
      this.statistics = entry.statistics;
      this.personalRecordsTimeline = entry.personalRecordsTimeline;
      this.heartRateZoneAnalysis = entry.heartRateZoneAnalysis;
    },
    updateCacheForCurrentKey() {
      const key = this.currentFiltersKey();
      this.cacheByKey[key] = {
        statistics: this.statistics,
        personalRecordsTimeline: this.personalRecordsTimeline,
        heartRateZoneAnalysis: this.heartRateZoneAnalysis,
      };
    },
    invalidateCache() {
      this.cacheByKey = {};
    },
    async fetchStatistics() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("statistics", contextStore.currentActivityType, contextStore.currentYear);
      const key = this.currentFiltersKey();
      this.isStatisticsLoading = true;
      this.statisticsError = null;
      try {
        this.statistics = await requestJson<Statistics[]>(url);
        this.updateCacheForCurrentKey();
      } catch (error) {
        this.statisticsError = error instanceof Error ? error.message : "Unable to load statistics.";
        const cached = this.cacheByKey[key];
        if (cached) {
          this.statistics = cached.statistics;
        }
      } finally {
        this.isStatisticsLoading = false;
      }
    },
    async fetchPersonalRecordsTimeline() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "statistics/personal-records-timeline",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      const key = this.currentFiltersKey();
      this.isPersonalRecordsTimelineLoading = true;
      this.personalRecordsTimelineError = null;
      try {
        this.personalRecordsTimeline = await requestJson<PersonalRecordTimeline[]>(url);
        this.updateCacheForCurrentKey();
      } catch (error) {
        this.personalRecordsTimelineError = error instanceof Error ? error.message : "Unable to load PR timeline.";
        const cached = this.cacheByKey[key];
        if (cached) {
          this.personalRecordsTimeline = cached.personalRecordsTimeline;
        }
      } finally {
        this.isPersonalRecordsTimelineLoading = false;
      }
    },
    async fetchHeartRateZoneAnalysis() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "statistics/heart-rate-zones",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      const key = this.currentFiltersKey();
      this.isHeartRateZoneAnalysisLoading = true;
      this.heartRateZoneAnalysisError = null;
      try {
        this.heartRateZoneAnalysis = await requestJson<HeartRateZoneAnalysis>(url);
        this.updateCacheForCurrentKey();
      } catch (error) {
        this.heartRateZoneAnalysisError = error instanceof Error ? error.message : "Unable to load HR zone analysis.";
        const cached = this.cacheByKey[key];
        if (cached) {
          this.heartRateZoneAnalysis = cached.heartRateZoneAnalysis;
        }
      } finally {
        this.isHeartRateZoneAnalysisLoading = false;
      }
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      const cached = this.cacheByKey[key];
      if (!force && cached) {
        this.applyCacheEntry(cached);
        this.statisticsError = null;
        this.personalRecordsTimelineError = null;
        this.heartRateZoneAnalysisError = null;
        this.isStatisticsLoading = false;
        this.isPersonalRecordsTimelineLoading = false;
        this.isHeartRateZoneAnalysisLoading = false;
        return;
      }

      const athleteStore = useAthleteStore();
      await Promise.allSettled([
        this.fetchStatistics(),
        this.fetchPersonalRecordsTimeline(),
        athleteStore.fetchHeartRateZoneSettings(),
        this.fetchHeartRateZoneAnalysis(),
      ]);
    },
    async refreshStatisticsDomain() {
      await this.ensureLoaded(true);
    },
  },
});

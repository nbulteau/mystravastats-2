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
    cacheByKey: {} as Record<string, StatisticsCacheEntry>,
  }),
  actions: {
    currentFiltersKey(): string {
      const contextStore = useContextStore();
      return `${contextStore.currentActivityType}__${contextStore.currentYear}`;
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
      this.statistics = await requestJson<Statistics[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchPersonalRecordsTimeline() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "statistics/personal-records-timeline",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      this.personalRecordsTimeline = await requestJson<PersonalRecordTimeline[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchHeartRateZoneAnalysis() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "statistics/heart-rate-zones",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      this.heartRateZoneAnalysis = await requestJson<HeartRateZoneAnalysis>(url);
      this.updateCacheForCurrentKey();
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      const cached = this.cacheByKey[key];
      if (!force && cached) {
        this.applyCacheEntry(cached);
        return;
      }

      const athleteStore = useAthleteStore();
      await Promise.all([
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

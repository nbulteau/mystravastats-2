import { defineStore } from "pinia";
import type { Statistics } from "@/models/statistics.model";
import type { PersonalRecordTimeline } from "@/models/personal-record-timeline.model";
import {
  type HeartRateZoneAnalysis,
  emptyHeartRateZoneAnalysis,
} from "@/models/heart-rate-zone.model";
import { buildFilteredApiUrl, requestJson } from "@/services/http-client";
import { useContextStore } from "@/stores/context";
import { useAthleteStore } from "@/stores/athlete";

type StatisticsCacheEntry = {
  statistics?: Statistics[];
  personalRecordsTimeline?: PersonalRecordTimeline[];
  heartRateZoneAnalysis?: HeartRateZoneAnalysis;
};

function isCompleteCacheEntry(entry: StatisticsCacheEntry | undefined): entry is Required<StatisticsCacheEntry> {
  return !!entry?.statistics && !!entry.personalRecordsTimeline && !!entry.heartRateZoneAnalysis;
}

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
    applyCacheEntry(entry: Required<StatisticsCacheEntry>) {
      this.statistics = entry.statistics;
      this.personalRecordsTimeline = entry.personalRecordsTimeline;
      this.heartRateZoneAnalysis = entry.heartRateZoneAnalysis;
    },
    isCurrentFiltersKey(key: string): boolean {
      return this.currentFiltersKey() === key;
    },
    updateCacheForKey(key: string, entry: StatisticsCacheEntry) {
      this.cacheByKey[key] = {
        ...(this.cacheByKey[key] ?? {}),
        ...entry,
      };
    },
    invalidateCache() {
      this.cacheByKey = {};
    },
    async fetchStatistics() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("getStatistics", contextStore.currentActivityType, contextStore.currentYear);
      const key = this.currentFiltersKey();
      this.isStatisticsLoading = true;
      this.statisticsError = null;
      try {
        const statistics = await requestJson<Statistics[]>(url);
        this.updateCacheForKey(key, { statistics });
        if (this.isCurrentFiltersKey(key)) {
          this.statistics = statistics;
        }
      } catch (error) {
        if (!this.isCurrentFiltersKey(key)) {
          return;
        }
        this.statisticsError = error instanceof Error ? error.message : "Unable to load statistics.";
        const cached = this.cacheByKey[key];
        if (cached?.statistics) {
          this.statistics = cached.statistics;
        }
      } finally {
        if (this.isCurrentFiltersKey(key)) {
          this.isStatisticsLoading = false;
        }
      }
    },
    async fetchPersonalRecordsTimeline() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "getPersonalRecordsTimeline",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      const key = this.currentFiltersKey();
      this.isPersonalRecordsTimelineLoading = true;
      this.personalRecordsTimelineError = null;
      try {
        const personalRecordsTimeline = await requestJson<PersonalRecordTimeline[]>(url);
        this.updateCacheForKey(key, { personalRecordsTimeline });
        if (this.isCurrentFiltersKey(key)) {
          this.personalRecordsTimeline = personalRecordsTimeline;
        }
      } catch (error) {
        if (!this.isCurrentFiltersKey(key)) {
          return;
        }
        this.personalRecordsTimelineError = error instanceof Error ? error.message : "Unable to load PR timeline.";
        const cached = this.cacheByKey[key];
        if (cached?.personalRecordsTimeline) {
          this.personalRecordsTimeline = cached.personalRecordsTimeline;
        }
      } finally {
        if (this.isCurrentFiltersKey(key)) {
          this.isPersonalRecordsTimelineLoading = false;
        }
      }
    },
    async fetchHeartRateZoneAnalysis() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "getHeartRateZoneAnalysis",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      const key = this.currentFiltersKey();
      this.isHeartRateZoneAnalysisLoading = true;
      this.heartRateZoneAnalysisError = null;
      try {
        const heartRateZoneAnalysis = await requestJson<HeartRateZoneAnalysis>(url);
        this.updateCacheForKey(key, { heartRateZoneAnalysis });
        if (this.isCurrentFiltersKey(key)) {
          this.heartRateZoneAnalysis = heartRateZoneAnalysis;
        }
      } catch (error) {
        if (!this.isCurrentFiltersKey(key)) {
          return;
        }
        this.heartRateZoneAnalysisError = error instanceof Error ? error.message : "Unable to load HR zone analysis.";
        const cached = this.cacheByKey[key];
        if (cached?.heartRateZoneAnalysis) {
          this.heartRateZoneAnalysis = cached.heartRateZoneAnalysis;
        }
      } finally {
        if (this.isCurrentFiltersKey(key)) {
          this.isHeartRateZoneAnalysisLoading = false;
        }
      }
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      const cached = this.cacheByKey[key];
      if (!force && isCompleteCacheEntry(cached)) {
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

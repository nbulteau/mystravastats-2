import { defineStore } from "pinia";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

type BadgesCacheEntry = {
  generalBadgesCheckResults: BadgeCheckResult[];
  famousClimbBadgesCheckResults: BadgeCheckResult[];
};

function splitBadgeResults(badgeResults: BadgeCheckResult[]): BadgesCacheEntry {
  return {
    generalBadgesCheckResults: badgeResults.filter(
      (badgeCheckResult) => !badgeCheckResult.badge.type.endsWith("FamousClimbBadge"),
    ),
    famousClimbBadgesCheckResults: badgeResults.filter((badgeCheckResult) =>
      badgeCheckResult.badge.type.endsWith("FamousClimbBadge"),
    ),
  };
}

export const useBadgesStore = defineStore("badges", {
  state: () => ({
    generalBadgesCheckResults: [] as BadgeCheckResult[],
    famousClimbBadgesCheckResults: [] as BadgeCheckResult[],
    badgesByKey: {} as Record<string, BadgesCacheEntry>,
    isLoading: false,
    error: null as string | null,
    loadedFiltersKey: null as string | null,
    loadingFiltersKey: null as string | null,
  }),
  getters: {
    hasBadges: (state) =>
      state.generalBadgesCheckResults.length > 0 && state.famousClimbBadgesCheckResults.length > 0,
  },
  actions: {
    currentFiltersKey(): string {
      return useContextStore().currentFiltersKey;
    },
    setFromCacheEntry(entry: BadgesCacheEntry, key?: string) {
      const filtersKey = key ?? this.currentFiltersKey();
      this.generalBadgesCheckResults = entry.generalBadgesCheckResults;
      this.famousClimbBadgesCheckResults = entry.famousClimbBadgesCheckResults;
      this.loadedFiltersKey = filtersKey;
      this.error = null;
    },
    invalidateCache() {
      this.badgesByKey = {};
    },
    async fetchBadges() {
      const contextStore = useContextStore();
      const key = contextStore.currentFiltersKey;
      const url = buildFilteredApiUrl("badges", contextStore.currentActivityType, contextStore.currentYear);
      this.isLoading = true;
      this.loadingFiltersKey = key;
      this.error = null;
      try {
        const badgeResults = await requestJson<BadgeCheckResult[]>(url);
        const cacheEntry = splitBadgeResults(badgeResults);
        this.badgesByKey[key] = cacheEntry;
        if (this.currentFiltersKey() === key) {
          this.setFromCacheEntry(cacheEntry, key);
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Unable to load badges.";
        const cached = this.badgesByKey[key];
        if (cached && this.currentFiltersKey() === key) {
          this.setFromCacheEntry(cached, key);
        }
      } finally {
        if (this.loadingFiltersKey === key) {
          this.isLoading = false;
          this.loadingFiltersKey = null;
        }
      }
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      const cached = this.badgesByKey[key];
      if (!force && cached) {
        this.setFromCacheEntry(cached, key);
        return;
      }
      await this.fetchBadges();
    },
    async ensureFiltersLoaded(activityType: string, year: string): Promise<BadgesCacheEntry> {
      const key = `${activityType}__${year}`;
      const cached = this.badgesByKey[key];
      if (cached) return cached;
      const badgeResults = await requestJson<BadgeCheckResult[]>(buildFilteredApiUrl("badges", activityType, year));
      const entry = splitBadgeResults(badgeResults);
      this.badgesByKey[key] = entry;
      return entry;
    },
  },
});

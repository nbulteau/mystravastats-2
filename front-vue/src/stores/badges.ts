import { defineStore } from "pinia";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

type BadgesCacheEntry = {
  generalBadgesCheckResults: BadgeCheckResult[];
  famousClimbBadgesCheckResults: BadgeCheckResult[];
};

export const useBadgesStore = defineStore("badges", {
  state: () => ({
    generalBadgesCheckResults: [] as BadgeCheckResult[],
    famousClimbBadgesCheckResults: [] as BadgeCheckResult[],
    badgesByKey: {} as Record<string, BadgesCacheEntry>,
  }),
  getters: {
    hasBadges: (state) =>
      state.generalBadgesCheckResults.length > 0 && state.famousClimbBadgesCheckResults.length > 0,
  },
  actions: {
    currentFiltersKey(): string {
      const contextStore = useContextStore();
      return `${contextStore.currentActivityType}__${contextStore.currentYear}`;
    },
    setFromCacheEntry(entry: BadgesCacheEntry) {
      this.generalBadgesCheckResults = entry.generalBadgesCheckResults;
      this.famousClimbBadgesCheckResults = entry.famousClimbBadgesCheckResults;
    },
    async fetchBadges() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("badges", contextStore.currentActivityType, contextStore.currentYear);
      const badgeResults = await requestJson<BadgeCheckResult[]>(url);
      this.generalBadgesCheckResults = badgeResults.filter(
        (badgeCheckResult) => !badgeCheckResult.badge.type.endsWith("FamousClimbBadge"),
      );
      this.famousClimbBadgesCheckResults = badgeResults.filter((badgeCheckResult) =>
        badgeCheckResult.badge.type.endsWith("FamousClimbBadge"),
      );
      this.badgesByKey[this.currentFiltersKey()] = {
        generalBadgesCheckResults: this.generalBadgesCheckResults,
        famousClimbBadgesCheckResults: this.famousClimbBadgesCheckResults,
      };
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      const cached = this.badgesByKey[key];
      if (!force && cached) {
        this.setFromCacheEntry(cached);
        return;
      }
      await this.fetchBadges();
    },
  },
});

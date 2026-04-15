import { defineStore } from "pinia";
import type {
  SegmentEffort,
  SegmentSummary,
  SegmentTargetSummary,
} from "@/models/segment-analysis.model";
import { requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

type SegmentMetric = "TIME" | "SPEED";

const SEGMENTS_CACHE_STORAGE_KEY = "mystravastats:segments:cache:v2";
const SEGMENTS_LIST_CACHE_TTL_MS = 10 * 60 * 1000;
const SEGMENTS_DETAILS_CACHE_TTL_MS = 10 * 60 * 1000;
const SEGMENTS_LIST_CACHE_MAX_ENTRIES = 40;
const SEGMENTS_DETAILS_CACHE_MAX_ENTRIES = 120;

type SegmentListCacheEntry = {
  segments: SegmentTargetSummary[];
  updatedAt: number;
  expiresAt: number;
};

type SegmentDetailCacheEntry = {
  efforts: SegmentEffort[];
  summary: SegmentSummary | null;
  updatedAt: number;
  expiresAt: number;
};

type SegmentPersistentCachePayload = {
  version: 1;
  savedAt: number;
  listCacheByKey: Record<string, SegmentListCacheEntry>;
  detailsCacheByKey: Record<string, SegmentDetailCacheEntry>;
};

export const useSegmentsStore = defineStore("segments", {
  state: () => ({
    segments: [] as SegmentTargetSummary[],
    efforts: [] as SegmentEffort[],
    summary: null as SegmentSummary | null,
    selectedSegmentId: null as number | null,
    metric: "TIME" as SegmentMetric,
    query: "",
    from: "",
    to: "",
    listCacheByKey: {} as Record<string, SegmentListCacheEntry>,
    detailsCacheByKey: {} as Record<string, SegmentDetailCacheEntry>,
    persistentCacheHydrated: false,
  }),
  getters: {
    selectedSegment(state): SegmentTargetSummary | undefined {
      return state.segments.find((segment) => segment.targetId === state.selectedSegmentId);
    },
  },
  actions: {
    baseFiltersKey(): string {
      const contextStore = useContextStore();
      return [
        contextStore.currentActivityType,
        contextStore.currentYear,
        this.metric,
        this.from || "none",
        this.to || "none",
      ].join("__");
    },
    listCacheKey(): string {
      return `${this.baseFiltersKey()}__${this.query.trim().toLowerCase() || "all"}`;
    },
    detailCacheKey(segmentId: number): string {
      return `${this.baseFiltersKey()}__segment_${segmentId}`;
    },
    buildSegmentsUrl(path: string, extra: Record<string, string | undefined>): string {
      const contextStore = useContextStore();
      const params = new URLSearchParams({
        activityType: contextStore.currentActivityType,
      });

      if (contextStore.currentYear !== "All years") {
        params.set("year", contextStore.currentYear);
      }
      params.set("metric", this.metric);
      if (this.query.trim().length > 0) {
        params.set("query", this.query.trim());
      }
      if (this.from) {
        params.set("from", this.from);
      }
      if (this.to) {
        params.set("to", this.to);
      }

      for (const [key, value] of Object.entries(extra)) {
        if (value && value.length > 0) {
          params.set(key, value);
        }
      }

      return `/api/${path}?${params.toString()}`;
    },
    updateFilters(payload: {
      metric?: SegmentMetric;
      query?: string;
      from?: string;
      to?: string;
    }) {
      if (payload.metric) {
        this.metric = payload.metric;
      }
      if (payload.query !== undefined) {
        this.query = payload.query;
      }
      if (payload.from !== undefined) {
        this.from = payload.from;
      }
      if (payload.to !== undefined) {
        this.to = payload.to;
      }
    },
    ensurePersistentCacheHydrated() {
      if (this.persistentCacheHydrated) {
        return;
      }
      this.persistentCacheHydrated = true;

      if (typeof window === "undefined") {
        return;
      }

      const rawPayload = window.localStorage.getItem(SEGMENTS_CACHE_STORAGE_KEY);
      if (!rawPayload) {
        return;
      }

      try {
        const parsed = JSON.parse(rawPayload) as Partial<SegmentPersistentCachePayload>;
        const listCache = parsed.listCacheByKey ?? {};
        const detailsCache = parsed.detailsCacheByKey ?? {};
        this.listCacheByKey = listCache;
        this.detailsCacheByKey = detailsCache;
      } catch {
        this.listCacheByKey = {};
        this.detailsCacheByKey = {};
      }

      this.pruneExpiredCacheEntries();
    },
    persistCache() {
      if (typeof window === "undefined") {
        return;
      }

      this.pruneExpiredCacheEntries();
      const payload: SegmentPersistentCachePayload = {
        version: 1,
        savedAt: Date.now(),
        listCacheByKey: this.listCacheByKey,
        detailsCacheByKey: this.detailsCacheByKey,
      };
      window.localStorage.setItem(SEGMENTS_CACHE_STORAGE_KEY, JSON.stringify(payload));
    },
    pruneExpiredCacheEntries() {
      const now = Date.now();
      this.listCacheByKey = Object.fromEntries(
        Object.entries(this.listCacheByKey).filter(([, entry]) => entry.expiresAt > now),
      );
      this.detailsCacheByKey = Object.fromEntries(
        Object.entries(this.detailsCacheByKey).filter(([, entry]) => entry.expiresAt > now),
      );
    },
    trimCacheEntries() {
      const trimByRecency = <T extends { updatedAt: number }>(
        source: Record<string, T>,
        maxEntries: number,
      ): Record<string, T> => {
        const sortedEntries = Object.entries(source)
          .sort(([, left], [, right]) => right.updatedAt - left.updatedAt)
          .slice(0, maxEntries);
        return Object.fromEntries(sortedEntries);
      };

      this.listCacheByKey = trimByRecency(this.listCacheByKey, SEGMENTS_LIST_CACHE_MAX_ENTRIES);
      this.detailsCacheByKey = trimByRecency(this.detailsCacheByKey, SEGMENTS_DETAILS_CACHE_MAX_ENTRIES);
    },
    async fetchSegments() {
      this.ensurePersistentCacheHydrated();
      const url = this.buildSegmentsUrl("segments", {});
      this.segments = await requestJson<SegmentTargetSummary[]>(url);
      const now = Date.now();
      this.listCacheByKey[this.listCacheKey()] = {
        segments: this.segments,
        updatedAt: now,
        expiresAt: now + SEGMENTS_LIST_CACHE_TTL_MS,
      };
      this.trimCacheEntries();
      this.persistCache();
    },
    async fetchSegmentEfforts(segmentId: number): Promise<SegmentEffort[]> {
      const url = this.buildSegmentsUrl(`segments/${segmentId}/efforts`, {});
      return requestJson<SegmentEffort[]>(url);
    },
    async fetchSegmentSummary(segmentId: number): Promise<SegmentSummary | null> {
      const url = this.buildSegmentsUrl(`segments/${segmentId}/summary`, {});
      try {
        return await requestJson<SegmentSummary>(url);
      } catch {
        return null;
      }
    },
    async ensureSegmentDetailsLoaded(segmentId: number, force = false) {
      const key = this.detailCacheKey(segmentId);
      const cached = this.detailsCacheByKey[key];
      if (!force && cached && cached.expiresAt > Date.now()) {
        this.efforts = cached.efforts;
        this.summary = cached.summary;
        return;
      }

      const [efforts, summary] = await Promise.all([
        this.fetchSegmentEfforts(segmentId),
        this.fetchSegmentSummary(segmentId),
      ]);
      this.efforts = efforts;
      this.summary = summary;
      const now = Date.now();
      this.detailsCacheByKey[key] = {
        efforts,
        summary,
        updatedAt: now,
        expiresAt: now + SEGMENTS_DETAILS_CACHE_TTL_MS,
      };
      this.trimCacheEntries();
      this.persistCache();
    },
    async selectSegment(segmentId: number, force = false) {
      this.selectedSegmentId = segmentId;
      await this.ensureSegmentDetailsLoaded(segmentId, force);
    },
    async ensureLoaded(force = false) {
      this.ensurePersistentCacheHydrated();
      this.pruneExpiredCacheEntries();

      const key = this.listCacheKey();
      const cached = this.listCacheByKey[key];
      if (!force && cached) {
        this.segments = cached.segments;
      } else {
        await this.fetchSegments();
      }

      if (this.segments.length === 0) {
        this.selectedSegmentId = null;
        this.efforts = [];
        this.summary = null;
        return;
      }

      const preferredSegmentId =
        this.selectedSegmentId && this.segments.some((segment) => segment.targetId === this.selectedSegmentId)
          ? this.selectedSegmentId
          : this.segments[0].targetId;

      await this.selectSegment(preferredSegmentId, force);
    },
    invalidateCache() {
      this.listCacheByKey = {};
      this.detailsCacheByKey = {};
      this.persistCache();
    },
  },
});

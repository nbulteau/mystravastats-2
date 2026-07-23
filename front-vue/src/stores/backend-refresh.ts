import { defineStore } from "pinia";
import type { HealthDetailsPayload } from "@/models/health.model";
import { requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

const DEFAULT_POLL_INTERVAL_MS = 2000;
const DEFAULT_IDLE_POLL_INTERVAL_MS = 10000;
const DEFAULT_MAX_POLLS = Number.POSITIVE_INFINITY;

type ActivityDatasetWatchOptions = {
  pollIntervalMs?: number;
  idlePollIntervalMs?: number;
  maxPolls?: number;
};

function wait(ms: number): Promise<void> {
  if (ms <= 0) {
    return Promise.resolve();
  }
  return new Promise((resolve) => globalThis.setTimeout(resolve, ms));
}

function isBackgroundRefreshInProgress(health: HealthDetailsPayload): boolean {
  return health.refresh?.backgroundInProgress === true;
}

function activityCount(health: HealthDetailsPayload): number | null {
  return typeof health.activities === "number" ? health.activities : null;
}

function activityDatasetFingerprint(health: HealthDetailsPayload): string {
  const activeProviders = [...(health.composite?.activeProviders ?? [])].sort();
  const sources = (health.composite?.sources ?? [])
    .map((source) => ({
      provider: source.provider ?? "",
      athleteId: source.athleteId ?? "",
      cacheRoot: source.cacheRoot ?? "",
      activities: source.activities ?? null,
      availableYearBins: [...(source.availableYearBins ?? [])].map(String).sort(),
    }))
    .sort((left, right) =>
      `${left.provider}\u0000${left.athleteId}\u0000${left.cacheRoot}`.localeCompare(
        `${right.provider}\u0000${right.athleteId}\u0000${right.cacheRoot}`,
      ),
    );

  return JSON.stringify({
    provider: health.provider ?? "",
    athleteId: health.athleteId ?? "",
    cacheRoot: health.cacheRoot ?? "",
    fitDirectory: health.fitDirectory ?? "",
    gpxDirectory: health.gpxDirectory ?? "",
    activities: activityCount(health),
    availableYearBins: [...(health.availableYearBins ?? [])].map(String).sort(),
    activeProviders,
    sources,
    sourceSyncCompletedAt: health.sourceSync?.completedAt ?? "",
  });
}

export const useBackendRefreshStore = defineStore("backendRefresh", {
  state: () => ({
    isWatchingStartupRefresh: false,
    observedStartupRefresh: false,
    lastActivityCount: null as number | null,
    lastDatasetFingerprint: null as string | null,
    error: null as string | null,
  }),
  actions: {
    async watchStartupActivityRefresh(options: ActivityDatasetWatchOptions = {}) {
      if (this.isWatchingStartupRefresh) {
        return;
      }

      const pollIntervalMs = options.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;
      const idlePollIntervalMs =
        options.idlePollIntervalMs ?? options.pollIntervalMs ?? DEFAULT_IDLE_POLL_INTERVAL_MS;
      const maxPolls = options.maxPolls ?? DEFAULT_MAX_POLLS;

      this.isWatchingStartupRefresh = true;
      this.observedStartupRefresh = false;
      this.error = null;

      try {
        let waitingForRefreshCompletion = false;
        for (let pollIndex = 0; pollIndex < maxPolls; pollIndex += 1) {
          let health: HealthDetailsPayload;
          try {
            health = await requestJson<HealthDetailsPayload>("/api/health/details", {
              method: "GET",
              headers: {
                Accept: "application/json",
              },
            });
            this.error = null;
          } catch (error) {
            this.error = error instanceof Error ? error.message : "Unable to watch backend activity refresh.";
            if (pollIndex + 1 < maxPolls) {
              await wait(idlePollIntervalMs);
            }
            continue;
          }
          const currentActivityCount = activityCount(health);
          const currentDatasetFingerprint = activityDatasetFingerprint(health);
          const datasetChanged =
            this.lastDatasetFingerprint !== null &&
            this.lastDatasetFingerprint !== currentDatasetFingerprint;

          if (currentActivityCount !== null) {
            this.lastActivityCount = currentActivityCount;
          }
          this.lastDatasetFingerprint = currentDatasetFingerprint;

          let refreshedThisPoll = false;
          if (datasetChanged) {
            await useContextStore().refreshAfterActivityDataChanged();
            refreshedThisPoll = true;
          }

          if (isBackgroundRefreshInProgress(health)) {
            this.observedStartupRefresh = true;
            waitingForRefreshCompletion = true;
            await wait(pollIntervalMs);
            continue;
          }

          if (waitingForRefreshCompletion && !refreshedThisPoll) {
            await useContextStore().refreshAfterActivityDataChanged();
          }
          waitingForRefreshCompletion = false;
          if (pollIndex + 1 < maxPolls) {
            await wait(idlePollIntervalMs);
          }
        }
      } finally {
        this.isWatchingStartupRefresh = false;
      }
    },
  },
});

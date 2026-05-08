import { defineStore } from "pinia";
import type { HealthDetailsPayload } from "@/models/health.model";
import { requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

const DEFAULT_POLL_INTERVAL_MS = 2000;
const DEFAULT_MAX_POLLS = 900;

type StartupRefreshWatchOptions = {
  pollIntervalMs?: number;
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

export const useBackendRefreshStore = defineStore("backendRefresh", {
  state: () => ({
    isWatchingStartupRefresh: false,
    observedStartupRefresh: false,
    lastActivityCount: null as number | null,
    error: null as string | null,
  }),
  actions: {
    async watchStartupActivityRefresh(options: StartupRefreshWatchOptions = {}) {
      if (this.isWatchingStartupRefresh) {
        return;
      }

      const pollIntervalMs = options.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;
      const maxPolls = options.maxPolls ?? DEFAULT_MAX_POLLS;

      this.isWatchingStartupRefresh = true;
      this.observedStartupRefresh = false;
      this.error = null;

      try {
        for (let pollIndex = 0; pollIndex < maxPolls; pollIndex += 1) {
          const health = await requestJson<HealthDetailsPayload>("/api/health/details", {
            method: "GET",
            headers: {
              Accept: "application/json",
            },
          });
          const currentActivityCount = activityCount(health);
          const activityCountChanged =
            this.lastActivityCount !== null &&
            currentActivityCount !== null &&
            this.lastActivityCount !== currentActivityCount;

          if (currentActivityCount !== null) {
            this.lastActivityCount = currentActivityCount;
          }

          let refreshedThisPoll = false;
          if (activityCountChanged) {
            await useContextStore().refreshAfterActivityDataChanged();
            refreshedThisPoll = true;
          }

          if (isBackgroundRefreshInProgress(health)) {
            this.observedStartupRefresh = true;
            await wait(pollIntervalMs);
            continue;
          }

          if (this.observedStartupRefresh && !refreshedThisPoll) {
            await useContextStore().refreshAfterActivityDataChanged();
          }
          return;
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Unable to watch backend activity refresh.";
      } finally {
        this.isWatchingStartupRefresh = false;
      }
    },
  },
});

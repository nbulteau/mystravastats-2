import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useBackendRefreshStore } from "@/stores/backend-refresh";
import { useContextStore } from "@/stores/context";
import { requestJson } from "@/stores/api";

vi.mock("@/stores/api", () => ({
  requestJson: vi.fn(),
}));

describe("backend refresh store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("refreshes activity-derived data after startup background refresh completes", async () => {
    vi.mocked(requestJson)
      .mockResolvedValueOnce({
        activities: 10,
        refresh: {
          backgroundInProgress: true,
        },
      })
      .mockResolvedValueOnce({
        activities: 12,
        refresh: {
          backgroundInProgress: false,
        },
      });
    const contextStore = useContextStore();
    const refreshSpy = vi.spyOn(contextStore, "refreshAfterActivityDataChanged").mockResolvedValue();
    const store = useBackendRefreshStore();

    await store.watchStartupActivityRefresh({ pollIntervalMs: 0, maxPolls: 2 });

    expect(requestJson).toHaveBeenCalledTimes(2);
    expect(refreshSpy).toHaveBeenCalledTimes(1);
    expect(store.observedStartupRefresh).toBe(true);
    expect(store.lastActivityCount).toBe(12);
    expect(store.isWatchingStartupRefresh).toBe(false);
  });

  it("does not force a reload when no startup refresh is running", async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      activities: 10,
      refresh: {
        backgroundInProgress: false,
      },
    });
    const contextStore = useContextStore();
    const refreshSpy = vi.spyOn(contextStore, "refreshAfterActivityDataChanged").mockResolvedValue();
    const store = useBackendRefreshStore();

    await store.watchStartupActivityRefresh({ pollIntervalMs: 0, maxPolls: 1 });

    expect(requestJson).toHaveBeenCalledTimes(1);
    expect(refreshSpy).not.toHaveBeenCalled();
    expect(store.observedStartupRefresh).toBe(false);
    expect(store.lastActivityCount).toBe(10);
    expect(store.isWatchingStartupRefresh).toBe(false);
  });

  it("refreshes derived caches when the source dataset changes without changing the activity count", async () => {
    vi.mocked(requestJson)
      .mockResolvedValueOnce({
        provider: "strava",
        activities: 10,
        composite: {
          activeProviders: ["strava"],
          sources: [{ provider: "strava", activities: 10 }],
        },
        refresh: {
          backgroundInProgress: false,
        },
      })
      .mockResolvedValueOnce({
        provider: "composite",
        activities: 10,
        composite: {
          activeProviders: ["strava", "fit"],
          sources: [
            { provider: "strava", activities: 8 },
            { provider: "fit", activities: 2 },
          ],
        },
        refresh: {
          backgroundInProgress: false,
        },
      });
    const contextStore = useContextStore();
    const refreshSpy = vi.spyOn(contextStore, "refreshAfterActivityDataChanged").mockResolvedValue();
    const store = useBackendRefreshStore();

    await store.watchStartupActivityRefresh({ pollIntervalMs: 0, maxPolls: 2 });

    expect(requestJson).toHaveBeenCalledTimes(2);
    expect(refreshSpy).toHaveBeenCalledTimes(1);
    expect(store.lastActivityCount).toBe(10);
    expect(store.isWatchingStartupRefresh).toBe(false);
  });

  it("keeps watching after a transient backend error", async () => {
    vi.mocked(requestJson)
      .mockRejectedValueOnce(new Error("backend unavailable"))
      .mockResolvedValueOnce({
        provider: "composite",
        activities: 12,
        refresh: {
          backgroundInProgress: false,
        },
      });
    const store = useBackendRefreshStore();

    await store.watchStartupActivityRefresh({ pollIntervalMs: 0, maxPolls: 2 });

    expect(requestJson).toHaveBeenCalledTimes(2);
    expect(store.lastActivityCount).toBe(12);
    expect(store.error).toBeNull();
  });

  it("stops polling after a bounded number of unchanged idle responses", async () => {
    vi.mocked(requestJson).mockResolvedValue({ activities: 10, refresh: { backgroundInProgress: false } });
    const store = useBackendRefreshStore();

    await store.watchStartupActivityRefresh({ pollIntervalMs: 0, idlePollLimit: 2 });

    expect(requestJson).toHaveBeenCalledTimes(2);
    expect(store.isWatchingStartupRefresh).toBe(false);
  });

  it("does not poll while the document is hidden", async () => {
    const documentTarget = new EventTarget() as EventTarget & { hidden: boolean };
    documentTarget.hidden = true;
    vi.stubGlobal("document", documentTarget);
    vi.mocked(requestJson).mockResolvedValue({ activities: 10, refresh: { backgroundInProgress: false } });
    const store = useBackendRefreshStore();

    const watching = store.watchStartupActivityRefresh({ pollIntervalMs: 0, maxPolls: 1 });
    await Promise.resolve();
    expect(requestJson).not.toHaveBeenCalled();
    documentTarget.hidden = false;
    documentTarget.dispatchEvent(new Event("visibilitychange"));
    await watching;

    expect(requestJson).toHaveBeenCalledTimes(1);
    vi.unstubAllGlobals();
  });
});

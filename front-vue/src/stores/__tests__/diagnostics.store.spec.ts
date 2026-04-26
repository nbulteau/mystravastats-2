import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { requestJson } from "@/stores/api";

vi.mock("@/stores/api", () => ({
  requestJson: vi.fn(),
}));

describe("diagnostics store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("loads health details from the backend", async () => {
    const payload = {
      provider: "strava",
      activities: 42,
      routing: {
        status: "up",
      },
    };
    vi.mocked(requestJson).mockResolvedValue(payload);
    const store = useDiagnosticsStore();

    await store.refreshDiagnostics();

    expect(requestJson).toHaveBeenCalledWith("/api/health/details", {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
    });
    expect(store.health).toEqual(payload);
    expect(store.error).toBeNull();
    expect(store.lastLoadedAt).not.toBeNull();
  });

  it("keeps the error readable when loading fails", async () => {
    vi.mocked(requestJson).mockRejectedValue(new Error("backend offline"));
    const store = useDiagnosticsStore();

    await store.refreshDiagnostics();

    expect(store.health).toBeNull();
    expect(store.error).toBe("backend offline");
    expect(store.isLoading).toBe(false);
  });

  it("previews a source mode", async () => {
    const payload = {
      mode: "GPX",
      activeMode: "STRAVA",
      path: "/data/gpx",
      configKey: "GPX_FILES_PATH",
      supported: true,
      active: false,
      configured: false,
      readable: true,
      validStructure: true,
      restartNeeded: true,
      activationCommand: "env -u FIT_FILES_PATH GPX_FILES_PATH='/data/gpx' ./mystravastats -port '8080'",
      fileCount: 2,
      validFileCount: 2,
      invalidFileCount: 0,
      activityCount: 2,
      years: [{ year: "2026", fileCount: 2, validFileCount: 2, activityCount: 2 }],
      missingFields: ["power"],
      environment: [{ key: "GPX_FILES_PATH", value: "/data/gpx", required: true }],
      errors: [],
      recommendations: ["Set GPX_FILES_PATH=/data/gpx to use this source."],
    };
    vi.mocked(requestJson).mockResolvedValue(payload);
    const store = useDiagnosticsStore();

    const preview = await store.previewSourceMode({ mode: "GPX", path: "/data/gpx" });

    expect(requestJson).toHaveBeenCalledWith("/api/source-modes/preview", {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ mode: "GPX", path: "/data/gpx" }),
    });
    expect(preview).toEqual(payload);
    expect(store.sourceModePreview).toEqual(payload);
    expect(store.sourceModePreviewError).toBeNull();
  });

  it("normalizes nullable preview lists from Go JSON", async () => {
    const payload = {
      mode: "FIT",
      path: "/data/fit",
      configKey: "FIT_FILES_PATH",
      supported: true,
      configured: false,
      readable: true,
      validStructure: true,
      restartNeeded: true,
      fileCount: 95,
      validFileCount: 95,
      invalidFileCount: 0,
      activityCount: 95,
      years: null,
      missingFields: null,
      environment: null,
      errors: null,
      recommendations: null,
    };
    vi.mocked(requestJson).mockResolvedValue(payload);
    const store = useDiagnosticsStore();

    const preview = await store.previewSourceMode({ mode: "FIT", path: "/data/fit" });

    expect(preview.years).toEqual([]);
    expect(preview.missingFields).toEqual([]);
    expect(preview.environment).toEqual([]);
    expect(preview.errors).toEqual([]);
    expect(preview.recommendations).toEqual([]);
  });
});

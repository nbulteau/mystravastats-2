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
});

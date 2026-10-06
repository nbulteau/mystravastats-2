import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useTrainingLoadStore } from "./training-load";
import { requestJson } from "@/services/http-client";
import type { TrainingReport } from "@/generated/api-contract";
vi.mock("@/services/http-client", () => ({ requestJson: vi.fn() }));
vi.mock("@/stores/context", () => ({ useContextStore: () => ({currentActivityType:"Ride"}) }));
const report = (method: string): TrainingReport => ({method,maxGapSeconds:10,undatedActivities:0,activities:[],weeks:[]});
describe("training load requests", () => {
  beforeEach(() => { setActivePinia(createPinia()); vi.resetAllMocks(); });
  it("keeps the latest selection when requests finish out of order", async () => {
    let first!: (value: TrainingReport) => void;
    vi.mocked(requestJson).mockImplementationOnce(() => new Promise(resolve => { first=resolve as typeof first; }));
    const store=useTrainingLoadStore();store.week="2026-10-05";
    const old=store.load();
    vi.mocked(requestJson).mockResolvedValueOnce(report("new"));
    await store.changeWeek("2026-10-12");first(report("old"));await old;
    expect(store.report?.method).toBe("new");expect(store.loading).toBe(false);
    expect(vi.mocked(requestJson).mock.calls[1]?.[0]).toContain("week=2026-10-12");
  });
  it("removes stale totals on failure and supports retry", async () => {
    const store=useTrainingLoadStore();store.report=report("old");
    vi.mocked(requestJson).mockRejectedValueOnce(new Error("Offline"));await store.load();
    expect(store.report).toBeNull();expect(store.error).toBe("Offline");expect(store.loading).toBe(false);
    vi.mocked(requestJson).mockResolvedValueOnce(report("retried"));await store.load();
    expect(store.error).toBe("");expect(store.report?.method).toBe("retried");
  });
});

import { describe, expect, it } from "vitest";
import { largestMeasuredContribution, loadNumber, shiftWeek, volumeComparison } from "./training-load-presentation";
import type { TrainingLoadActivity, TrainingLoadPeriod } from "@/generated/api-contract";
describe("weekly report explanations", () => {
  it("never formats missing load as a zero", () => { expect(loadNumber(null)).toBe("Unavailable");expect(loadNumber(0)).toBe("0.0"); });
  it("moves across year and daylight-saving boundaries by calendar weeks", () => {
    expect(shiftWeek("2027-01-01",-1)).toBe("2026-12-25");expect(shiftWeek("2026-03-30",-1)).toBe("2026-03-23");
  });
  it("does not make growth claims against an unrecorded week", () => {
    expect(volumeComparison({movingSeconds:3600} as TrainingLoadPeriod,{activityCount:0} as TrainingLoadPeriod)).toContain("No activities recorded");
  });
  it("keeps estimates out of the largest measured contribution", () => {
    const rows=[{source:"estimated",load:200},{source:"measured",load:null},{source:"measured",load:50,activityId:3}] as TrainingLoadActivity[];
    expect(largestMeasuredContribution(rows)?.activityId).toBe(3);
  });
});

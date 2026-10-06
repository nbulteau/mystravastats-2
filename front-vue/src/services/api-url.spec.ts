import { describe, expect, it } from "vitest";
import { apiUrl } from "@/services/api-url";

describe("apiUrl", () => {
  it("uses the canonical operation path and encodes path parameters", () => {
    expect(apiUrl("editGeneratedRoute", { path: { routeId: "route/42" } }))
      .toBe("/api/routes/route%2F42/edit");
  });

  it("serializes defined query parameters", () => {
    expect(apiUrl("listActivities", {
      query: { activityType: "Hike Walk", year: 2026, raw: false, ignored: undefined },
    })).toBe("/api/activities?activityType=Hike+Walk&year=2026&raw=false");
  });

  it("rejects a missing path parameter", () => {
    expect(() => apiUrl("getActivity")).toThrow(/activityId/);
  });

  it("preserves correction URLs with the shared OpenAPI path template", () => {
    expect(apiUrl("applyDataQualityCorrection", { path: { id: "issue-1" } }))
      .toBe("/api/data-quality/corrections/issue-1");
    expect(apiUrl("revertDataQualityCorrection", { path: { id: "correction-1" } }))
      .toBe("/api/data-quality/corrections/correction-1");
  });
});

import { describe, expect, it } from "vitest";
import { getActivityTypeColor } from "./mapTrackColors";

describe("map track colors", () => {
  it("uses saturated, distinct colors for gravel and virtual rides", () => {
    expect(getActivityTypeColor("GravelRide")).toBe("#d81b60");
    expect(getActivityTypeColor("VirtualRide")).toBe("#7b1fa2");
    expect(getActivityTypeColor("GravelRide")).not.toBe(getActivityTypeColor("VirtualRide"));
  });

  it("keeps a neutral fallback for unknown activity types", () => {
    expect(getActivityTypeColor("Unknown")).toBe("#546e7a");
  });
});

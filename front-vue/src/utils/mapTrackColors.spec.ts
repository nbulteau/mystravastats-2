import { describe, expect, it } from "vitest";
import {
  getActivityTypeColor,
  MAP_TRACK_HALO_COLOR,
  MAP_TRACK_HALO_WEIGHT_DELTA,
} from "./mapTrackColors";

describe("map track colors", () => {
  const gpsActivityTypes = [
    "Ride", "Commute", "GravelRide", "MountainBikeRide", "VirtualRide",
    "Run", "TrailRun", "Walk", "Hike", "AlpineSki", "InlineSkate", "Swim", "Rowing",
  ];

  it("uses saturated, distinct colors for gravel and virtual rides", () => {
    expect(getActivityTypeColor("GravelRide")).toBe("#d81b60");
    expect(getActivityTypeColor("VirtualRide")).toBe("#7b1fa2");
    expect(getActivityTypeColor("GravelRide")).not.toBe(getActivityTypeColor("VirtualRide"));
  });

  it("does not use brown for any GPS activity palette entry", () => {
    expect(getActivityTypeColor("Commute")).toBe("#2457c5");
    expect(getActivityTypeColor("MountainBikeRide")).toBe("#16823b");
    expect(getActivityTypeColor("Run")).toBe("#e31a1c");
    expect(getActivityTypeColor("TrailRun")).toBe("#9c27b0");
    expect(getActivityTypeColor("Walk")).toBe("#00838f");
    expect(getActivityTypeColor("Hike")).toBe("#217a3c");
  });

  it("defines a visible casing around tracks", () => {
    expect(MAP_TRACK_HALO_COLOR).toBe("#ffffff");
    expect(MAP_TRACK_HALO_WEIGHT_DELTA).toBeGreaterThanOrEqual(2);
  });

  it("keeps every GPS track color distinguishable from its white casing", () => {
    gpsActivityTypes.forEach((activityType) => {
      expect(contrastRatio(getActivityTypeColor(activityType), MAP_TRACK_HALO_COLOR)).toBeGreaterThanOrEqual(3);
    });
  });

  it("keeps a neutral fallback for unknown activity types", () => {
    expect(getActivityTypeColor("Unknown")).toBe("#546e7a");
  });
});

function contrastRatio(first: string, second: string): number {
  const firstLuminance = relativeLuminance(first);
  const secondLuminance = relativeLuminance(second);
  const lighter = Math.max(firstLuminance, secondLuminance);
  const darker = Math.min(firstLuminance, secondLuminance);
  return (lighter + 0.05) / (darker + 0.05);
}

function relativeLuminance(hex: string): number {
  const value = Number.parseInt(hex.slice(1), 16);
  const channels = [value >> 16, (value >> 8) & 0xff, value & 0xff].map((channel) => {
    const normalized = channel / 255;
    return normalized <= 0.04045
      ? normalized / 12.92
      : ((normalized + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
}

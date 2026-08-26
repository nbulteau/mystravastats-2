import { describe, expect, it } from "vitest";
import { buildAnnualRecapSvg } from "@/utils/annual-recap";

const metrics = {
  activities: 142,
  activeDays: 96,
  distanceKm: 4210.4,
  elevationM: 53200,
  movingTimeSeconds: 540000,
  longestActivityKm: 184.6,
  longestActivityDate: "2026-07-14T08:00:00Z",
};

describe("annual recap", () => {
  it("builds the selected social format with escaped personal data", () => {
    const svg = buildAnnualRecapSvg({
      year: "2026",
      athleteName: "Nico & <friends>",
      activityLabel: "Ride & Run",
      metrics,
      theme: "dark",
      format: "story",
      includeMap: false,
    });

    expect(svg).toContain('width="1080" height="1920"');
    expect(svg).toContain("Nico &amp; &lt;friends&gt;");
    expect(svg).toContain("Ride &amp; Run");
    expect(svg).toContain("Map hidden for privacy");
    expect(svg).not.toContain("Nico & <friends>");
  });

  it("draws a normalized activity fingerprint without coordinate labels", () => {
    const svg = buildAnnualRecapSvg({
      year: "2026",
      activityLabel: "Cycling",
      metrics,
      theme: "light",
      format: "portrait",
      includeMap: true,
      tracks: [{
        activityId: 1,
        activityName: "Home ride",
        activityDate: "2026-05-03",
        activityType: "Ride",
        distanceKm: 50,
        elevationGainM: 800,
        coordinates: [[45, 6], [45.1, 6.2], [45.2, 6.1]],
      }],
    });

    expect(svg).toContain("ACTIVITY FINGERPRINT · 1 TRACES");
    expect(svg).toContain("<path d=");
    expect(svg).not.toContain("Home ride");
    expect(svg).not.toContain("45.1");
  });
});

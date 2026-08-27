import { describe, expect, it } from "vitest";
import {
  ANNUAL_RECAP_PAGES,
  buildAnnualRecapHighlights,
  buildAnnualRecapSvg,
} from "@/utils/annual-recap";

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

  it("builds every Version 2 carousel card", () => {
    const previousMetrics = {
      ...metrics,
      activities: 100,
      activeDays: 80,
      distanceKm: 3500,
      elevationM: 45000,
    };

    const cards = ANNUAL_RECAP_PAGES.map((page) => buildAnnualRecapSvg({
      year: "2026",
      athleteName: "Nico",
      activityLabel: "Cycling",
      metrics,
      previousMetrics,
      consistencyPercent: 72.5,
      previousConsistencyPercent: 65,
      highlights: ["A strong & consistent year"],
      theme: "light",
      format: "portrait",
      includeMap: false,
      page: page.id,
    }));

    expect(cards).toHaveLength(5);
    expect(cards[1]).toContain("How 2026 compares");
    expect(cards[1]).toContain("2025: 3,500");
    expect(cards[2]).toContain("72.5%");
    expect(cards[3]).toContain("Every trace tells part of the story");
    expect(cards[4]).toContain("A strong &amp; consistent year");
  });

  it("creates meaningful highlights with year-over-year comparison", () => {
    const highlights = buildAnnualRecapHighlights(metrics, {
      ...metrics,
      distanceKm: metrics.distanceKm / 1.2,
    }, 70);

    expect(highlights[0]).toBe("20% more distance than the previous year");
    expect(highlights).toContain("Climbed the equivalent of 6.0 Everests");
    expect(highlights).toContain("70% consistency across the year");
  });

  it("labels in-progress year comparisons as year to date", () => {
    const highlights = buildAnnualRecapHighlights(metrics, {
      ...metrics,
      distanceKm: metrics.distanceKm * 2,
    }, 70, true);
    const progress = buildAnnualRecapSvg({
      year: "2026",
      activityLabel: "Cycling",
      metrics,
      previousMetrics: { ...metrics, distanceKm: metrics.distanceKm * 2 },
      consistencyPercent: 70,
      theme: "light",
      format: "portrait",
      includeMap: false,
      page: "progress",
      yearToDate: true,
    });

    expect(highlights[0]).toBe("Distance so far is 50% below last year's total");
    expect(progress).toContain("2026 to date compared with the full 2025 total");
  });
});

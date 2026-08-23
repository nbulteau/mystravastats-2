import { describe, expect, it } from "vitest";
import {
  buildClimbGradientSegments,
  buildClimbPosterSvg,
  posterDesignMaxClimbs,
  type ClimbPosterEntry,
} from "@/utils/climb-poster";

const climb: ClimbPosterEntry = {
  label: "Col <Test> from Valley & Lake",
  category: "1",
  details: {
    name: "Col <Test>",
    country: "FR",
    massif: "Alpes",
    summitAltitude: 1850,
    minimumAltitude: 740,
    lengthKm: 13.8,
    totalAscent: 1110,
    difficulty: 800,
    averageGradient: 8.1,
    maximumGradient: 12.6,
    profile: [
      { distanceKm: 0, elevation: 740 },
      { distanceKm: 6.9, elevation: 1210 },
      { distanceKm: 13.8, elevation: 1850 },
    ],
    ascentCount: 2,
    bestAscent: { activityId: 1, date: "2025-06-12T08:00:00Z", durationSeconds: 3588 },
  },
};

describe("climb poster", () => {
  it("generates all three printable designs with escaped climb content", () => {
    for (const design of ["altitude", "topo", "collection"] as const) {
      const svg = buildClimbPosterSvg({
        design,
        climbs: [climb],
        yearLabel: "All time",
        athleteName: "Nicolas & Co",
      });

      expect(svg).toContain('<svg xmlns="http://www.w3.org/2000/svg"');
      expect(svg).toContain("&lt;TEST&gt;");
      expect(svg).toContain("COL &lt;TEST&gt; DEPUIS VALLEY &amp; LAKE");
      expect(svg).toContain("NICOLAS &amp; CO");
      expect(svg).toContain("12,6");
      expect(svg).toMatch(/ALT[ .]MAX/);
      expect(svg).toContain("DIFFICULTÉ");
      expect(svg).toContain("PTS");
      expect(svg).toContain("12.06.25");
      expect(svg).toContain("59:48");
      expect(svg).toContain("1 COL · 2 ASCENSIONS");
      expect(svg).not.toContain("M D+");
      expect(svg).toContain("data-profile-segment=");
      expect(svg).toContain("data-gradient=");
      expect(svg).not.toMatch(/class="(?:climb-name|topo-name|vertical-name)"[^>]*lengthAdjust=/);
      expect(svg).not.toContain("Col <Test>");
    }
  });

  it("calculates readable distance segments and their average gradients", () => {
    const segments = buildClimbGradientSegments([
      { distanceKm: 0, elevation: 100 },
      { distanceKm: 1, elevation: 150 },
      { distanceKm: 2, elevation: 130 },
      { distanceKm: 3, elevation: 250 },
    ], 3);

    expect(segments).toHaveLength(3);
    expect(segments.map((segment) => segment.averageGradient)).toEqual([5, -2, 12]);
    expect(segments.at(-1)).toMatchObject({ startKm: 2, endKm: 3, startElevation: 130, endElevation: 250 });
  });

  it("keeps altitude annotations far enough apart to remain readable", () => {
    const crowdedProfile = {
      ...climb,
      details: {
        ...climb.details,
        lengthKm: 18.6,
        profile: Array.from({ length: 20 }, (_, index) => ({
          distanceKm: Math.min(index, 18.6),
          elevation: 740 + index * 55,
        })),
      },
    };
    crowdedProfile.details.profile.push({ distanceKm: 18.6, elevation: 1850 });

    const svg = buildClimbPosterSvg({
      design: "altitude",
      climbs: [crowdedProfile],
      yearLabel: "2026",
    });
    const labelPositions = [...svg.matchAll(/data-profile-altitude-label="[^"]+" x="([\d.]+)"/g)]
      .map((match) => Number(match[1]));

    expect(labelPositions.length).toBeGreaterThanOrEqual(2);
    for (let index = 1; index < labelPositions.length; index += 1) {
      expect((labelPositions[index] ?? 0) - (labelPositions[index - 1] ?? 0)).toBeGreaterThanOrEqual(52);
    }
  });

  it("raises the starting altitude above the first gradient label", () => {
    const svg = buildClimbPosterSvg({
      design: "altitude",
      climbs: [climb],
      yearLabel: "2026",
    });
    const profileBase = svg.match(/<line[^>]*y1="([\d.]+)"[^>]*class="profile-base"/);
    const startingAltitude = svg.match(/data-profile-altitude-label="740"[^>]*y="([\d.]+)"/);

    expect(profileBase).not.toBeNull();
    expect(startingAltitude).not.toBeNull();
    expect(Number(profileBase?.[1]) - Number(startingAltitude?.[1])).toBeGreaterThanOrEqual(18);
  });

  it("enforces the layout capacity", () => {
    expect(posterDesignMaxClimbs("altitude")).toBe(50);
    expect(posterDesignMaxClimbs("topo")).toBe(50);
    expect(posterDesignMaxClimbs("collection")).toBe(50);

    const svg = buildClimbPosterSvg({
      design: "collection",
      climbs: [
        ...Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Climb ${index + 1}` })),
        { ...climb, label: "Should not render" },
      ],
      yearLabel: "2026",
    });
    expect(svg).not.toContain("Should not render");
  });

  it("switches to dense multi-column layouts when needed", () => {
    const altitude = buildClimbPosterSvg({
      design: "altitude",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Altitude ${index + 1}` })),
      yearLabel: "All time",
    });
    const topo = buildClimbPosterSvg({
      design: "topo",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Topo ${index + 1}` })),
      yearLabel: "All time",
    });
    const collection = buildClimbPosterSvg({
      design: "collection",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Collection ${index + 1}` })),
      yearLabel: "All time",
    });

    expect(altitude).toContain("dense-divider");
    expect(topo).toContain("dense-topo-card");
    expect(collection).toContain("dense-collection-card");
    expect(altitude).toContain(".dense-name{font:500");
    expect(topo).toContain(".dense-topo-name{font:600");
    expect(collection).toContain(".dense-collection-name{font:600");
    expect(topo).toContain("TECHNICAL");
    expect(collection).toContain("FR · ALPES");
    expect(altitude).not.toMatch(/class="dense-name"[^>]*lengthAdjust=/);
    expect(topo).not.toMatch(/class="dense-topo-name"[^>]*lengthAdjust=/);
    expect(collection).not.toMatch(/class="dense-collection-name"[^>]*lengthAdjust=/);
    expect(altitude).toContain('width="2000" height="3000"');
    expect(altitude).toContain('data-grid-column="4" data-grid-row="9"');
  });

  it("places five climbs on each dense poster row", () => {
    for (const design of ["altitude", "topo", "collection"] as const) {
      const svg = buildClimbPosterSvg({
        design,
        climbs: Array.from({ length: 6 }, (_, index) => ({ ...climb, label: `${design} ${index + 1}` })),
        yearLabel: "2026",
      });

      expect(svg).toContain('data-grid-column="0" data-grid-row="0"');
      expect(svg).toContain('data-grid-column="4" data-grid-row="0"');
      expect(svg).toContain('data-grid-column="0" data-grid-row="1"');
    }
  });

  it("expands dense Altitude and Topo profiles into the available tile height", () => {
    for (const design of ["altitude", "topo"] as const) {
      const svg = buildClimbPosterSvg({
        design,
        climbs: Array.from({ length: 20 }, (_, index) => ({ ...climb, label: `${design} ${index + 1}` })),
        yearLabel: "2026",
      });
      const profileHeight = svg.match(/data-grid-column="0" data-grid-row="0" data-profile-height="([\d.]+)"/);

      expect(profileHeight).not.toBeNull();
      expect(Number(profileHeight?.[1])).toBeGreaterThanOrEqual(175);
    }
  });

  it("uses the full Collection tile height without overlapping the profile labels", () => {
    const svg = buildClimbPosterSvg({
      design: "collection",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Collection ${index + 1}` })),
      yearLabel: "All time",
    });
    const firstTile = svg.match(/<g data-grid-column="0" data-grid-row="0" data-profile-bottom-y="([\d.]+)" data-metrics-y="([\d.]+)" data-ascent-y="([\d.]+)" data-tile-bottom-y="([\d.]+)">/);

    expect(firstTile).not.toBeNull();
    expect(Number(firstTile?.[2]) - Number(firstTile?.[1])).toBeGreaterThanOrEqual(16);
    expect(Number(firstTile?.[4]) - Number(firstTile?.[3])).toBe(8);
    expect(svg).toContain("13,8 KM · +1 110 M · ALT MAX 1 850 M · DIFFICULTÉ 800 PTS");
    expect(svg).toContain("2 ASCENTS");
    expect(svg).not.toContain("2×");
  });
});

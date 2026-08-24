import { describe, expect, it } from "vitest";
import {
  buildClimbGradientSegments,
  buildDetailedClimbProfileSvg,
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
    summitCoordinate: { latitude: 45.2, longitude: 6.2 },
    startCoordinate: { latitude: 45.1, longitude: 6.1 },
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
    for (const design of ["alpine-index", "massif-atlas", "profile-wall"] as const) {
      const svg = buildClimbPosterSvg({
        design,
        climbs: [climb],
        yearLabel: "All time",
        athleteName: "Nicolas & Co",
      });

      expect(svg).toContain('<svg xmlns="http://www.w3.org/2000/svg"');
      expect(svg).toContain("&lt;TEST&gt;");
      expect(svg).toContain("COL &lt;TEST&gt;");
      expect(svg).toContain("VALLEY &amp; LAKE");
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
      expect(svg).not.toMatch(/class="(?:alpine-name|atlas-name|wall-name)"[^>]*lengthAdjust=/);
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

  it("uses the poster profile engine for the detailed kilometre view", () => {
    const svg = buildDetailedClimbProfileSvg(climb.details);
    const segments = [...svg.matchAll(/data-profile-segment=/g)];

    expect(svg).toContain('aria-label="Profil kilométrique de l\'ascension"');
    expect(segments).toHaveLength(14);
    expect(svg).toContain('data-profile-altitude-label="740"');
    expect(svg).toContain('data-profile-altitude-label="1 850"');
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
      design: "alpine-index",
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
      design: "alpine-index",
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
    expect(posterDesignMaxClimbs("alpine-index")).toBe(50);
    expect(posterDesignMaxClimbs("massif-atlas")).toBe(50);
    expect(posterDesignMaxClimbs("profile-wall")).toBe(50);

    const svg = buildClimbPosterSvg({
      design: "profile-wall",
      climbs: [
        ...Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Climb ${index + 1}` })),
        { ...climb, label: "Should not render" },
      ],
      yearLabel: "2026",
    });
    expect(svg).not.toContain("Should not render");
  });

  it("switches to dense multi-column layouts when needed", () => {
    const alpineIndex = buildClimbPosterSvg({
      design: "alpine-index",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Alpine ${index + 1}` })),
      yearLabel: "All time",
    });
    const massifAtlas = buildClimbPosterSvg({
      design: "massif-atlas",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Atlas ${index + 1}` })),
      yearLabel: "All time",
    });
    const profileWall = buildClimbPosterSvg({
      design: "profile-wall",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Wall ${index + 1}` })),
      yearLabel: "All time",
    });

    expect(alpineIndex).toContain("alpine-grid-rule");
    expect(massifAtlas).toContain("atlas-entry");
    expect(profileWall).toContain("wall-tile");
    expect(alpineIndex).toContain(".alpine-name{font:600");
    expect(massifAtlas).toContain(".atlas-name{font:650");
    expect(profileWall).toContain(".wall-name{font:650");
    expect(alpineIndex).toContain("ALPINE INDEX");
    expect(massifAtlas).toContain("MASSIF ATLAS");
    expect(profileWall).toContain("PROFILE WALL");
    expect(massifAtlas).toContain("FR · ALPES");
    expect(massifAtlas).toContain("<tspan");
    expect(alpineIndex).not.toMatch(/class="alpine-name"[^>]*lengthAdjust=/);
    expect(massifAtlas).not.toMatch(/class="atlas-name"[^>]*lengthAdjust=/);
    expect(profileWall).not.toMatch(/class="wall-name"[^>]*lengthAdjust=/);
    expect(alpineIndex).toContain('width="2000" height="3000"');
    expect(alpineIndex).toContain('data-grid-column="4" data-grid-row="9"');
  });

  it("places five climbs on each dense poster row", () => {
    for (const design of ["alpine-index", "massif-atlas", "profile-wall"] as const) {
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

  it("expands profile-driven poster profiles into the available tile height", () => {
    for (const design of ["alpine-index", "profile-wall"] as const) {
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

  it("uses the full Profile Wall tile height without forcing profile labels", () => {
    const svg = buildClimbPosterSvg({
      design: "profile-wall",
      climbs: Array.from({ length: 50 }, (_, index) => ({ ...climb, label: `Wall ${index + 1}` })),
      yearLabel: "All time",
    });
    const firstTile = svg.match(/<g data-grid-column="0" data-grid-row="0" data-profile-height="([\d.]+)">/);

    expect(firstTile).not.toBeNull();
    expect(Number(firstTile?.[1])).toBeGreaterThanOrEqual(90);
    expect(svg).toContain("13,8 KM · +1 110 M · 8,1 % AVG · 12,6 % MAX");
    expect(svg).toContain("ALT MAX 1 850 M · DIFFICULTÉ 800 PTS");
    expect(svg).toContain("2 ASCENTS");
    expect(svg).not.toContain("2×");
    expect(svg).not.toContain("data-profile-altitude-label");
  });
});

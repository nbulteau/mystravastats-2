import { describe, expect, it } from "vitest";
import { drawRouteSketchPng, safeSketchFilename } from "@/services/route-sketch-export";

describe("route sketch export", () => {
  it("creates stable download names", () => {
    expect(safeSketchFilename("  My GPS ♥ Art  ")).toBe("my-gps-art");
    expect(safeSketchFilename("***")).toBe("strava-art-sketch");
  });

  it("rejects an incomplete sketch before drawing", () => {
    const canvas = { getContext: () => null } as unknown as HTMLCanvasElement;
    expect(drawRouteSketchPng(canvas, [[48.1, -1.6]], "Sketch")).toBe(false);
  });
});

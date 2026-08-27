const ACTIVITY_TYPE_COLORS: Record<string, string> = {
  Ride: "#fc4c02",
  Commute: "#2457c5",
  GravelRide: "#d81b60",
  MountainBikeRide: "#16823b",
  VirtualRide: "#7b1fa2",
  Run: "#e31a1c",
  TrailRun: "#9c27b0",
  Walk: "#00838f",
  Hike: "#217a3c",
  AlpineSki: "#1565c0",
  InlineSkate: "#00897b",
  Swim: "#0288d1",
  Rowing: "#00796b",
  WeightTraining: "#5d4037",
};

export const MAP_TRACK_HALO_COLOR = "#ffffff";
export const MAP_TRACK_HALO_WEIGHT_DELTA = 2.4;

export type MapTrackEmphasis = "normal" | "dimmed" | "focused";

export type MapTrackStrokeStyle = {
  weight: number;
  opacity: number;
};

export function getMapTrackStrokeStyle(
  baseWeight: number,
  baseOpacity: number,
  emphasis: MapTrackEmphasis,
  halo = false,
): MapTrackStrokeStyle {
  if (emphasis === "dimmed") {
    return {
      weight: halo ? baseWeight + MAP_TRACK_HALO_WEIGHT_DELTA : baseWeight,
      opacity: halo ? 0.04 : Math.max(0.1, baseOpacity * 0.2),
    };
  }
  if (emphasis === "focused") {
    return {
      weight: halo ? baseWeight + MAP_TRACK_HALO_WEIGHT_DELTA + 2 : baseWeight + 1.8,
      opacity: halo ? 0.96 : 1,
    };
  }
  return {
    weight: halo ? baseWeight + MAP_TRACK_HALO_WEIGHT_DELTA : baseWeight,
    opacity: halo ? Math.min(0.82, baseOpacity + 0.16) : baseOpacity,
  };
}

export function getActivityTypeColor(activityType: string): string {
  return ACTIVITY_TYPE_COLORS[activityType] ?? "#546e7a";
}

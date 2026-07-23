export const CYCLING_ACTIVITY_TYPES = ["Ride", "Commute", "GravelRide", "MountainBikeRide", "VirtualRide"] as const;
export const RUNNING_ACTIVITY_TYPES = ["Run", "TrailRun"] as const;
export const HIKING_ACTIVITY_TYPES = ["Hike", "Walk"] as const;
export const OTHER_ACTIVITY_TYPES = ["AlpineSki", "InlineSkate"] as const;

export const ALL_ACTIVITY_TYPES = [
  ...CYCLING_ACTIVITY_TYPES,
  ...RUNNING_ACTIVITY_TYPES,
  ...HIKING_ACTIVITY_TYPES,
  ...OTHER_ACTIVITY_TYPES,
] as const;

export type ActivityTypeName = typeof ALL_ACTIVITY_TYPES[number];

export const ALL_ACTIVITY_TYPE_FILTER = [...ALL_ACTIVITY_TYPES].sort().join("_");
export const DEFAULT_ACTIVITY_TYPE_FILTER = [...CYCLING_ACTIVITY_TYPES].sort().join("_");

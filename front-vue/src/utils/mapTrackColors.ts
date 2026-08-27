const ACTIVITY_TYPE_COLORS: Record<string, string> = {
  Ride: "#fc4c02",
  Commute: "#3949ab",
  GravelRide: "#d81b60",
  MountainBikeRide: "#2e7d32",
  VirtualRide: "#7b1fa2",
  Run: "#f4511e",
  TrailRun: "#8e24aa",
  Walk: "#6d4c41",
  Hike: "#1b5e20",
  AlpineSki: "#1565c0",
  InlineSkate: "#00897b",
  Swim: "#0288d1",
  Rowing: "#00796b",
  WeightTraining: "#5d4037",
};

export function getActivityTypeColor(activityType: string): string {
  return ACTIVITY_TYPE_COLORS[activityType] ?? "#546e7a";
}

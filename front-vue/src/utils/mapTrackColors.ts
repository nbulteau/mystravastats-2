const ACTIVITY_TYPE_COLORS: Record<string, string> = {
  Ride: "#fc4c02",
  Commute: "#3949ab",
  GravelRide: "#8d6e63",
  MountainBikeRide: "#2e7d32",
  VirtualRide: "#6d4c41",
  Run: "#f4511e",
  TrailRun: "#8e24aa",
  Walk: "#6d4c41",
  Hike: "#1b5e20",
  AlpineSki: "#1565c0",
  Swim: "#0288d1",
  Rowing: "#00796b",
  WeightTraining: "#5d4037",
};

export function getActivityTypeColor(activityType: string): string {
  return ACTIVITY_TYPE_COLORS[activityType] ?? "#546e7a";
}

export type MapTrack = {
  activityId: number;
  activityName: string;
  activityDate: string;
  activityType: string;
  distanceKm: number;
  elevationGainM: number;
  coordinates: number[][];
};

export type MapPassageSegment = {
  coordinates: number[][];
  passageCount: number;
  activityCount: number;
  distanceKm: number;
  activityTypeCounts?: Record<string, number>;
};

export type MapPassages = {
  segments: MapPassageSegment[];
  includedActivities: number;
  excludedActivities: number;
  missingStreamActivities: number;
  resolutionMeters: number;
  minPassageCount: number;
  omittedSegments: number;
};

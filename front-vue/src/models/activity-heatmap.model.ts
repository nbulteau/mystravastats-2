export type HeatmapMetricKey = "distanceKm" | "elevationGainM" | "durationSec";

export interface HeatmapActivityDetail {
  id: number;
  name: string;
  type: string;
  distanceKm: number;
  elevationGainM: number;
  durationSec: number;
}

export interface HeatmapDayData {
  distanceKm: number;
  elevationGainM: number;
  durationSec: number;
  activityCount: number;
  activities: HeatmapActivityDetail[];
}

export type ActivityHeatmap = Record<string, Record<string, HeatmapDayData>>;

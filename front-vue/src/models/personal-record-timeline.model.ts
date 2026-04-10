import type { ActivityShort } from "./activity.model";

export interface PersonalRecordTimeline {
  metricKey: string;
  metricLabel: string;
  activityDate: string;
  value: string;
  previousValue?: string;
  improvement?: string;
  activity: ActivityShort;
}

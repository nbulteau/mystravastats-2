import type { Activity } from "./activity.model";
import type { Badge } from "./badge.model";

export interface BadgeCheckResult {
    badge: Badge;
    activities: Activity[];
    nbCheckedActivities: number;
}

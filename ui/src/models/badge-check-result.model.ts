import type { Activity } from "./activity.model";
import type { Badge } from "./badge.model";

export interface BadgeCheckResult {
    badge: Badge;
    activity: Activity
    isCompleted: boolean;
}

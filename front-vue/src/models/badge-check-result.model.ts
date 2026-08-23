import type { Activity } from "./activity.model";
import type { Badge } from "./badge.model";

export interface BadgeCheckResult {
    badge: Badge;
    activities: Activity[];
    nbCheckedActivities: number;
    climbDetails?: ClimbDetails | null;
}

export interface ClimbDetails {
    name: string;
    country: string;
    massif: string;
    sourceUrl?: string | null;
    summitCoordinate: ClimbCoordinate;
    startCoordinate: ClimbCoordinate;
    summitAltitude: number;
    minimumAltitude: number;
    lengthKm: number;
    totalAscent: number;
    difficulty: number;
    averageGradient: number;
    maximumGradient?: number | null;
    profile: ClimbProfilePoint[];
    ascentCount: number;
    bestAscent?: ClimbAscent | null;
    ascents?: ClimbAscent[];
}

export interface ClimbCoordinate {
    latitude: number;
    longitude: number;
}

export interface ClimbProfilePoint {
    distanceKm: number;
    elevation: number;
}

export interface ClimbAscent {
    activityId: number;
    activityName?: string;
    date: string;
    durationSeconds: number;
    vamMetersPerHour?: number | null;
    averageSpeedKph?: number | null;
    averagePowerWatts?: number | null;
    averageHeartRateBpm?: number | null;
}

import type { ActivityShort } from "./activity.model";

export interface HeartRateZoneSettings {
    maxHr?: number | null;
    thresholdHr?: number | null;
    reserveHr?: number | null;
}

export interface ResolvedHeartRateZoneSettings {
    maxHr: number;
    thresholdHr?: number | null;
    reserveHr?: number | null;
    method: "MAX" | "THRESHOLD" | "RESERVE";
    source: "ATHLETE_SETTINGS" | "DERIVED_FROM_DATA";
}

export interface HeartRateZoneDistribution {
    zone: string;
    label: string;
    seconds: number;
    percentage: number;
}

export interface HeartRateZoneActivitySummary {
    activity: ActivityShort;
    activityDate: string;
    totalTrackedSeconds: number;
    easySeconds: number;
    hardSeconds: number;
    easyHardRatio?: number | null;
    zones: HeartRateZoneDistribution[];
}

export interface HeartRateZonePeriodSummary {
    period: string;
    totalTrackedSeconds: number;
    easySeconds: number;
    hardSeconds: number;
    easyHardRatio?: number | null;
    zones: HeartRateZoneDistribution[];
}

export interface HeartRateZoneAnalysis {
    settings: HeartRateZoneSettings;
    resolvedSettings?: ResolvedHeartRateZoneSettings | null;
    hasHeartRateData: boolean;
    totalTrackedSeconds: number;
    easyHardRatio?: number | null;
    zones: HeartRateZoneDistribution[];
    activities: HeartRateZoneActivitySummary[];
    byMonth: HeartRateZonePeriodSummary[];
    byYear: HeartRateZonePeriodSummary[];
}

export function emptyHeartRateZoneAnalysis(): HeartRateZoneAnalysis {
    return {
        settings: {
            maxHr: null,
            thresholdHr: null,
            reserveHr: null,
        },
        resolvedSettings: null,
        hasHeartRateData: false,
        totalTrackedSeconds: 0,
        easyHardRatio: null,
        zones: [],
        activities: [],
        byMonth: [],
        byYear: [],
    };
}

import type {
    HeartRateZoneDistribution,
    HeartRateZoneSettings,
    ResolvedHeartRateZoneSettings,
} from "@/models/heart-rate-zone.model";

const ZONE_CODES = ["Z1", "Z2", "Z3", "Z4", "Z5"];
const ZONE_LABELS = ["Recovery", "Endurance", "Tempo", "Threshold", "VO2 Max"];

export interface HeartRateZoneComputation {
    totalTrackedSeconds: number;
    easySeconds: number;
    hardSeconds: number;
    easyHardRatio?: number | null;
    zones: HeartRateZoneDistribution[];
}

export function resolveHeartRateZoneSettings(
    settings: HeartRateZoneSettings | null | undefined,
    fallbackMaxHr?: number | null,
): ResolvedHeartRateZoneSettings | null {
    const maxHr = normalizeInt(settings?.maxHr ?? null);
    const thresholdHr = normalizeInt(settings?.thresholdHr ?? null);
    const reserveHr = normalizeInt(settings?.reserveHr ?? null);

    if (thresholdHr !== null) {
        return {
            maxHr: maxHr ?? fallbackMaxHr ?? thresholdHr,
            thresholdHr,
            reserveHr,
            method: "THRESHOLD",
            source: "ATHLETE_SETTINGS",
        };
    }

    if (maxHr !== null && reserveHr !== null && reserveHr > 0 && reserveHr < maxHr) {
        return {
            maxHr,
            thresholdHr: null,
            reserveHr,
            method: "RESERVE",
            source: "ATHLETE_SETTINGS",
        };
    }

    if (maxHr !== null) {
        return {
            maxHr,
            thresholdHr: null,
            reserveHr,
            method: "MAX",
            source: "ATHLETE_SETTINGS",
        };
    }

    if (fallbackMaxHr && fallbackMaxHr > 0) {
        return {
            maxHr: fallbackMaxHr,
            thresholdHr: null,
            reserveHr: null,
            method: "MAX",
            source: "DERIVED_FROM_DATA",
        };
    }

    return null;
}

export function computeHeartRateZoneDistribution(
    heartrate: number[] | null | undefined,
    time: number[] | null | undefined,
    resolvedSettings: ResolvedHeartRateZoneSettings | null,
): HeartRateZoneComputation | null {
    if (!resolvedSettings || !heartrate || !time) {
        return null;
    }

    const sampleSize = Math.min(heartrate.length, time.length);
    if (sampleSize < 2) {
        return null;
    }

    const zoneSeconds = [0, 0, 0, 0, 0];
    let totalTrackedSeconds = 0;

    for (let index = 0; index < sampleSize - 1; index += 1) {
        const hr = heartrate[index] ?? 0;
        const delta = (time[index + 1] ?? 0) - (time[index] ?? 0);
        if (hr <= 0 || delta <= 0) {
            continue;
        }

        const zoneIndex = resolveZoneIndex(hr, resolvedSettings);
        zoneSeconds[zoneIndex] += delta;
        totalTrackedSeconds += delta;
    }

    if (totalTrackedSeconds <= 0) {
        return null;
    }

    const easySeconds = zoneSeconds[0] + zoneSeconds[1];
    const hardSeconds = zoneSeconds[3] + zoneSeconds[4];
    const easyHardRatio =
        easySeconds > 0 && hardSeconds > 0
            ? Math.round((easySeconds / hardSeconds) * 100) / 100
            : null;

    return {
        totalTrackedSeconds,
        easySeconds,
        hardSeconds,
        easyHardRatio,
        zones: zoneSeconds.map((seconds, index) => ({
            zone: ZONE_CODES[index],
            label: ZONE_LABELS[index],
            seconds,
            percentage:
                totalTrackedSeconds > 0
                    ? Math.round((seconds / totalTrackedSeconds) * 10000) / 100
                    : 0,
        })),
    };
}

function resolveZoneIndex(hr: number, resolvedSettings: ResolvedHeartRateZoneSettings): number {
    let z1Upper = 0;
    let z2Upper = 0;
    let z3Upper = 0;
    let z4Upper = 0;

    if (resolvedSettings.method === "THRESHOLD") {
        const threshold = resolvedSettings.thresholdHr ?? resolvedSettings.maxHr;
        z1Upper = threshold * 0.81;
        z2Upper = threshold * 0.89;
        z3Upper = threshold * 0.93;
        z4Upper = threshold * 0.99;
    } else if (resolvedSettings.method === "RESERVE") {
        const reserve = resolvedSettings.reserveHr ?? 0;
        const resting = Math.max(resolvedSettings.maxHr - reserve, 35);
        z1Upper = resting + reserve * 0.6;
        z2Upper = resting + reserve * 0.7;
        z3Upper = resting + reserve * 0.8;
        z4Upper = resting + reserve * 0.9;
    } else {
        z1Upper = resolvedSettings.maxHr * 0.6;
        z2Upper = resolvedSettings.maxHr * 0.7;
        z3Upper = resolvedSettings.maxHr * 0.8;
        z4Upper = resolvedSettings.maxHr * 0.9;
    }

    if (hr <= z1Upper) return 0;
    if (hr <= z2Upper) return 1;
    if (hr <= z3Upper) return 2;
    if (hr <= z4Upper) return 3;
    return 4;
}

function normalizeInt(value: number | null | undefined): number | null {
    if (value === null || value === undefined || Number.isNaN(value)) {
        return null;
    }
    const normalized = Math.trunc(value);
    if (normalized <= 0) {
        return null;
    }
    return normalized;
}

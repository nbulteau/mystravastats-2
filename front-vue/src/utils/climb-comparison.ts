import type {
  ClimbAscent,
  ClimbAscentComparisonPoint,
} from "@/models/badge-check-result.model";

export type ClimbComparisonMetric =
  | "elapsedSeconds"
  | "speedKph"
  | "vamMetersPerHour"
  | "powerWatts"
  | "heartRateBpm";

export interface ClimbSectorComparison {
  startKm: number;
  endKm: number;
  sectorSeconds: number | null;
  deltaSeconds: number | null;
}

export function defaultComparedAscentIds(
  ascents: ClimbAscent[],
  bestAscentId: number | null,
): number[] {
  const eligible = ascents.filter((ascent) => (ascent.comparisonPoints?.length ?? 0) >= 2);
  const ids: number[] = [];
  const best = eligible.find((ascent) => ascent.activityId === bestAscentId);
  if (best) ids.push(best.activityId);
  const latest = eligible[0];
  if (latest && !ids.includes(latest.activityId)) ids.push(latest.activityId);
  if (ids.length < 2) {
    const alternative = eligible.find((ascent) => !ids.includes(ascent.activityId));
    if (alternative) ids.push(alternative.activityId);
  }
  return ids;
}

export function comparisonValueAtDistance(
  ascent: ClimbAscent,
  distanceKm: number,
  metric: ClimbComparisonMetric,
): number | null {
  const points = (ascent.comparisonPoints ?? [])
    .filter((point) => Number.isFinite(point.distanceKm))
    .sort((left, right) => left.distanceKm - right.distanceKm);
  if (!points.length || distanceKm < points[0].distanceKm || distanceKm > points[points.length - 1].distanceKm) {
    return null;
  }
  const exact = points.find((point) => point.distanceKm === distanceKm);
  if (exact) return metricValue(exact, metric);
  const rightIndex = points.findIndex((point) => point.distanceKm > distanceKm);
  if (rightIndex <= 0) return null;
  const left = points[rightIndex - 1];
  const right = points[rightIndex];
  const leftValue = metricValue(left, metric);
  const rightValue = metricValue(right, metric);
  if (leftValue == null || rightValue == null || right.distanceKm <= left.distanceKm) return null;
  const ratio = (distanceKm - left.distanceKm) / (right.distanceKm - left.distanceKm);
  return leftValue + (rightValue - leftValue) * ratio;
}

export function buildSectorComparisons(
  ascent: ClimbAscent,
  reference: ClimbAscent,
  lengthKm: number,
): ClimbSectorComparison[] {
  const sectorCount = Math.max(1, Math.ceil(lengthKm));
  return Array.from({ length: sectorCount }, (_, index) => {
    const startKm = index;
    const endKm = Math.min(index + 1, lengthKm);
    const ascentStart = comparisonValueAtDistance(ascent, startKm, "elapsedSeconds");
    const ascentEnd = comparisonValueAtDistance(ascent, endKm, "elapsedSeconds");
    const referenceStart = comparisonValueAtDistance(reference, startKm, "elapsedSeconds");
    const referenceEnd = comparisonValueAtDistance(reference, endKm, "elapsedSeconds");
    const sectorSeconds = ascentStart == null || ascentEnd == null ? null : ascentEnd - ascentStart;
    const referenceSeconds = referenceStart == null || referenceEnd == null ? null : referenceEnd - referenceStart;
    return {
      startKm,
      endKm,
      sectorSeconds,
      deltaSeconds: sectorSeconds == null || referenceSeconds == null ? null : sectorSeconds - referenceSeconds,
    };
  });
}

function metricValue(point: ClimbAscentComparisonPoint, metric: ClimbComparisonMetric): number | null {
  const value = point[metric];
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

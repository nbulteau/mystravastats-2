import type { BadgeCheckResult, ClimbAscent, ClimbDetails } from "@/models/badge-check-result.model";
import { buildClimbGradientSegments, type ClimbGradientSegment } from "@/utils/climb-poster";

export interface ClimbSectorInsight extends ClimbGradientSegment {
  elevationGain: number;
}

export function climbVariantTitle(result: BadgeCheckResult): string {
  const label = result.badge.label.trim();
  return label || result.climbDetails?.name || "Climb unavailable";
}

export function climbVariantStart(result: BadgeCheckResult): string {
  const title = climbVariantTitle(result);
  const separator = " from ";
  const index = title.toLocaleLowerCase("en-US").indexOf(separator);
  return index >= 0 ? title.slice(index + separator.length) : "Start unavailable";
}

export function climbAscentHistory(result: BadgeCheckResult): ClimbAscent[] {
  const ascents = result.climbDetails?.ascents
    ?? (result.climbDetails?.bestAscent ? [result.climbDetails.bestAscent] : []);
  return [...ascents].sort((left, right) => {
    if (left.date !== right.date) return right.date.localeCompare(left.date);
    return right.activityId - left.activityId;
  });
}

export function climbKilometerSectors(details: ClimbDetails): ClimbSectorInsight[] {
  return buildClimbGradientSegments(details.profile, Math.max(1, Math.ceil(details.lengthKm)))
    .map((segment) => ({
      ...segment,
      elevationGain: Math.max(0, Math.round(segment.endElevation - segment.startElevation)),
    }));
}

export function hardestClimbSectors(details: ClimbDetails, limit = 5): ClimbSectorInsight[] {
  return [...climbKilometerSectors(details)]
    .sort((left, right) => (
      right.averageGradient - left.averageGradient ||
      left.startKm - right.startKm
    ))
    .slice(0, Math.max(0, limit));
}

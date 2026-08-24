import type { BadgeCheckResult } from "@/models/badge-check-result.model";

export type ClimbSelectionOrder = "hardest" | "longest" | "steepest" | "elevation-gain" | "highest";

export function orderClimbsForPoster(
  climbs: BadgeCheckResult[],
  order: ClimbSelectionOrder,
): BadgeCheckResult[] {
  return [...climbs].sort(comparatorForOrder(order));
}

export function selectTopClimbsForPoster(
  climbs: BadgeCheckResult[],
  order: ClimbSelectionOrder,
  limit: number,
): BadgeCheckResult[] {
  return orderClimbsForPoster(climbs, order).slice(0, Math.max(0, limit));
}

function compareHardest(left: BadgeCheckResult, right: BadgeCheckResult): number {
  return (
    (right.climbDetails?.difficulty ?? 0) - (left.climbDetails?.difficulty ?? 0) ||
    (right.climbDetails?.lengthKm ?? 0) - (left.climbDetails?.lengthKm ?? 0) ||
    left.badge.label.localeCompare(right.badge.label)
  );
}

function compareLongest(left: BadgeCheckResult, right: BadgeCheckResult): number {
  return (
    (right.climbDetails?.lengthKm ?? 0) - (left.climbDetails?.lengthKm ?? 0) ||
    (right.climbDetails?.difficulty ?? 0) - (left.climbDetails?.difficulty ?? 0) ||
    left.badge.label.localeCompare(right.badge.label)
  );
}

function compareSteepest(left: BadgeCheckResult, right: BadgeCheckResult): number {
  return (
    (right.climbDetails?.averageGradient ?? 0) - (left.climbDetails?.averageGradient ?? 0) ||
    (right.climbDetails?.difficulty ?? 0) - (left.climbDetails?.difficulty ?? 0) ||
    left.badge.label.localeCompare(right.badge.label)
  );
}

function compareElevationGain(left: BadgeCheckResult, right: BadgeCheckResult): number {
  return (
    (right.climbDetails?.totalAscent ?? 0) - (left.climbDetails?.totalAscent ?? 0) ||
    (right.climbDetails?.difficulty ?? 0) - (left.climbDetails?.difficulty ?? 0) ||
    left.badge.label.localeCompare(right.badge.label)
  );
}

function compareHighest(left: BadgeCheckResult, right: BadgeCheckResult): number {
  return (
    (right.climbDetails?.summitAltitude ?? 0) - (left.climbDetails?.summitAltitude ?? 0) ||
    (right.climbDetails?.difficulty ?? 0) - (left.climbDetails?.difficulty ?? 0) ||
    left.badge.label.localeCompare(right.badge.label)
  );
}

function comparatorForOrder(order: ClimbSelectionOrder): (left: BadgeCheckResult, right: BadgeCheckResult) => number {
  if (order === "longest") {
    return compareLongest;
  }
  if (order === "steepest") {
    return compareSteepest;
  }
  if (order === "elevation-gain") {
    return compareElevationGain;
  }
  if (order === "highest") {
    return compareHighest;
  }
  return compareHardest;
}

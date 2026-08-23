import type { BadgeCheckResult } from "@/models/badge-check-result.model";

export type ClimbSelectionOrder = "hardest" | "longest";

export function orderClimbsForPoster(
  climbs: BadgeCheckResult[],
  order: ClimbSelectionOrder,
): BadgeCheckResult[] {
  return [...climbs].sort(order === "hardest" ? compareHardest : compareLongest);
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

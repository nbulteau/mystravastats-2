import type { BadgeCheckResult } from "@/models/badge-check-result.model";

function slugPart(value: string): string {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "") || "unknown";
}

function coordinatePart(value: number): string {
  return value.toFixed(5).replace("-", "m").replace(".", "p");
}

export function climbSummitId(result: BadgeCheckResult): string {
  const details = result.climbDetails;
  if (!details) {
    return `climb-${slugPart(result.badge.label)}`;
  }
  if (details.summitId) {
    return details.summitId;
  }
  const { latitude, longitude } = details.summitCoordinate;
  return [
    "climb",
    slugPart(details.country),
    slugPart(details.name),
    coordinatePart(latitude),
    coordinatePart(longitude),
  ].join("-");
}

export function climbVariantId(result: BadgeCheckResult): string {
  if (result.climbDetails?.variantId) {
    return result.climbDetails.variantId;
  }
  return `${climbSummitId(result)}--${slugPart(result.badge.label)}`;
}

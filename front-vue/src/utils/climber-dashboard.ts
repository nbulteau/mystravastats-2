import type { BadgeCheckResult, ClimbAscent } from "@/models/badge-check-result.model";
import { climbSummitId, climbVariantId } from "@/utils/climb-id";

export interface ClimberBreakdownItem {
  key: string;
  label: string;
  climbCount: number;
  ascentCount: number;
  variantIds: string[];
}

export interface ClimberRecord {
  label: string;
  value: number;
  variantIds: string[];
  ascent?: ClimbAscent;
  estimated?: boolean;
}

export interface ClimberDashboardStats {
  climbedVariants: number;
  climbedSummits: number;
  ascentCount: number;
  cumulativeSummitAltitude: number;
  climbElevationGain: number;
  excludedRecordCount: number;
  records: {
    vam: ClimberRecord | null;
    longest: ClimberRecord | null;
    hardest: ClimberRecord | null;
    mostClimbed: ClimberRecord | null;
  };
  years: ClimberBreakdownItem[];
  countries: ClimberBreakdownItem[];
  massifs: ClimberBreakdownItem[];
  categories: ClimberBreakdownItem[];
  altitudeBands: ClimberBreakdownItem[];
}

export function buildClimberDashboardStats(results: BadgeCheckResult[]): ClimberDashboardStats {
  const climbed = results.filter((result) => result.climbDetails && result.climbDetails.ascentCount > 0);
  const ascentCountFor = (result: BadgeCheckResult) => Math.max(
    result.climbDetails?.ascentCount ?? 0,
    result.climbDetails?.ascents?.length ?? 0,
    result.nbCheckedActivities,
  );
  const variantIds = climbed.map(climbVariantId);
  const allAscents = climbed.flatMap((result) => (result.climbDetails?.ascents ?? []).map((ascent) => ({ result, ascent })));
  const rejectedAscents = allAscents.filter(({ ascent }) => !recordQualityAccepted(ascent));
  const vamCandidates = allAscents.filter(({ ascent }) => (
    recordQualityAccepted(ascent) && (ascent.vamMetersPerHour ?? 0) > 0
  ));
  const bestVam = vamCandidates.sort((left, right) => (
    (right.ascent.vamMetersPerHour ?? 0) - (left.ascent.vamMetersPerHour ?? 0)
  ))[0];
  const longest = maxResult(climbed, (result) => result.climbDetails?.lengthKm ?? 0);
  const hardest = maxResult(climbed, (result) => result.climbDetails?.difficulty ?? 0);
  const mostClimbed = maxResult(climbed, ascentCountFor);

  return {
    climbedVariants: climbed.length,
    climbedSummits: new Set(climbed.map(climbSummitId)).size,
    ascentCount: climbed.reduce((total, result) => total + ascentCountFor(result), 0),
    cumulativeSummitAltitude: climbed.reduce((total, result) => (
      total + (result.climbDetails?.summitAltitude ?? 0) * ascentCountFor(result)
    ), 0),
    climbElevationGain: climbed.reduce((total, result) => (
      total + (result.climbDetails?.totalAscent ?? 0) * ascentCountFor(result)
    ), 0),
    excludedRecordCount: rejectedAscents.length,
    records: {
      vam: bestVam ? {
        label: bestVam.result.climbDetails?.name ?? bestVam.result.badge.label,
        value: bestVam.ascent.vamMetersPerHour ?? 0,
        variantIds: [climbVariantId(bestVam.result)],
        ascent: bestVam.ascent,
        estimated: bestVam.ascent.comparisonQuality?.precision !== "high",
      } : null,
      longest: resultRecord(longest, (result) => result.climbDetails?.lengthKm ?? 0),
      hardest: resultRecord(hardest, (result) => result.climbDetails?.difficulty ?? 0),
      mostClimbed: resultRecord(mostClimbed, ascentCountFor),
    },
    years: groupByAscents(climbed, ({ ascent }) => yearOf(ascent.date)),
    countries: groupByClimbs(climbed, (result) => result.climbDetails?.country || "Non renseigné", ascentCountFor),
    massifs: groupByClimbs(climbed, (result) => result.climbDetails?.massif || "Non renseigné", ascentCountFor),
    categories: groupByClimbs(climbed, (result) => `Cat. ${result.badge.category?.trim().toUpperCase() || "?"}`, ascentCountFor),
    altitudeBands: groupByClimbs(climbed, (result) => altitudeBand(result.climbDetails?.summitAltitude ?? 0), ascentCountFor),
  };
}

function recordQualityAccepted(ascent: ClimbAscent): boolean {
  if (ascent.durationSeconds <= 0) return false;
  const rejectedWarnings = new Set([
    "MISSING_STREAM",
    "INCOMPLETE_DISTANCE_STREAM",
    "INCOMPLETE_TIME_STREAM",
    "DISTANCE_DIFFERENCE_OVER_10_PERCENT",
  ]);
  return !(ascent.comparisonQuality?.warnings ?? []).some((warning) => rejectedWarnings.has(warning));
}

function resultRecord(
  result: BadgeCheckResult | undefined,
  value: (result: BadgeCheckResult) => number,
): ClimberRecord | null {
  return result ? {
    label: result.climbDetails?.name ?? result.badge.label,
    value: value(result),
    variantIds: [climbVariantId(result)],
  } : null;
}

function maxResult(
  results: BadgeCheckResult[],
  value: (result: BadgeCheckResult) => number,
): BadgeCheckResult | undefined {
  return [...results].sort((left, right) => value(right) - value(left))[0];
}

function groupByAscents(
  results: BadgeCheckResult[],
  keyOf: (entry: { result: BadgeCheckResult; ascent: ClimbAscent }) => string,
): ClimberBreakdownItem[] {
  const entries = results.flatMap((result) => (result.climbDetails?.ascents ?? []).map((ascent) => ({ result, ascent })));
  const grouped = new Map<string, { variants: Set<string>; ascents: number }>();
  for (const entry of entries) {
    const key = keyOf(entry);
    if (!key) continue;
    const group = grouped.get(key) ?? { variants: new Set<string>(), ascents: 0 };
    group.variants.add(climbVariantId(entry.result));
    group.ascents += 1;
    grouped.set(key, group);
  }
  return Array.from(grouped, ([key, value]) => ({
    key,
    label: key,
    climbCount: value.variants.size,
    ascentCount: value.ascents,
    variantIds: Array.from(value.variants),
  })).sort((left, right) => right.key.localeCompare(left.key, "fr", { numeric: true }));
}

function groupByClimbs(
  results: BadgeCheckResult[],
  keyOf: (result: BadgeCheckResult) => string,
  ascentCountFor: (result: BadgeCheckResult) => number,
): ClimberBreakdownItem[] {
  const grouped = new Map<string, { variants: Set<string>; ascents: number }>();
  for (const result of results) {
    const key = keyOf(result);
    const group = grouped.get(key) ?? { variants: new Set<string>(), ascents: 0 };
    group.variants.add(climbVariantId(result));
    group.ascents += ascentCountFor(result);
    grouped.set(key, group);
  }
  return Array.from(grouped, ([key, value]) => ({
    key,
    label: key,
    climbCount: value.variants.size,
    ascentCount: value.ascents,
    variantIds: Array.from(value.variants),
  })).sort((left, right) => right.ascentCount - left.ascentCount || left.label.localeCompare(right.label, "fr"));
}

function yearOf(date: string): string {
  const match = /^(\d{4})/.exec(date);
  return match?.[1] ?? "Date inconnue";
}

function altitudeBand(altitude: number): string {
  if (altitude >= 2500) return "≥ 2 500 m";
  if (altitude >= 2000) return "2 000–2 499 m";
  if (altitude >= 1500) return "1 500–1 999 m";
  if (altitude >= 1000) return "1 000–1 499 m";
  return "< 1 000 m";
}

import type { BadgeCheckResult, ClimbDetails } from "@/models/badge-check-result.model";

export const CLIMB_MAP_ALL_FILTER = "ALL";

export type ClimbMapStatusFilter = "ALL" | "CLIMBED" | "UNCLIMBED" | "FAVORITE";

export interface ClimbMapFilters {
  country: string;
  massif: string;
  category: string;
  status: ClimbMapStatusFilter;
}

export interface ClimbMapVariant {
  id: string;
  summitId: string;
  label: string;
  category: string;
  climbed: boolean;
  details: ClimbDetails;
  result: BadgeCheckResult;
}

export interface ClimbMapSummit {
  id: string;
  name: string;
  country: string;
  massif: string;
  latitude: number;
  longitude: number;
  summitAltitude: number;
  climbed: boolean;
  ascentCount: number;
  variants: ClimbMapVariant[];
}

export interface ClimbMapCluster {
  id: string;
  latitude: number;
  longitude: number;
  summits: ClimbMapSummit[];
}

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

function validCoordinate(latitude: number, longitude: number): boolean {
  return Number.isFinite(latitude)
    && Number.isFinite(longitude)
    && latitude >= -90
    && latitude <= 90
    && longitude >= -180
    && longitude <= 180;
}

export function climbSummitId(result: BadgeCheckResult): string {
  const details = result.climbDetails;
  if (!details) {
    return `climb-${slugPart(result.badge.label)}`;
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
  return `${climbSummitId(result)}--${slugPart(result.badge.label)}`;
}

export function buildClimbMapSummits(results: BadgeCheckResult[]): ClimbMapSummit[] {
  const summits = new Map<string, ClimbMapSummit>();

  results.forEach((result) => {
    const details = result.climbDetails;
    if (!details) {
      return;
    }
    const { latitude, longitude } = details.summitCoordinate;
    if (!validCoordinate(latitude, longitude)) {
      return;
    }

    const summitId = climbSummitId(result);
    const climbed = details.ascentCount > 0 || result.nbCheckedActivities > 0;
    const variant: ClimbMapVariant = {
      id: climbVariantId(result),
      summitId,
      label: result.badge.label,
      category: result.badge.category?.trim().toUpperCase() || "—",
      climbed,
      details,
      result,
    };
    const existing = summits.get(summitId);
    if (existing) {
      existing.variants.push(variant);
      existing.climbed ||= climbed;
      existing.ascentCount += details.ascentCount;
      return;
    }

    summits.set(summitId, {
      id: summitId,
      name: details.name,
      country: details.country,
      massif: details.massif,
      latitude,
      longitude,
      summitAltitude: details.summitAltitude,
      climbed,
      ascentCount: details.ascentCount,
      variants: [variant],
    });
  });

  return [...summits.values()]
    .map((summit) => ({
      ...summit,
      variants: [...summit.variants].sort((left, right) => left.label.localeCompare(right.label)),
    }))
    .sort((left, right) => left.name.localeCompare(right.name));
}

export function filterClimbMapSummits(
  summits: ClimbMapSummit[],
  filters: ClimbMapFilters,
  favoriteIds: ReadonlySet<string>,
): ClimbMapSummit[] {
  return summits.flatMap((summit) => {
    if (filters.country !== CLIMB_MAP_ALL_FILTER && summit.country !== filters.country) {
      return [];
    }
    if (filters.massif !== CLIMB_MAP_ALL_FILTER && summit.massif !== filters.massif) {
      return [];
    }
    if (filters.status === "FAVORITE" && !favoriteIds.has(summit.id)) {
      return [];
    }

    const variants = filters.category === CLIMB_MAP_ALL_FILTER
      ? summit.variants
      : summit.variants.filter((variant) => variant.category === filters.category);
    if (variants.length === 0) {
      return [];
    }
    const climbed = variants.some((variant) => variant.climbed);
    if (filters.status === "CLIMBED" && !climbed) {
      return [];
    }
    if (filters.status === "UNCLIMBED" && climbed) {
      return [];
    }

    return [{
      ...summit,
      climbed,
      ascentCount: variants.reduce((total, variant) => total + variant.details.ascentCount, 0),
      variants,
    }];
  });
}

function clusterCellSize(zoom: number): number {
  if (zoom >= 11) return 0;
  if (zoom >= 10) return 0.12;
  if (zoom >= 9) return 0.25;
  if (zoom >= 8) return 0.5;
  if (zoom >= 7) return 1;
  if (zoom >= 6) return 2;
  return 4;
}

export function clusterClimbMapSummits(summits: ClimbMapSummit[], zoom: number): ClimbMapCluster[] {
  const cellSize = clusterCellSize(zoom);
  if (cellSize === 0) {
    return summits.map((summit) => ({
      id: summit.id,
      latitude: summit.latitude,
      longitude: summit.longitude,
      summits: [summit],
    }));
  }

  const cells = new Map<string, ClimbMapSummit[]>();
  summits.forEach((summit) => {
    const key = `${Math.floor((summit.latitude + 90) / cellSize)}:${Math.floor((summit.longitude + 180) / cellSize)}`;
    const cell = cells.get(key);
    if (cell) {
      cell.push(summit);
    } else {
      cells.set(key, [summit]);
    }
  });

  return [...cells.entries()].map(([key, groupedSummits]) => ({
    id: `cluster-${zoom}-${key}`,
    latitude: groupedSummits.reduce((total, summit) => total + summit.latitude, 0) / groupedSummits.length,
    longitude: groupedSummits.reduce((total, summit) => total + summit.longitude, 0) / groupedSummits.length,
    summits: groupedSummits,
  }));
}

import type { LocationQuery, LocationQueryRaw } from "vue-router";
import { ALL_ACTIVITY_TYPES } from "@/utils/activityTypes";

export interface RouteFilters {
  year: string;
  activityType: string;
}

const knownActivityTypes = new Set<string>(ALL_ACTIVITY_TYPES);

function singleQueryValue(value: LocationQuery[string]): string | null {
  return typeof value === "string" ? value : null;
}

export function filtersFromQuery(query: LocationQuery, fallback: RouteFilters): RouteFilters {
  const requestedYear = singleQueryValue(query.year);
  const year = requestedYear === "all"
    ? "All years"
    : requestedYear && /^\d{4}$/.test(requestedYear)
      ? requestedYear
      : fallback.year;

  const requestedTypes = singleQueryValue(query.activityType)?.split("_").filter(Boolean) ?? [];
  const activityType = requestedTypes.length > 0 && requestedTypes.every((type) => knownActivityTypes.has(type))
    ? [...new Set(requestedTypes)].sort().join("_")
    : fallback.activityType;

  return { year, activityType };
}

export function filtersToQuery(filters: RouteFilters, current: LocationQuery): LocationQueryRaw {
  return {
    ...current,
    year: filters.year === "All years" ? "all" : filters.year,
    activityType: filters.activityType,
  };
}

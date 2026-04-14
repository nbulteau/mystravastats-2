import { ErrorService } from "@/services/error.service";

export async function requestJson<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  if (!response.ok) {
    await ErrorService.catchError(response);
  }
  return response.json() as Promise<T>;
}

export function buildFilteredApiUrl(
  path: string,
  activityType: string,
  currentYear: string,
): string {
  const params = new URLSearchParams({
    activityType,
  });

  if (currentYear !== "All years") {
    params.set("year", currentYear);
  }

  return `/api/${path}?${params.toString()}`;
}

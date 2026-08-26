import { ErrorService } from "@/services/error.service";

export async function requestJson<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await requestResponse(url, init);
  return response.json() as Promise<T>;
}

export async function requestResponse(url: string, init?: RequestInit): Promise<Response> {
  const response = await fetch(url, init);
  if (!response.ok) {
    await ErrorService.catchError(response);
  }
  return response;
}

export async function requestVoid(url: string, init?: RequestInit): Promise<void> {
  await requestResponse(url, init);
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

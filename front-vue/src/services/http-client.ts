import { ErrorService } from "@/services/error.service";
import { apiUrl } from "@/services/api-url";
import type { ApiOperationId } from "@/generated/api-contract";

export async function requestJson<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await requestResponse(url, init);
  return response.json() as Promise<T>;
}

export async function requestResponse(url: string, init?: RequestInit): Promise<Response> {
  const response = await fetchResponse(url, init);
  if (!response.ok) {
    await ErrorService.catchError(response);
  }
  return response;
}

export function fetchResponse(url: string, init?: RequestInit): Promise<Response> {
  return fetch(url, init);
}

export async function requestVoid(url: string, init?: RequestInit): Promise<void> {
  await requestResponse(url, init);
}

export function buildFilteredApiUrl(
  operationId: ApiOperationId,
  activityType: string,
  currentYear: string,
): string {
  return apiUrl(operationId, {
    query: {
      activityType,
      year: currentYear === "All years" ? undefined : currentYear,
    },
  });
}

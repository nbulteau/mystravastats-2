import { apiOperations, type ApiOperationId } from "@/generated/api-contract";

type PathValue = string | number;
type QueryValue = string | number | boolean | null | undefined;

export type ApiUrlOptions = {
  path?: Record<string, PathValue>;
  query?: Record<string, QueryValue>;
};

export function apiUrl(operationId: ApiOperationId, options: ApiUrlOptions = {}): string {
  const operation = apiOperations[operationId];
  let path = operation.path.replace(/\{([^}]+)\}/g, (_placeholder, parameter: string) => {
    const value = options.path?.[parameter];
    if (value === undefined) {
      throw new Error(`Missing path parameter "${parameter}" for API operation "${operationId}".`);
    }
    return encodeURIComponent(String(value));
  });

  const query = new URLSearchParams();
  for (const [name, value] of Object.entries(options.query ?? {})) {
    if (value !== undefined && value !== null) query.set(name, String(value));
  }
  const serializedQuery = query.toString();
  if (serializedQuery) path += `?${serializedQuery}`;
  return path;
}

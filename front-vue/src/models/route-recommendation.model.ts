export type RouteMode = "SHAPE";

export type RouteType =
  | "RIDE"
  | "MTB"
  | "GRAVEL"
  | "RUN"
  | "TRAIL"
  | "HIKE";

export type ShapeInputType =
  | "draw"
  | "polyline"
  | "gpx"
  | "svg";

export type RouteCoordinate = ContractRouteCoordinate;
export type RouteGenerationScore = ContractRouteGenerationScore;
export type GeneratedRoute = ContractGeneratedRoute;
export type GenerateRoutesResponse = ContractGenerateRoutesResponse;

export interface EditGeneratedRouteRequest {
  routeType?: RouteType;
  controlPoints: RouteCoordinate[];
}

export interface EditGeneratedRouteResponse {
  route?: GeneratedRoute;
  controlPoints?: RouteCoordinate[];
  diagnostics?: RouteGenerationDiagnostic[];
}

export type RouteGenerationDiagnostic = ContractRouteGenerationDiagnostic;
import type {
  GeneratedRoute as ContractGeneratedRoute,
  GenerateRoutesResponse as ContractGenerateRoutesResponse,
  RouteCoordinate as ContractRouteCoordinate,
  RouteGenerationDiagnostic as ContractRouteGenerationDiagnostic,
  RouteGenerationScore as ContractRouteGenerationScore,
} from "@/generated/api-contract";

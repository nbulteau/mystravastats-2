export type RouteMode = "TARGET" | "SHAPE";
export type TargetGenerationMode = "AUTOMATIC" | "CUSTOM";

export type RouteType =
  | "RIDE"
  | "MTB"
  | "GRAVEL"
  | "RUN"
  | "TRAIL"
  | "HIKE";

export type StartDirection =
  | "UNDEFINED"
  | "N"
  | "S"
  | "E"
  | "W";

export type BacktrackingProfile =
  | "BALANCED"
  | "STRICT"
  | "ULTRA";

export type ShapeInputType =
  | "draw"
  | "polyline";

export interface RouteCoordinate {
  lat: number;
  lng: number;
}

export interface RouteGenerationScore {
  global: number;
  distance: number;
  elevation: number;
  duration: number;
  direction: number;
  shape: number;
  roadFitness: number;
}

export interface GeneratedRoute {
  routeId: string;
  title: string;
  variantType: string;
  routeType?: string;
  startDirection?: string;
  distanceKm: number;
  elevationGainM: number;
  durationSec: number;
  estimatedDurationSec: number;
  score: RouteGenerationScore;
  reasons: string[];
  previewLatLng: number[][];
  start?: RouteCoordinate;
  end?: RouteCoordinate;
  activityId?: number;
  isRoadGraphGenerated: boolean;
}

export interface GenerateRoutesResponse {
  routes: GeneratedRoute[];
  diagnostics?: RouteGenerationDiagnostic[];
}

export interface RouteGenerationDiagnostic {
  code: string;
  message: string;
}

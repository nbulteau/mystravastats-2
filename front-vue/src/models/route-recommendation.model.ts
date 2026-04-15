import type { ActivityShort } from "@/models/activity.model";

export interface RouteCoordinate {
  lat: number;
  lng: number;
}

export interface RouteRecommendation {
  activity: ActivityShort;
  activityDate: string;
  distanceKm: number;
  elevationGainM: number;
  durationSec: number;
  isLoop: boolean;
  start?: RouteCoordinate;
  end?: RouteCoordinate;
  startArea: string;
  season: string;
  variantType: string;
  matchScore: number;
  reasons: string[];
  previewLatLng: number[][];
  shape?: string;
  shapeScore?: number;
  experimental: boolean;
}

export interface ShapeRemixRecommendation {
  id: string;
  shape: string;
  distanceKm: number;
  elevationGainM: number;
  durationSec: number;
  matchScore: number;
  reasons: string[];
  components: ActivityShort[];
  previewLatLng: number[][];
  experimental: boolean;
}

export interface RouteExplorerResult {
  closestLoops: RouteRecommendation[];
  variants: RouteRecommendation[];
  seasonal: RouteRecommendation[];
  shapeMatches: RouteRecommendation[];
  shapeRemixes: ShapeRemixRecommendation[];
}


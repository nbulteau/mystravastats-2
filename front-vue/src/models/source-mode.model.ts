export type SourceMode = "STRAVA" | "FIT" | "GPX";

export interface SourceModePreviewRequest {
  mode: SourceMode;
  path: string;
}

export interface SourceModeYearPreview {
  year: string;
  fileCount: number;
  validFileCount: number;
  activityCount: number;
}

export interface SourceModePreviewError {
  path: string;
  message: string;
}

export interface SourceModeEnvironmentVariable {
  key: string;
  value: string;
  required: boolean;
}

export interface SourceModePreview {
  mode: SourceMode;
  activeMode: SourceMode;
  path: string;
  configKey: string;
  supported: boolean;
  active: boolean;
  configured: boolean;
  readable: boolean;
  validStructure: boolean;
  restartNeeded: boolean;
  activationCommand: string;
  fileCount: number;
  validFileCount: number;
  invalidFileCount: number;
  activityCount: number;
  years: SourceModeYearPreview[];
  missingFields: string[];
  environment: SourceModeEnvironmentVariable[];
  errors: SourceModePreviewError[];
  recommendations: string[];
}

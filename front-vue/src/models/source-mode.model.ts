export type SourceMode = "STRAVA" | "FIT" | "GPX";

export interface SourceModePreviewRequest {
  mode: SourceMode;
  path: string;
}

export interface StravaOAuthStartRequest {
  path: string;
  clientId: string;
  clientSecret: string;
  useCache: boolean;
}

export interface StravaOAuthStartResult {
  status: string;
  message: string;
  authorizeUrl: string;
  settingsUrl: string;
  callbackDomain: string;
  oauthCallbackUrl: string;
  credentialsFile: string;
  tokenFile: string;
  cacheOnly: boolean;
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

export interface StravaOAuthStatus {
  status: string;
  message: string;
  settingsUrl: string;
  callbackDomain: string;
  oauthCallbackUrl: string;
  setupCommand: string;
  credentialsFile: string;
  tokenFile: string;
  credentialsFilePresent: boolean;
  credentialsPresent: boolean;
  clientIdPresent: boolean;
  clientSecretPresent: boolean;
  cacheOnly: boolean;
  tokenPresent: boolean;
  tokenReadable: boolean;
  accessTokenPresent: boolean;
  refreshTokenPresent: boolean;
  tokenExpired: boolean;
  tokenExpiresAt: string;
  athleteId: string;
  athleteName: string;
  scopesVerified: boolean;
  grantedScopes: string[];
  requiredScopes: string[];
  missingScopes: string[];
  tokenError: string;
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
  stravaOAuth: StravaOAuthStatus | null;
}

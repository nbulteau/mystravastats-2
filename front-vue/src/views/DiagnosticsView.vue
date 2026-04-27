<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useContextStore } from "@/stores/context";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import TooltipHint from "@/components/TooltipHint.vue";
import { ToastTypeEnum } from "@/models/toast.model";
import type { DataQualityCorrection, DataQualityIssue, DataQualitySummary } from "@/models/data-quality.model";
import type { HealthRecord } from "@/models/health.model";
import type { SourceMode } from "@/models/source-mode.model";
import { RouterLink } from "vue-router";

const contextStore = useContextStore();
const diagnosticsStore = useDiagnosticsStore();
const uiStore = useUiStore();
const showRawPayload = ref(false);
const sourceModePathEdited = ref(false);
const sourceModeInitialized = ref(false);
const qualityActionActivityId = ref<number | null>(null);
const qualityActionIssueId = ref<string | null>(null);
const qualityBatchActionInProgress = ref(false);
const showAllDataQualityIssues = ref(false);
const dataQualityPreviewLimit = 12;
const selectedSourceMode = ref<SourceMode>("STRAVA");
const sourceModePath = ref("");
const sourceModeOptions: Array<{ mode: SourceMode; label: string; icon: string }> = [
  { mode: "STRAVA", label: "Strava", icon: "fa-brands fa-strava" },
  { mode: "FIT", label: "FIT", icon: "fa-solid fa-file-lines" },
  { mode: "GPX", label: "GPX", icon: "fa-solid fa-route" },
];

onMounted(() => contextStore.updateCurrentView("diagnostics"));

const health = computed(() => diagnosticsStore.health);
const root = computed(() => asRecord(health.value));
const routing = computed(() => asRecord(root.value.routing));
const manifest = computed(() => asRecord(root.value.manifest));
const warmup = computed(() => asRecord(manifest.value.warmup));
const bestEffortCache = computed(() => asRecord(manifest.value.bestEffortCache));
const rateLimit = computed(() => asRecord(root.value.rateLimit));
const refresh = computed(() => asRecord(root.value.refresh));
const files = computed(() => asRecord(root.value.files));
const runtimeConfig = computed(() => asRecord(root.value.runtimeConfig));
const runtimeData = computed(() => asRecord(runtimeConfig.value.data));
const runtimeServer = computed(() => asRecord(runtimeConfig.value.server));
const runtimeCors = computed(() => asRecord(runtimeConfig.value.cors));
const runtimeRouting = computed(() => asRecord(runtimeConfig.value.routing));
const hasRuntimeConfig = computed(() => Object.keys(runtimeConfig.value).length > 0);
const dataQualitySummary = computed<DataQualitySummary | null>(() => {
  if (diagnosticsStore.dataQualityReport) {
    return diagnosticsStore.dataQualityReport.summary;
  }
  return normalizeDataQualitySummary(root.value.dataQuality);
});
const dataQualityIssues = computed<DataQualityIssue[]>(() => diagnosticsStore.dataQualityReport?.issues ?? dataQualitySummary.value?.topIssues ?? []);
const dataQualityCorrections = computed<DataQualityCorrection[]>(() => diagnosticsStore.dataQualityReport?.corrections ?? []);
const activeDataQualityCorrections = computed(() => dataQualityCorrections.value.filter((correction) => correction.status !== "reverted"));
const hasFullDataQualityReport = computed(() => diagnosticsStore.dataQualityReport !== null);
const displayedDataQualityIssues = computed(() => {
  if (showAllDataQualityIssues.value) {
    return dataQualityIssues.value;
  }
  return dataQualityIssues.value.slice(0, dataQualityPreviewLimit);
});
const dataQualityIssueListLabel = computed(() => {
  const visible = displayedDataQualityIssues.value.length;
  const total = hasFullDataQualityReport.value
    ? dataQualityIssues.value.length
    : dataQualitySummary.value?.issueCount ?? dataQualityIssues.value.length;
  if (total <= visible) {
    return `${formatInteger(visible)} issues`;
  }
  return `Showing ${formatInteger(visible)} of ${formatInteger(total)} issues`;
});
const canToggleDataQualityIssues = computed(() => hasFullDataQualityReport.value && dataQualityIssues.value.length > dataQualityPreviewLimit);
const dataQualityStatusLabel = computed(() => {
  const status = dataQualitySummary.value?.status ?? "not_applicable";
  if (status === "ok") return "OK";
  if (status === "critical") return "Critical";
  if (status === "warning") return "To check";
  return "No local source";
});
const dataQualityStatusClass = computed(() => {
  const status = dataQualitySummary.value?.status ?? "not_applicable";
  if (status === "ok") return "status-chip status-chip--up";
  if (status === "critical") return "status-chip status-chip--down";
  if (status === "warning") return "status-chip status-chip--warn";
  return "status-chip status-chip--neutral";
});
const dataQualityStats = computed(() => {
  const summary = dataQualitySummary.value;
  const bySeverity = summary?.bySeverity ?? {};
  const criticalCount = bySeverity.critical ?? 0;
  const warningCount = bySeverity.warning ?? 0;
  const infoCount = bySeverity.info ?? 0;
  const hasActionableFindings = criticalCount > 0 || warningCount > 0;
  return [
    { label: "Total issues", value: formatInteger(summary?.issueCount ?? 0), tone: hasActionableFindings ? "warn" : "neutral" },
    { label: "Affected activities", value: formatInteger(summary?.impactedActivities ?? 0), tone: hasActionableFindings ? "warn" : "neutral" },
    { label: "Excluded from stats", value: formatInteger(summary?.excludedActivities ?? 0), tone: (summary?.excludedActivities ?? 0) > 0 ? "warn" : "neutral" },
    { label: "Safe fixes", value: formatInteger(summary?.safeCorrectionCount ?? 0), tone: (summary?.safeCorrectionCount ?? 0) > 0 ? "warn" : "neutral" },
    { label: "Active fixes", value: formatInteger(summary?.correctionCount ?? activeDataQualityCorrections.value.length), tone: activeDataQualityCorrections.value.length > 0 ? "warn" : "neutral" },
    { label: "Critical issues", value: formatInteger(criticalCount), tone: criticalCount > 0 ? "down" : "neutral" },
    { label: "Warning issues", value: formatInteger(warningCount), tone: warningCount > 0 ? "warn" : "neutral" },
    { label: "Info issues", value: formatInteger(infoCount), tone: "neutral" },
  ];
});
const dataQualityCategories = computed(() =>
  Object.entries(dataQualitySummary.value?.byCategory ?? {})
    .sort((left, right) => {
      const priorityDelta = dataQualityCategoryPriority(left[0]) - dataQualityCategoryPriority(right[0]);
      if (priorityDelta !== 0) return priorityDelta;
      return right[1] - left[1];
    })
    .map(([category, count]) => ({ category, count })),
);

const providerLabel = computed(() => formatProvider(textValue(root.value.provider) || inferProvider(root.value)));
const activityCount = computed(() => numberValue(root.value.activities));
const sourcePath = computed(() =>
  textValue(root.value.cacheRoot)
  || textValue(root.value.fitDirectory)
  || textValue(root.value.gpxDirectory)
  || textValue(runtimeData.value.fitFilesPath)
  || textValue(runtimeData.value.gpxFilesPath)
  || textValue(runtimeData.value.stravaCachePath)
  || "",
);
const availableYears = computed(() => listValues(root.value.availableYearBins).sort((left, right) => right.localeCompare(left)));
const timestampLabel = computed(() => formatDateTime(textValue(root.value.timestamp)));
const loadedAtLabel = computed(() => formatDateTime(diagnosticsStore.lastLoadedAt));
const rateLimitActive = computed(() => booleanValue(rateLimit.value.active));
const rateLimitUntilLabel = computed(() => formatEpochMs(numberValue(rateLimit.value.untilEpochMs)));
const routingStatus = computed(() => (textValue(routing.value.status) || "unknown").toLowerCase());
const routingReachable = computed(() => booleanValue(routing.value.reachable));
const routingStatusLabel = computed(() => {
  if (routingStatus.value === "up" && routingReachable.value) return "Online";
  if (routingStatus.value === "disabled") return "Disabled";
  if (routingStatus.value === "misconfigured") return "Misconfigured";
  if (routingStatus.value === "down") return "Offline";
  return "Unknown";
});
const routingStatusClass = computed(() => statusClass(routingStatus.value));
const healthStatusLabel = computed(() => {
  if (diagnosticsStore.error) return "Backend unavailable";
  if (rateLimitActive.value) return "Cache-only";
  if (routingStatus.value === "down" || routingStatus.value === "misconfigured") return "Degraded";
  if (routingStatus.value === "disabled") return "Limited";
  return "Operational";
});
const healthStatusClass = computed(() => {
  if (diagnosticsStore.error) return "status-chip status-chip--down";
  if (rateLimitActive.value) return "status-chip status-chip--warn";
  return statusClass(routingStatus.value === "down" || routingStatus.value === "misconfigured" ? routingStatus.value : "up");
});
const warmupStatusItems = computed(() => [
  { label: "Résumé annuel", value: textValue(warmup.value.priority1) || "n/a" },
  { label: "Records principaux", value: textValue(warmup.value.priority2) || "n/a" },
  { label: "Métriques avancées", value: textValue(warmup.value.priority3) || "n/a" },
]);
const warmupInProgress = computed(() => booleanValue(refresh.value.warmupInProgress));
const backgroundRefreshInProgress = computed(() => booleanValue(refresh.value.backgroundInProgress));
const preparedYears = computed(() => listValues(warmup.value.preparedYears).sort((left, right) => right.localeCompare(left)));
const bestEffortEntries = computed(() => ({
  persisted: numberValue(bestEffortCache.value.entriesPersisted),
  memory: numberValue(bestEffortCache.value.entriesInMemory),
}));
const supportedRouteTypes = computed(() => listValues(routing.value.supportedRouteTypes));
const cacheFiles = computed(() =>
  Object.entries(files.value).map(([name, value]) => {
    const details = asRecord(value);
    const sizeBytes = numberValue(details.sizeBytes);
    const lastModifiedRaw = textValue(details.lastModified);
    return {
      name,
      path: textValue(details.path) || "n/a",
      exists: booleanValue(details.exists),
      sizeBytes,
      size: formatBytes(sizeBytes),
      lastModifiedRaw,
      lastModified: formatDateTime(lastModifiedRaw),
    };
  }),
);
const cacheFileStats = computed(() => {
  const presentFiles = cacheFiles.value.filter((file) => file.exists);
  const missingFiles = cacheFiles.value.filter((file) => !file.exists);
  const totalBytes = presentFiles.reduce((total, file) => total + (file.sizeBytes ?? 0), 0);
  const latestTimestamp = presentFiles.reduce((latest, file) => {
    if (!file.lastModifiedRaw) return latest;
    const timestamp = new Date(file.lastModifiedRaw).getTime();
    return Number.isNaN(timestamp) ? latest : Math.max(latest, timestamp);
  }, 0);

  return {
    total: cacheFiles.value.length,
    present: presentFiles.length,
    missing: missingFiles.length,
    totalBytes,
    latestModified: latestTimestamp > 0 ? new Date(latestTimestamp).toISOString() : "",
  };
});
const cacheStatusLabel = computed(() => {
  if (!sourcePath.value) return "Unknown";
  if (cacheFileStats.value.missing > 0) return "Partial";
  if ((activityCount.value ?? 0) <= 0) return "Empty";
  return "Ready";
});
const cacheStatusClass = computed(() => {
  if (cacheStatusLabel.value === "Ready") return "status-chip status-chip--up";
  if (cacheStatusLabel.value === "Partial") return "status-chip status-chip--warn";
  return "status-chip status-chip--neutral";
});
const cacheSourceLabel = computed(() => {
  const provider = textValue(root.value.provider) || textValue(runtimeData.value.provider);
  if (provider.toLowerCase() === "fit") return "FIT directory";
  if (provider.toLowerCase() === "gpx") return "GPX directory";
  if (provider.toLowerCase() === "strava") return "Strava cache";
  return "Data source";
});
const cacheSummaryItems = computed<Array<{ label: string; value: string; monospace: boolean }>>(() => [
  { label: "Source", value: cacheSourceLabel.value, monospace: false },
  { label: "Activities", value: formatInteger(activityCount.value), monospace: false },
  { label: "Years", value: availableYears.value.length > 0 ? availableYears.value.join(", ") : "n/a", monospace: false },
  { label: "Files", value: cacheFileStats.value.total > 0 ? `${cacheFileStats.value.present}/${cacheFileStats.value.total} present` : "n/a", monospace: false },
  { label: "Listed size", value: cacheFileStats.value.total > 0 ? formatBytes(cacheFileStats.value.totalBytes) : "n/a", monospace: false },
  { label: "Latest file", value: formatDateTime(cacheFileStats.value.latestModified), monospace: false },
  { label: "Manifest", value: formatDateTime(textValue(manifest.value.updatedAt)), monospace: false },
  { label: "Background refresh", value: backgroundRefreshInProgress.value ? "Running" : "Idle", monospace: false },
]);
const cacheManifestItems = computed<Array<{ label: string; value: string; monospace: boolean }>>(() => {
  if (Object.keys(manifest.value).length === 0) return [];
  return [
    { label: "Manifest schema", value: textValue(manifest.value.schemaVersion) || "n/a", monospace: false },
    { label: "Best-effort file", value: textValue(bestEffortCache.value.file) || "n/a", monospace: true },
    { label: "Best-effort saved", value: formatInteger(bestEffortEntries.value.persisted), monospace: false },
    { label: "Best-effort memory", value: formatInteger(bestEffortEntries.value.memory), monospace: false },
    { label: "Warmup file", value: textValue(warmup.value.file) || "n/a", monospace: true },
    { label: "Warmup last run", value: formatDateTime(textValue(warmup.value.lastRunAt)), monospace: false },
  ];
});
const runtimeListenAddress = computed(() => {
  const host = textValue(runtimeServer.value.host) || textValue(runtimeServer.value.address);
  const port = textValue(runtimeServer.value.port);
  if (host && port) return `${host}:${port}`;
  return host || port || "n/a";
});
const runtimeConfigItems = computed<Array<{ label: string; value: string; monospace: boolean }>>(() => [
  { label: "Backend", value: textValue(runtimeConfig.value.backend) || "n/a", monospace: false },
  { label: "Provider", value: formatProvider(textValue(runtimeData.value.provider)), monospace: false },
  { label: "Strava cache", value: textValue(runtimeData.value.stravaCachePath) || "n/a", monospace: true },
  { label: "FIT files", value: textValue(runtimeData.value.fitFilesPath) || "n/a", monospace: true },
  { label: "GPX files", value: textValue(runtimeData.value.gpxFilesPath) || "n/a", monospace: true },
  { label: "CORS origins", value: displayList(runtimeCors.value.allowedOrigins), monospace: true },
  { label: "CORS headers", value: displayList(runtimeCors.value.allowedHeaders), monospace: true },
  { label: "CORS credentials", value: yesNo(runtimeCors.value.allowCredentials), monospace: false },
  { label: "Listen", value: runtimeListenAddress.value, monospace: true },
  { label: "OSRM base URL", value: textValue(runtimeRouting.value.baseUrl) || "n/a", monospace: true },
  { label: "OSRM enabled", value: yesNo(runtimeRouting.value.enabled), monospace: false },
  { label: "History bias", value: yesNo(runtimeRouting.value.historyBiasEnabled), monospace: false },
]);
const sourceModePreview = computed(() => diagnosticsStore.sourceModePreview);
const activeSourceMode = computed(() => normalizeSourceMode(textValue(runtimeData.value.provider) || textValue(root.value.provider)));
const sourceModeConfigKey = computed(() => configKeyForSourceMode(selectedSourceMode.value));
const sourceModePathLabel = computed(() => {
  if (selectedSourceMode.value === "STRAVA") return "Cache path";
  return `${selectedSourceMode.value} directory`;
});
const sourceModeStatusLabel = computed(() => {
  const preview = sourceModePreview.value;
  if (diagnosticsStore.sourceModePreviewError) return "Unavailable";
  if (!preview) return "Not checked";
  if (preview.active) return "Active";
  if (!preview.supported || preview.errors.length > 0 || !preview.readable || !preview.validStructure) return "Needs attention";
  if (preview.restartNeeded) return "Ready after restart";
  return "Ready";
});
const sourceModeStatusClass = computed(() => {
  const preview = sourceModePreview.value;
  if (diagnosticsStore.sourceModePreviewError) return "status-chip status-chip--down";
  if (!preview) return "status-chip status-chip--neutral";
  if (preview.active) return "status-chip status-chip--up";
  if (!preview.supported || preview.errors.length > 0 || !preview.readable || !preview.validStructure) return "status-chip status-chip--warn";
  if (preview.restartNeeded) return "status-chip status-chip--warn";
  return "status-chip status-chip--up";
});
const sourceModePreviewStats = computed<Array<{ label: string; value: string; tone?: "warn" | "neutral" }>>(() => {
  const preview = sourceModePreview.value;
  if (!preview) return [];
  return [
    { label: "Activities", value: formatInteger(preview.activityCount) },
    { label: "Files", value: `${formatInteger(preview.validFileCount)}/${formatInteger(preview.fileCount)}` },
    { label: "Invalid", value: formatInteger(preview.invalidFileCount), tone: preview.invalidFileCount > 0 ? "warn" : "neutral" },
    { label: "Years", value: formatInteger(preview.years.length) },
    { label: "Config", value: preview.configured ? "Set" : "Unset", tone: preview.configured ? "neutral" : "warn" },
    { label: "Restart", value: preview.restartNeeded ? "Required" : "No", tone: preview.restartNeeded ? "warn" : "neutral" },
  ];
});
const sourceModeActivationCommand = computed(() => sourceModePreview.value?.activationCommand ?? "");
const sourceModeActivationSummary = computed<Array<{ label: string; value: string; tone?: "warn" | "up" }>>(() => {
  const preview = sourceModePreview.value;
  const verification = preview?.active
    ? "Active"
    : preview?.restartNeeded
      ? "Restart required"
      : preview
        ? "No restart"
        : "Preview first";
  return [
    { label: "Current", value: formatProvider(activeSourceMode.value), tone: activeSourceMode.value === selectedSourceMode.value ? "up" : "warn" },
    { label: "Target", value: selectedSourceMode.value },
    { label: "Verification", value: verification, tone: preview?.active ? "up" : preview ? "warn" : undefined },
  ];
});
const sourceModeEnvironment = computed(() => sourceModePreview.value?.environment ?? []);
const degradedReasons = computed(() => {
  const reasons: Array<{ title: string; detail: string; tone: "warn" | "down" | "info" }> = [];
  if (diagnosticsStore.error) {
    reasons.push({
      title: "Backend API",
      detail: diagnosticsStore.error,
      tone: "down",
    });
  }
  if (rateLimitActive.value) {
    reasons.push({
      title: "Strava rate limit",
      detail: rateLimitUntilLabel.value === "n/a"
        ? "Network refresh is paused and cached data is used."
        : `Network refresh is paused until ${rateLimitUntilLabel.value}.`,
      tone: "warn",
    });
  }
  if (routingStatus.value === "down" || routingStatus.value === "misconfigured") {
    reasons.push({
      title: "OSRM routing",
      detail: textValue(routing.value.error) || "Road graph checks are unavailable.",
      tone: "down",
    });
  }
  if (routingStatus.value === "disabled") {
    reasons.push({
      title: "OSRM routing",
      detail: "Route generation is limited to non-OSRM fallbacks.",
      tone: "info",
    });
  }
  if (warmupInProgress.value || backgroundRefreshInProgress.value) {
    reasons.push({
      title: "Background jobs",
      detail: warmupInProgress.value ? "Warmup is still running." : "Cache refresh is still running.",
      tone: "info",
    });
  }
  return reasons;
});
const rawPayload = computed(() => JSON.stringify(health.value ?? {}, null, 2));

watch(
  runtimeData,
  () => {
    if (!sourceModeInitialized.value) {
      selectedSourceMode.value = normalizeSourceMode(textValue(runtimeData.value.provider));
      sourceModePath.value = defaultSourceModePath(selectedSourceMode.value);
      sourceModeInitialized.value = true;
      return;
    }

    if (!sourceModePathEdited.value) {
      sourceModePath.value = defaultSourceModePath(selectedSourceMode.value);
    }
  },
  { immediate: true },
);

function asRecord(value: unknown): HealthRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? value as HealthRecord
    : {};
}

function normalizeDataQualitySummary(value: unknown): DataQualitySummary | null {
  const record = asRecord(value);
  if (Object.keys(record).length === 0) {
    return null;
  }
  return {
    status: textValue(record.status) || "not_applicable",
    provider: textValue(record.provider),
    issueCount: numberValue(record.issueCount) ?? 0,
    impactedActivities: numberValue(record.impactedActivities) ?? 0,
    excludedActivities: numberValue(record.excludedActivities) ?? 0,
    bySeverity: numberRecord(record.bySeverity),
    byCategory: numberRecord(record.byCategory),
    topIssues: Array.isArray(record.topIssues)
      ? record.topIssues.map(normalizeDataQualityIssue).filter((issue): issue is DataQualityIssue => issue !== null)
      : [],
  };
}

function normalizeDataQualityIssue(value: unknown): DataQualityIssue | null {
  const record = asRecord(value);
  if (!textValue(record.id) && !textValue(record.message)) {
    return null;
  }
  return {
    id: textValue(record.id) || `${textValue(record.category)}-${textValue(record.field)}-${textValue(record.activityId)}`,
    source: textValue(record.source),
    activityId: numberValue(record.activityId),
    activityName: textValue(record.activityName),
    activityType: textValue(record.activityType),
    year: textValue(record.year),
    filePath: textValue(record.filePath),
    severity: textValue(record.severity) || "info",
    category: textValue(record.category),
    field: textValue(record.field),
    message: textValue(record.message),
    rawValue: textValue(record.rawValue),
    suggestion: textValue(record.suggestion),
    excludedFromStats: booleanValue(record.excludedFromStats),
    excludedAt: textValue(record.excludedAt),
  };
}

function numberRecord(value: unknown): Record<string, number> {
  const record = asRecord(value);
  return Object.fromEntries(
    Object.entries(record)
      .map(([key, entry]) => [key, numberValue(entry) ?? 0]),
  );
}

function textValue(value: unknown): string {
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

function numberValue(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

function booleanValue(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "string") return value.toLowerCase() === "true";
  return false;
}

function listValues(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((item) => textValue(item))
    .filter((item) => item.length > 0);
}

function displayList(value: unknown): string {
  const values = listValues(value);
  return values.length > 0 ? values.join(", ") : "n/a";
}

function yesNo(value: unknown): string {
  return booleanValue(value) ? "Yes" : "No";
}

function inferProvider(payload: HealthRecord): string {
  if (payload.fitDirectory) return "fit";
  if (payload.gpxDirectory) return "gpx";
  if (payload.cacheRoot || payload.manifest) return "strava";
  return "unknown";
}

function formatProvider(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "gpx") return "GPX";
  if (normalized === "fit") return "FIT";
  if (normalized === "strava") return "Strava";
  if (normalized === "") return "Unknown";
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}

function formatDateTime(value: string | null): string {
  if (!value) return "n/a";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function formatEpochMs(value: number | null): string {
  if (!value || value <= 0) return "n/a";
  return formatDateTime(new Date(value).toISOString());
}

function formatBytes(value: number | null): string {
  if (value === null) return "n/a";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function formatInteger(value: number | null): string {
  if (value === null) return "n/a";
  return new Intl.NumberFormat().format(value);
}

function formatSignedMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(0)} m`;
}

function formatSignedDistance(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : "";
  return `${sign}${(value / 1000).toFixed(2)} km`;
}

function statusClass(status: string): string {
  if (status === "up") return "status-chip status-chip--up";
  if (status === "disabled") return "status-chip status-chip--neutral";
  if (status === "misconfigured") return "status-chip status-chip--warn";
  if (status === "down") return "status-chip status-chip--down";
  return "status-chip status-chip--neutral";
}

function fileStatusClass(exists: boolean): string {
  return exists ? "file-state file-state--ok" : "file-state file-state--missing";
}

function normalizeSourceMode(value: string): SourceMode {
  const normalized = value.trim().toUpperCase();
  if (normalized === "FIT" || normalized === "GPX") return normalized;
  return "STRAVA";
}

function configKeyForSourceMode(mode: SourceMode): string {
  if (mode === "FIT") return "FIT_FILES_PATH";
  if (mode === "GPX") return "GPX_FILES_PATH";
  return "STRAVA_CACHE_PATH";
}

function defaultSourceModePath(mode: SourceMode): string {
  if (mode === "FIT") {
    return textValue(root.value.fitDirectory) || textValue(runtimeData.value.fitFilesPath) || "";
  }
  if (mode === "GPX") {
    return textValue(root.value.gpxDirectory) || textValue(runtimeData.value.gpxFilesPath) || "";
  }
  return textValue(root.value.cacheRoot) || textValue(runtimeData.value.stravaCachePath) || "strava-cache";
}

function selectSourceMode(mode: SourceMode) {
  selectedSourceMode.value = mode;
  sourceModePathEdited.value = false;
  sourceModePath.value = defaultSourceModePath(mode);
  clearSourceModePreview();
}

function markSourceModePathEdited() {
  sourceModePathEdited.value = true;
  clearSourceModePreview();
}

function clearSourceModePreview() {
  diagnosticsStore.sourceModePreview = null;
  diagnosticsStore.sourceModePreviewError = null;
}

function formatSourceField(value: string): string {
  const labels: Record<string, string> = {
    activities: "Activities",
    trace: "GPS trace",
    elevation: "Elevation",
    heartRate: "Heart rate",
    power: "Power",
    cadence: "Cadence",
  };
  return labels[value] || value;
}

function formatDataQualityCategory(value: string): string {
  const labels: Record<string, string> = {
    INVALID_FILE: "Invalid file",
    MISSING_STREAM: "Missing detailed stream",
    MISSING_STREAM_FIELD: "Missing stream field",
    STREAM_DATA_COVERAGE: "Stream coverage",
    INVALID_VALUE: "Invalid value",
    INCONSISTENT_TIME: "Time",
    GPS_GLITCH: "GPS glitch",
    ALTITUDE_SPIKE: "Altitude spike",
    FALLBACK_VALUE: "Fallback",
  };
  return labels[value] || value;
}

function correctionLabel(value: string): string {
  const labels: Record<string, string> = {
    REMOVE_GPS_POINT: "Remove GPS point",
    SMOOTH_ALTITUDE_SPIKE: "Smooth altitude spike",
    MASK_INVALID_VALUE: "Mask invalid value",
    RECALCULATE_FROM_STREAM: "Recalculate from stream",
  };
  return labels[value] || value;
}

function dataQualityCategoryTooltip(value: string): string {
  const descriptions: Record<string, string> = {
    INVALID_FILE: "The source file could not be parsed reliably or contains an unsupported payload.",
    MISSING_STREAM: "The activity has no detailed stream in the local cache. In Strava mode this can usually be fetched from the API.",
    MISSING_STREAM_FIELD: "A detailed stream exists, but one required field such as time, distance, GPS trace, or altitude is missing or inconsistent.",
    STREAM_DATA_COVERAGE: "Optional sensor samples are incomplete. Summary values may exist, but charts using sample-by-sample data will be partial.",
    INVALID_VALUE: "A summary value is missing, not serializable, or outside a plausible range for the activity type.",
    INCONSISTENT_TIME: "Timing fields disagree, for example moving time is greater than elapsed time.",
    GPS_GLITCH: "The GPS trace contains a jump that implies an impossible speed for this activity type.",
    ALTITUDE_SPIKE: "The altitude stream contains a sharp elevation jump that can distort elevation gain.",
    FALLBACK_VALUE: "The displayed value comes from a fallback calculation instead of the original source data.",
  };
  return descriptions[value] || "Data quality finding detected for this category.";
}

function dataQualityCategoryPriority(value: string): number {
  const priority: Record<string, number> = {
    INVALID_FILE: 0,
    INVALID_VALUE: 1,
    INCONSISTENT_TIME: 2,
    GPS_GLITCH: 3,
    ALTITUDE_SPIKE: 4,
    MISSING_STREAM_FIELD: 5,
    MISSING_STREAM: 6,
    FALLBACK_VALUE: 7,
    STREAM_DATA_COVERAGE: 8,
  };
  return priority[value] ?? 99;
}

function dataQualitySeverityClass(severity: string): string {
  if (severity === "critical") return "quality-severity quality-severity--critical";
  if (severity === "warning") return "quality-severity quality-severity--warning";
  return "quality-severity quality-severity--info";
}

function dataQualityStatClass(tone: string): string {
  if (tone === "down") return "quality-metric quality-metric--down";
  if (tone === "warn") return "quality-metric quality-metric--warn";
  return "quality-metric";
}

async function toggleStatsExclusion(issue: DataQualityIssue) {
  if (!issue.activityId) {
    return;
  }
  qualityActionActivityId.value = issue.activityId;
  try {
    if (issue.excludedFromStats) {
      await diagnosticsStore.includeActivityInStats(issue.activityId);
      uiStore.showToast({
        id: `quality-include-${Date.now()}`,
        type: ToastTypeEnum.NORMAL,
        message: "Activity included in statistics.",
        timeout: 2600,
      });
    } else {
      await diagnosticsStore.excludeActivityFromStats(issue.activityId);
      uiStore.showToast({
        id: `quality-exclude-${Date.now()}`,
        type: ToastTypeEnum.NORMAL,
        message: "Activity excluded from statistics.",
        timeout: 2600,
      });
    }
  } catch (error) {
    uiStore.showToast({
      id: `quality-action-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to update data quality exclusion.",
      timeout: 3600,
    });
  } finally {
    qualityActionActivityId.value = null;
  }
}

async function applyIssueCorrection(issue: DataQualityIssue) {
  if (!issue.correction?.available || issue.correction.safety !== "safe") {
    return;
  }
  qualityActionIssueId.value = issue.id;
  try {
    await diagnosticsStore.applyCorrection(issue.id);
    uiStore.showToast({
      id: `quality-fix-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Local correction applied.",
      timeout: 2600,
    });
  } catch (error) {
    uiStore.showToast({
      id: `quality-fix-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to apply local correction.",
      timeout: 3600,
    });
  } finally {
    qualityActionIssueId.value = null;
  }
}

async function applySafeCorrections() {
  qualityBatchActionInProgress.value = true;
  try {
    const preview = await diagnosticsStore.previewSafeCorrections();
    if (preview.summary.safeCorrectionCount <= 0) {
      uiStore.showToast({
        id: `quality-fix-empty-${Date.now()}`,
        type: ToastTypeEnum.NORMAL,
        message: "No safe local correction available.",
        timeout: 2600,
      });
      return;
    }
    const confirmed = window.confirm(
      [
        `${preview.summary.safeCorrectionCount} safe corrections on ${preview.summary.activityCount} activities.`,
        `Distance delta: ${formatSignedDistance(preview.summary.distanceDeltaMeters)}.`,
        `Elevation delta: ${formatSignedMeters(preview.summary.elevationDeltaMeters)}.`,
        preview.summary.potentiallyImpactsRecords ? "Records and statistics may change." : "",
      ].filter(Boolean).join("\n"),
    );
    if (!confirmed) {
      return;
    }
    await diagnosticsStore.applySafeCorrections();
    uiStore.showToast({
      id: `quality-fix-batch-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Safe local corrections applied.",
      timeout: 2800,
    });
  } catch (error) {
    uiStore.showToast({
      id: `quality-fix-batch-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to apply safe corrections.",
      timeout: 3600,
    });
  } finally {
    qualityBatchActionInProgress.value = false;
  }
}

async function revertCorrection(correction: DataQualityCorrection) {
  qualityActionIssueId.value = correction.issueId;
  try {
    await diagnosticsStore.revertCorrection(correction.id);
    uiStore.showToast({
      id: `quality-revert-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Local correction reverted.",
      timeout: 2600,
    });
  } catch (error) {
    uiStore.showToast({
      id: `quality-revert-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to revert local correction.",
      timeout: 3600,
    });
  } finally {
    qualityActionIssueId.value = null;
  }
}

async function refreshDiagnostics() {
  await diagnosticsStore.refreshDiagnostics();
}

async function checkRouting() {
  await diagnosticsStore.refreshDiagnostics();
}

async function copySourcePath() {
  if (!sourcePath.value) {
    return;
  }
  try {
    await navigator.clipboard.writeText(sourcePath.value);
    uiStore.showToast({
      id: `diagnostics-copy-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Cache path copied.",
      timeout: 2400,
    });
  } catch {
    uiStore.showToast({
      id: `diagnostics-copy-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: "Unable to copy cache path.",
      timeout: 3200,
    });
  }
}

async function copySourceModeCommand() {
  if (!sourceModeActivationCommand.value) {
    return;
  }
  try {
    await navigator.clipboard.writeText(sourceModeActivationCommand.value);
    uiStore.showToast({
      id: `source-mode-command-copy-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Restart command copied.",
      timeout: 2400,
    });
  } catch {
    uiStore.showToast({
      id: `source-mode-command-copy-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: "Unable to copy restart command.",
      timeout: 3200,
    });
  }
}

async function previewSourceMode() {
  try {
    const preview = await diagnosticsStore.previewSourceMode({
      mode: selectedSourceMode.value,
      path: sourceModePath.value,
    });
    uiStore.showToast({
      id: `source-mode-preview-${Date.now()}`,
      type: preview.errors.length > 0 ? ToastTypeEnum.WARN : ToastTypeEnum.NORMAL,
      message: preview.errors.length > 0 ? "Data source needs attention." : "Data source checked.",
      timeout: 2800,
    });
  } catch {
    uiStore.showToast({
      id: `source-mode-preview-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: diagnosticsStore.sourceModePreviewError || "Unable to preview data source.",
      timeout: 3600,
    });
  }
}
</script>

<template>
  <div class="diagnostics-page">
    <section class="diagnostics-toolbar">
      <div>
        <p class="diagnostics-kicker">
          Diagnostics
        </p>
        <h1>System Status</h1>
      </div>
      <div class="diagnostics-actions">
        <button
          type="button"
          class="btn btn-outline-secondary"
          :disabled="diagnosticsStore.isLoading"
          @click="checkRouting"
        >
          <i class="fa-solid fa-route" aria-hidden="true" />
          Check OSRM
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="diagnosticsStore.isLoading"
          @click="refreshDiagnostics"
        >
          <i class="fa-solid fa-rotate-right" aria-hidden="true" />
          {{ diagnosticsStore.isLoading ? "Refreshing" : "Refresh" }}
        </button>
      </div>
    </section>

    <div
      v-if="diagnosticsStore.isLoading && !diagnosticsStore.hasHealth"
      class="chart-empty"
    >
      Loading diagnostics...
    </div>

    <div
      v-else
      class="diagnostics-grid"
    >
      <section class="diagnostics-panel diagnostics-panel--summary">
        <div class="summary-main">
          <span :class="healthStatusClass">{{ healthStatusLabel }}</span>
          <h2>{{ providerLabel }}</h2>
          <p>{{ activityCount ?? 0 }} activities · {{ availableYears.length }} years</p>
        </div>
        <dl class="summary-list">
          <div>
            <dt>Health payload</dt>
            <dd>{{ timestampLabel }}</dd>
          </div>
          <div>
            <dt>Loaded in UI</dt>
            <dd>{{ loadedAtLabel }}</dd>
          </div>
          <div>
            <dt>Athlete</dt>
            <dd>{{ textValue(root.athleteId) || "n/a" }}</dd>
          </div>
        </dl>
      </section>

      <section
        v-if="hasRuntimeConfig"
        class="diagnostics-panel diagnostics-panel--wide"
      >
        <div class="panel-heading">
          <h2>Runtime Config</h2>
          <span class="status-chip status-chip--neutral">{{ textValue(runtimeCors.source) || "default" }}</span>
        </div>
        <div class="runtime-config-grid">
          <div
            v-for="item in runtimeConfigItems"
            :key="item.label"
            class="runtime-config-item"
          >
            <span>{{ item.label }}</span>
            <strong :class="{ monospace: item.monospace }">{{ item.value }}</strong>
          </div>
        </div>
      </section>

      <section class="diagnostics-panel diagnostics-panel--wide">
        <div class="panel-heading">
          <h2>Data Source</h2>
          <span :class="sourceModeStatusClass">{{ sourceModeStatusLabel }}</span>
        </div>
        <div class="source-mode-layout">
          <div class="source-mode-form">
            <div
              class="source-mode-tabs"
              role="tablist"
              aria-label="Data source mode"
            >
              <button
                v-for="option in sourceModeOptions"
                :key="option.mode"
                type="button"
                :class="['source-mode-tab', { 'source-mode-tab--active': selectedSourceMode === option.mode }]"
                :aria-selected="selectedSourceMode === option.mode"
                role="tab"
                @click="selectSourceMode(option.mode)"
              >
                <i :class="option.icon" aria-hidden="true" />
                {{ option.label }}
              </button>
            </div>
            <label class="source-path-field">
              <span>{{ sourceModePathLabel }}</span>
              <input
                v-model="sourceModePath"
                type="text"
                class="form-control"
                :placeholder="selectedSourceMode === 'STRAVA' ? 'strava-cache' : '/path/to/year-folders'"
                @input="markSourceModePathEdited"
              >
            </label>
            <div class="source-mode-actions">
              <button
                type="button"
                class="btn btn-primary"
                :disabled="diagnosticsStore.isPreviewingSourceMode"
                @click="previewSourceMode"
              >
                <i class="fa-solid fa-magnifying-glass" aria-hidden="true" />
                {{ diagnosticsStore.isPreviewingSourceMode ? "Checking" : "Preview" }}
              </button>
              <span class="source-config-key monospace">{{ sourceModeConfigKey }}</span>
            </div>
            <div class="source-activation">
              <div class="source-activation-summary">
                <div
                  v-for="item in sourceModeActivationSummary"
                  :key="item.label"
                  :class="['source-activation-item', item.tone === 'warn' ? 'source-activation-item--warn' : '', item.tone === 'up' ? 'source-activation-item--up' : '']"
                >
                  <span>{{ item.label }}</span>
                  <strong>{{ item.value }}</strong>
                </div>
              </div>
              <div
                v-if="sourceModeEnvironment.length > 0"
                class="source-env-list"
              >
                <div
                  v-for="variable in sourceModeEnvironment"
                  :key="variable.key"
                  class="source-env-row"
                >
                  <span class="monospace">{{ variable.key }}</span>
                  <strong :class="{ monospace: variable.value }">{{ variable.value || "unset" }}</strong>
                </div>
              </div>
              <div
                v-if="sourceModeActivationCommand"
                class="source-command"
              >
                <code>{{ sourceModeActivationCommand }}</code>
                <button
                  type="button"
                  class="btn btn-outline-secondary btn-sm"
                  @click="copySourceModeCommand"
                >
                  <i class="fa-solid fa-copy" aria-hidden="true" />
                  Copy
                </button>
              </div>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm source-verify-button"
                :disabled="diagnosticsStore.isLoading"
                @click="refreshDiagnostics"
              >
                <i class="fa-solid fa-circle-check" aria-hidden="true" />
                Verify active mode
              </button>
            </div>
            <p
              v-if="diagnosticsStore.sourceModePreviewError"
              class="source-mode-error"
            >
              {{ diagnosticsStore.sourceModePreviewError }}
            </p>
          </div>

          <div
            v-if="sourceModePreview"
            class="source-mode-preview"
          >
            <div class="source-preview-metrics">
              <div
                v-for="stat in sourceModePreviewStats"
                :key="stat.label"
                :class="['source-preview-metric', stat.tone === 'warn' ? 'source-preview-metric--warn' : '']"
              >
                <span>{{ stat.label }}</span>
                <strong>{{ stat.value }}</strong>
              </div>
            </div>

            <div
              v-if="sourceModePreview.years.length > 0"
              class="source-years-table"
            >
              <div class="source-years-row source-years-row--head">
                <span>Year</span>
                <span>Files</span>
                <span>Valid</span>
                <span>Activities</span>
              </div>
              <div
                v-for="year in sourceModePreview.years"
                :key="year.year"
                class="source-years-row"
              >
                <span>{{ year.year }}</span>
                <span>{{ year.fileCount }}</span>
                <span>{{ year.validFileCount }}</span>
                <span>{{ year.activityCount }}</span>
              </div>
            </div>

            <div
              v-if="sourceModePreview.missingFields.length > 0"
              class="source-chip-group"
            >
              <span
                v-for="field in sourceModePreview.missingFields"
                :key="field"
                class="source-chip source-chip--warn"
              >
                {{ formatSourceField(field) }}
              </span>
            </div>

            <div
              v-if="sourceModePreview.errors.length > 0"
              class="source-message-list source-message-list--errors"
            >
              <div
                v-for="error in sourceModePreview.errors"
                :key="`${error.path}-${error.message}`"
                class="source-message-item"
              >
                <strong>{{ error.message }}</strong>
                <span
                  v-if="error.path"
                  class="monospace"
                >{{ error.path }}</span>
              </div>
            </div>

            <div
              v-if="sourceModePreview.recommendations.length > 0"
              class="source-message-list"
            >
              <div
                v-for="recommendation in sourceModePreview.recommendations"
                :key="recommendation"
                class="source-message-item"
              >
                <span>{{ recommendation }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section
        v-if="dataQualitySummary"
        class="diagnostics-panel diagnostics-panel--wide"
      >
        <div class="panel-heading">
          <h2>Data Quality</h2>
          <span :class="dataQualityStatusClass">{{ dataQualityStatusLabel }}</span>
        </div>
        <div class="quality-layout">
          <div class="quality-overview">
            <div class="quality-metrics">
              <div
                v-for="stat in dataQualityStats"
                :key="stat.label"
                :class="dataQualityStatClass(stat.tone)"
              >
                <span>{{ stat.label }}</span>
                <strong>{{ stat.value }}</strong>
              </div>
            </div>
            <div
              v-if="dataQualityCategories.length > 0"
              class="quality-categories"
            >
              <span
                v-for="item in dataQualityCategories"
                :key="item.category"
                class="quality-category"
                :title="dataQualityCategoryTooltip(item.category)"
              >
                {{ formatDataQualityCategory(item.category) }} · {{ item.count }}
                <TooltipHint :text="dataQualityCategoryTooltip(item.category)" />
              </span>
            </div>
          </div>

          <div
            v-if="dataQualityIssues.length > 0"
            class="quality-table"
          >
            <div class="quality-table-toolbar">
              <div>
                <strong>Issue list</strong>
                <small>{{ dataQualityIssueListLabel }}</small>
              </div>
              <div class="quality-table-actions">
                <button
                  v-if="(dataQualitySummary.safeCorrectionCount ?? 0) > 0"
                  type="button"
                  class="btn btn-sm btn-primary"
                  :disabled="qualityBatchActionInProgress"
                  @click="applySafeCorrections"
                >
                  <i class="fa-solid fa-wand-magic-sparkles" aria-hidden="true" />
                  Fix safe issues
                </button>
                <button
                  v-if="canToggleDataQualityIssues"
                  type="button"
                  class="btn btn-sm btn-outline-secondary"
                  @click="showAllDataQualityIssues = !showAllDataQualityIssues"
                >
                  <i
                    :class="showAllDataQualityIssues ? 'fa-solid fa-compress' : 'fa-solid fa-list'"
                    aria-hidden="true"
                  />
                  {{ showAllDataQualityIssues ? `Show top ${dataQualityPreviewLimit}` : "Show all" }}
                </button>
              </div>
            </div>
            <div class="quality-row quality-row--head">
              <span>Severity</span>
              <span>Activity</span>
              <span>Problem</span>
              <span>Value</span>
              <span>Action</span>
            </div>
            <div
              v-for="issue in displayedDataQualityIssues"
              :key="issue.id"
              class="quality-row"
            >
              <span :class="dataQualitySeverityClass(issue.severity)">{{ issue.severity }}</span>
              <span>
                <RouterLink
                  v-if="issue.activityId"
                  :to="`/activities/${issue.activityId}`"
                  class="quality-activity-link"
                >
                  {{ issue.activityName || issue.activityId }}
                </RouterLink>
                <span v-else>{{ issue.activityName || "n/a" }}</span>
                <small>{{ [issue.activityType, issue.year].filter(Boolean).join(" · ") }}</small>
              </span>
              <span>
                <strong class="quality-problem-label">
                  {{ formatDataQualityCategory(issue.category) }}
                  <TooltipHint :text="dataQualityCategoryTooltip(issue.category)" />
                </strong>
                <small>{{ issue.message }}</small>
                <small
                  v-if="issue.correction?.available"
                  :class="['quality-correction-chip', `quality-correction-chip--${issue.correction.safety}`]"
                >
                  {{ issue.correction.safety === "safe" ? "Safe local fix" : "Manual review" }}
                </small>
              </span>
              <span class="monospace">{{ issue.rawValue || issue.field }}</span>
              <span class="quality-action-cell">
                <button
                  v-if="issue.correction?.available && issue.correction.safety === 'safe'"
                  type="button"
                  class="btn btn-sm btn-outline-primary"
                  :disabled="qualityActionIssueId === issue.id"
                  @click="applyIssueCorrection(issue)"
                >
                  <i class="fa-solid fa-wand-magic-sparkles" aria-hidden="true" />
                  Fix
                </button>
                <button
                  v-if="issue.activityId"
                  type="button"
                  :class="['btn btn-sm', issue.excludedFromStats ? 'btn-outline-secondary' : 'btn-outline-danger']"
                  :disabled="qualityActionActivityId === issue.activityId"
                  @click="toggleStatsExclusion(issue)"
                >
                  <i
                    :class="issue.excludedFromStats ? 'fa-solid fa-rotate-left' : 'fa-solid fa-ban'"
                    aria-hidden="true"
                  />
                  {{ issue.excludedFromStats ? "Include" : "Exclude" }}
                </button>
                <small v-if="issue.excludedFromStats">Excluded from stats</small>
                <small v-else>{{ issue.suggestion || "Review source" }}</small>
              </span>
            </div>
          </div>

          <div
            v-if="activeDataQualityCorrections.length > 0"
            class="quality-corrections"
          >
            <div class="quality-table-toolbar">
              <div>
                <strong>Local corrections</strong>
                <small>{{ formatInteger(activeDataQualityCorrections.length) }} active</small>
              </div>
            </div>
            <div
              v-for="correction in activeDataQualityCorrections"
              :key="correction.id"
              class="quality-correction-row"
            >
              <span>
                <strong>{{ correction.activityName || correction.activityId }}</strong>
                <small>{{ [correction.activityType, correction.year].filter(Boolean).join(" · ") }}</small>
              </span>
              <span>
                {{ correctionLabel(correction.type) }}
                <small>{{ correction.modifiedFields.join(", ") }}</small>
              </span>
              <span>
                {{ formatSignedDistance(correction.impact.distanceDeltaMeters) }}
                <small>{{ formatSignedMeters(correction.impact.elevationDeltaMeters) }}</small>
              </span>
              <button
                type="button"
                class="btn btn-sm btn-outline-secondary"
                :disabled="qualityActionIssueId === correction.issueId"
                @click="revertCorrection(correction)"
              >
                <i class="fa-solid fa-rotate-left" aria-hidden="true" />
                Undo
              </button>
            </div>
          </div>

          <div
            v-if="dataQualityIssues.length === 0"
            class="quality-empty"
          >
            {{ dataQualitySummary.status === "not_applicable" ? "Local data quality checks are available in FIT or GPX mode." : "No local data quality issue detected." }}
          </div>
        </div>
      </section>

      <section
        v-if="degradedReasons.length > 0"
        class="diagnostics-panel diagnostics-panel--wide"
      >
        <div class="panel-heading">
          <h2>Degraded Mode</h2>
          <span class="panel-count">{{ degradedReasons.length }}</span>
        </div>
        <div class="degraded-list">
          <article
            v-for="reason in degradedReasons"
            :key="reason.title"
            :class="['degraded-item', `degraded-item--${reason.tone}`]"
          >
            <strong>{{ reason.title }}</strong>
            <span>{{ reason.detail }}</span>
          </article>
        </div>
      </section>

      <section class="diagnostics-panel diagnostics-panel--wide">
        <div class="panel-heading">
          <h2>Cache</h2>
          <span :class="cacheStatusClass">{{ cacheStatusLabel }}</span>
        </div>
        <dl class="detail-list detail-list--columns">
          <div>
            <dt>Path</dt>
            <dd class="monospace">{{ sourcePath || "n/a" }}</dd>
          </div>
          <div
            v-for="item in cacheSummaryItems"
            :key="item.label"
          >
            <dt>{{ item.label }}</dt>
            <dd :class="{ monospace: item.monospace }">{{ item.value }}</dd>
          </div>
        </dl>
        <dl
          v-if="cacheManifestItems.length > 0"
          class="detail-list detail-list--columns cache-manifest-list"
        >
          <div
            v-for="item in cacheManifestItems"
            :key="item.label"
          >
            <dt>{{ item.label }}</dt>
            <dd :class="{ monospace: item.monospace }">{{ item.value }}</dd>
          </div>
        </dl>
        <button
          type="button"
          class="btn btn-outline-secondary btn-sm diagnostics-inline-button"
          :disabled="!sourcePath"
          @click="copySourcePath"
        >
          <i class="fa-solid fa-folder-open" aria-hidden="true" />
          Copy path
        </button>
      </section>

      <section class="diagnostics-panel">
        <div class="panel-heading">
          <h2>Warmup</h2>
          <span :class="warmupInProgress ? 'status-chip status-chip--warn' : 'status-chip status-chip--up'">
            {{ warmupInProgress ? "Running" : "Idle" }}
          </span>
        </div>
        <div class="warmup-steps">
          <div
            v-for="item in warmupStatusItems"
            :key="item.label"
            class="warmup-step"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
        <dl class="detail-list">
          <div>
            <dt>Prepared years</dt>
            <dd>{{ preparedYears.length > 0 ? preparedYears.join(", ") : "n/a" }}</dd>
          </div>
          <div>
            <dt>Last run</dt>
            <dd>{{ formatDateTime(textValue(warmup.lastRunAt)) }}</dd>
          </div>
          <div>
            <dt>Best efforts</dt>
            <dd>{{ bestEffortEntries.memory ?? 0 }} memory · {{ bestEffortEntries.persisted ?? 0 }} persisted</dd>
          </div>
        </dl>
      </section>

      <section class="diagnostics-panel">
        <div class="panel-heading">
          <h2>Strava API</h2>
          <span :class="rateLimitActive ? 'status-chip status-chip--warn' : 'status-chip status-chip--up'">
            {{ rateLimitActive ? "Limited" : "Ready" }}
          </span>
        </div>
        <dl class="detail-list">
          <div>
            <dt>Rate limit</dt>
            <dd>{{ rateLimitActive ? "Active" : "Inactive" }}</dd>
          </div>
          <div>
            <dt>Until</dt>
            <dd>{{ rateLimitUntilLabel }}</dd>
          </div>
          <div>
            <dt>Provider</dt>
            <dd>{{ providerLabel }}</dd>
          </div>
        </dl>
      </section>

      <section class="diagnostics-panel">
        <div class="panel-heading">
          <h2>Routing</h2>
          <span :class="routingStatusClass">{{ routingStatusLabel }}</span>
        </div>
        <dl class="detail-list">
          <div>
            <dt>Engine</dt>
            <dd>{{ textValue(routing.engine).toUpperCase() || "OSRM" }}</dd>
          </div>
          <div>
            <dt>Base URL</dt>
            <dd class="monospace">{{ textValue(routing.baseUrl) || "n/a" }}</dd>
          </div>
          <div>
            <dt>Profile</dt>
            <dd>{{ textValue(routing.effectiveProfile) || textValue(routing.profile) || "unknown" }}</dd>
          </div>
          <div>
            <dt>Route types</dt>
            <dd>{{ supportedRouteTypes.length > 0 ? supportedRouteTypes.join(", ") : "n/a" }}</dd>
          </div>
          <div v-if="textValue(routing.error)">
            <dt>Error</dt>
            <dd>{{ textValue(routing.error) }}</dd>
          </div>
        </dl>
      </section>

      <section
        v-if="cacheFiles.length > 0"
        class="diagnostics-panel diagnostics-panel--wide"
      >
        <div class="panel-heading">
          <h2>Files</h2>
          <span class="panel-count">{{ cacheFiles.length }}</span>
        </div>
        <div class="files-table">
          <div class="files-row files-row--head">
            <span>Name</span>
            <span>Status</span>
            <span>Size</span>
            <span>Modified</span>
            <span>Path</span>
          </div>
          <div
            v-for="file in cacheFiles"
            :key="file.name"
            class="files-row"
          >
            <span>{{ file.name }}</span>
            <span :class="fileStatusClass(file.exists)">{{ file.exists ? "Present" : "Missing" }}</span>
            <span>{{ file.size }}</span>
            <span>{{ file.lastModified }}</span>
            <span class="monospace">{{ file.path }}</span>
          </div>
        </div>
      </section>

      <section class="diagnostics-panel diagnostics-panel--wide">
        <div class="panel-heading">
          <h2>Raw Payload</h2>
          <button
            type="button"
            class="btn btn-outline-secondary btn-sm"
            @click="showRawPayload = !showRawPayload"
          >
            <i class="fa-solid fa-code" aria-hidden="true" />
            {{ showRawPayload ? "Hide" : "Show" }}
          </button>
        </div>
        <pre v-if="showRawPayload" class="raw-payload">{{ rawPayload }}</pre>
      </section>
    </div>
  </div>
</template>

<style scoped>
.diagnostics-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.diagnostics-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  border-bottom: 1px solid var(--ms-border);
  padding: 2px 0 12px;
}

.diagnostics-kicker {
  margin: 0 0 2px;
  color: var(--ms-primary);
  font-size: 0.74rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.diagnostics-toolbar h1 {
  margin: 0;
  font-size: 1.55rem;
}

.diagnostics-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.diagnostics-actions .btn,
.diagnostics-inline-button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.diagnostics-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.diagnostics-panel {
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #ffffff;
  box-shadow: var(--ms-shadow-soft);
  padding: 14px;
}

.diagnostics-panel--wide {
  grid-column: 1 / -1;
}

.diagnostics-panel--summary {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  border-left: 4px solid var(--ms-primary);
}

.summary-main {
  min-width: 220px;
}

.summary-main h2 {
  margin: 8px 0 2px;
  font-size: 1.35rem;
}

.summary-main p {
  margin: 0;
  color: var(--ms-text-muted);
}

.summary-list,
.detail-list {
  display: grid;
  gap: 8px;
  margin: 0;
}

.summary-list {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  flex: 1;
}

.detail-list--columns {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.cache-manifest-list {
  margin-top: 12px;
  border-top: 1px solid var(--ms-border);
  padding-top: 12px;
}

.summary-list div,
.detail-list div {
  min-width: 0;
}

dt {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 700;
}

dd {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--ms-text);
  font-weight: 700;
}

.panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
}

.panel-heading h2 {
  margin: 0;
  font-size: 1rem;
}

.panel-count,
.status-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 24px;
  border-radius: 999px;
  padding: 2px 9px;
  font-size: 0.76rem;
  font-weight: 800;
  white-space: nowrap;
}

.status-chip--up {
  border: 1px solid #99d6b0;
  background: #eaf8ef;
  color: #176a37;
}

.status-chip--warn {
  border: 1px solid #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.status-chip--down {
  border: 1px solid #efa4a4;
  background: #fff0f0;
  color: #9b1c1c;
}

.status-chip--neutral,
.panel-count {
  border: 1px solid #c8d2e1;
  background: #eef4fb;
  color: #31506f;
}

.degraded-list {
  display: grid;
  gap: 8px;
}

.degraded-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  border-radius: 8px;
  padding: 10px 12px;
}

.degraded-item strong {
  min-width: 150px;
}

.degraded-item span {
  color: var(--ms-text-muted);
  text-align: right;
}

.degraded-item--warn {
  background: #fff8e3;
  color: #805d05;
}

.degraded-item--down {
  background: #fff0f0;
  color: #9b1c1c;
}

.degraded-item--info {
  background: #eef4fb;
  color: #31506f;
}

.warmup-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 12px;
}

.warmup-step {
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  padding: 9px;
  background: #fafbfe;
}

.warmup-step span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 700;
}

.warmup-step strong {
  display: block;
  margin-top: 2px;
}

.runtime-config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.runtime-config-item {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  padding: 9px;
  background: #fafbfe;
}

.runtime-config-item span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 700;
}

.runtime-config-item strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.source-mode-layout {
  display: grid;
  grid-template-columns: minmax(260px, 0.85fr) minmax(0, 1.15fr);
  gap: 14px;
}

.source-mode-form,
.source-mode-preview {
  min-width: 0;
}

.source-mode-tabs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-bottom: 12px;
}

.source-mode-tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 38px;
  border: 1px solid #c8d2e1;
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text);
  font-weight: 800;
}

.source-mode-tab--active {
  border-color: var(--ms-primary);
  background: #fff2ea;
  color: var(--ms-primary);
}

.source-path-field {
  display: grid;
  gap: 6px;
  margin: 0;
}

.source-path-field span {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
}

.source-mode-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.source-mode-actions .btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.source-config-key {
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 7px 9px;
}

.source-activation {
  display: grid;
  gap: 9px;
  margin-top: 12px;
}

.source-activation-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
}

.source-activation-item,
.source-env-row {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 8px 9px;
}

.source-activation-item--warn {
  border-color: #f3d17e;
  background: #fff8e3;
}

.source-activation-item--up {
  border-color: #99d6b0;
  background: #eaf8ef;
}

.source-activation-item span,
.source-env-row span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
}

.source-activation-item strong,
.source-env-row strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.source-env-list {
  display: grid;
  gap: 6px;
}

.source-env-row {
  display: grid;
  grid-template-columns: minmax(120px, 0.8fr) minmax(0, 1.2fr);
  gap: 8px;
  align-items: start;
}

.source-env-row span,
.source-env-row strong {
  margin: 0;
}

.source-command {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: start;
  border: 1px solid #c8d2e1;
  border-radius: 8px;
  background: #f3f6fa;
  padding: 8px;
}

.source-command code {
  min-width: 0;
  overflow-wrap: anywhere;
  color: #1f3146;
  font-size: 0.82rem;
}

.source-command .btn,
.source-verify-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.source-mode-error {
  margin: 10px 0 0;
  border-radius: 8px;
  background: #fff0f0;
  color: #9b1c1c;
  padding: 9px 10px;
  font-weight: 700;
}

.source-preview-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.source-preview-metric {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 9px;
}

.source-preview-metric--warn {
  border-color: #f3d17e;
  background: #fff8e3;
}

.source-preview-metric span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 700;
}

.source-preview-metric strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.source-years-table {
  display: grid;
  gap: 1px;
  margin-top: 10px;
  overflow-x: auto;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
}

.source-years-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(86px, 1fr));
  gap: 8px;
  min-width: 420px;
  padding: 8px 10px;
  background: #ffffff;
}

.source-years-row--head {
  background: #f3f6fa;
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
  text-transform: uppercase;
}

.source-chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.source-chip {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  border-radius: 999px;
  padding: 2px 9px;
  font-size: 0.76rem;
  font-weight: 800;
}

.source-chip--warn {
  border: 1px solid #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.source-message-list {
  display: grid;
  gap: 6px;
  margin-top: 10px;
}

.source-message-item {
  display: grid;
  gap: 2px;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 8px 10px;
  overflow-wrap: anywhere;
}

.source-message-list--errors .source-message-item {
  border-color: #efa4a4;
  background: #fff0f0;
  color: #9b1c1c;
}

.quality-layout {
  display: grid;
  gap: 12px;
}

.quality-overview {
  display: grid;
  gap: 10px;
}

.quality-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
  gap: 8px;
}

.quality-metric {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 9px;
}

.quality-metric--warn {
  border-color: #f3d17e;
  background: #fff8e3;
}

.quality-metric--down {
  border-color: #efa4a4;
  background: #fff0f0;
}

.quality-metric span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 700;
}

.quality-metric strong {
  display: block;
  margin-top: 2px;
}

.quality-categories {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.quality-category {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  border: 1px solid #c8d2e1;
  border-radius: 999px;
  background: #eef4fb;
  color: #31506f;
  padding: 2px 9px;
  font-size: 0.76rem;
  font-weight: 800;
}

.quality-problem-label {
  display: inline-flex;
  align-items: center;
}

.quality-table {
  display: grid;
  gap: 1px;
  overflow-x: auto;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
}

.quality-table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 900px;
  padding: 9px 10px;
  background: #ffffff;
}

.quality-table-toolbar strong {
  display: block;
  font-size: 0.86rem;
}

.quality-table-toolbar small {
  display: block;
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
}

.quality-table-toolbar .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  white-space: nowrap;
}

.quality-table-actions {
  display: inline-flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.quality-row {
  display: grid;
  grid-template-columns: minmax(88px, 0.55fr) minmax(180px, 1.1fr) minmax(240px, 1.5fr) minmax(110px, 0.8fr) minmax(220px, 1.3fr);
  gap: 10px;
  min-width: 900px;
  padding: 9px 10px;
  background: #ffffff;
}

.quality-row--head {
  background: #f3f6fa;
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
  text-transform: uppercase;
}

.quality-row span {
  min-width: 0;
  overflow-wrap: anywhere;
}

.quality-row small {
  display: block;
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.quality-correction-chip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 22px;
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 0.72rem;
  font-weight: 900;
}

.quality-correction-chip--safe {
  border: 1px solid #9ec9aa;
  background: #edf8ef;
  color: #176a37;
}

.quality-correction-chip--manual {
  border: 1px solid #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.quality-activity-link {
  color: #254e7b;
  font-weight: 800;
  text-decoration: none;
}

.quality-activity-link:hover {
  color: var(--ms-primary);
}

.quality-severity {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  height: 24px;
  border-radius: 999px;
  padding: 2px 9px;
  font-size: 0.74rem;
  font-weight: 900;
  text-transform: uppercase;
}

.quality-severity--critical {
  border: 1px solid #efa4a4;
  background: #fff0f0;
  color: #9b1c1c;
}

.quality-severity--warning {
  border: 1px solid #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.quality-severity--info {
  border: 1px solid #c8d2e1;
  background: #eef4fb;
  color: #31506f;
}

.quality-action-cell {
  display: grid;
  align-content: start;
  justify-items: start;
  gap: 4px;
}

.quality-action-cell .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  font-weight: 800;
}

.quality-corrections {
  display: grid;
  gap: 1px;
  overflow-x: auto;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
}

.quality-correction-row {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(220px, 1fr) minmax(110px, 0.6fr) minmax(90px, 0.35fr);
  gap: 10px;
  min-width: 760px;
  padding: 9px 10px;
  background: #ffffff;
}

.quality-correction-row span {
  min-width: 0;
  overflow-wrap: anywhere;
}

.quality-correction-row small {
  display: block;
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
}

.quality-correction-row .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  width: fit-content;
}

.quality-empty {
  border: 1px dashed var(--ms-border);
  border-radius: 8px;
  background: #fafbfe;
  color: var(--ms-text-muted);
  padding: 12px;
  font-weight: 700;
}

.files-table {
  display: grid;
  gap: 1px;
  overflow-x: auto;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
}

.files-row {
  display: grid;
  grid-template-columns: minmax(120px, 0.8fr) minmax(90px, 0.6fr) minmax(80px, 0.5fr) minmax(160px, 0.9fr) minmax(220px, 1.6fr);
  gap: 10px;
  min-width: 760px;
  padding: 9px 10px;
  background: #ffffff;
}

.files-row--head {
  background: #f3f6fa;
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
  text-transform: uppercase;
}

.file-state {
  font-weight: 800;
}

.file-state--ok {
  color: #176a37;
}

.file-state--missing {
  color: #9b1c1c;
}

.monospace {
  font-family: "SFMono-Regular", "Cascadia Code", "Liberation Mono", monospace;
  font-size: 0.86rem;
}

.raw-payload {
  max-height: 420px;
  overflow: auto;
  margin: 0;
  border-radius: 8px;
  background: #15202b;
  color: #e7edf3;
  padding: 12px;
  font-size: 0.82rem;
}

@media (max-width: 992px) {
  .diagnostics-toolbar,
  .diagnostics-panel--summary {
    align-items: stretch;
    flex-direction: column;
  }

  .diagnostics-actions {
    justify-content: flex-start;
  }

  .diagnostics-grid,
  .summary-list,
  .detail-list--columns,
  .warmup-steps,
  .runtime-config-grid,
  .quality-metrics,
  .source-mode-layout,
  .source-activation-summary,
  .source-env-row,
  .source-command,
  .source-preview-metrics {
    grid-template-columns: 1fr;
  }

  .degraded-item {
    flex-direction: column;
  }

  .degraded-item span {
    text-align: left;
  }
}
</style>

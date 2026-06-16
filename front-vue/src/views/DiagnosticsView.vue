<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useContextStore } from "@/stores/context";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import TooltipHint from "@/components/TooltipHint.vue";
import { ToastTypeEnum } from "@/models/toast.model";
import type { DataQualityCorrection, DataQualityCorrectionPreview, DataQualityIssue, DataQualitySummary } from "@/models/data-quality.model";
import type { HealthRecord } from "@/models/health.model";
import type { SourceMode, StravaOAuthStatus } from "@/models/source-mode.model";
import { RouterLink } from "vue-router";

const contextStore = useContextStore();
const diagnosticsStore = useDiagnosticsStore();
const uiStore = useUiStore();
const showRawPayload = ref(false);
const sourceModePathEdited = ref(false);
const sourceModeInitialized = ref(false);
const lastSourceModeAutoPreviewKey = ref("");
const stravaClientIdInput = ref("");
const stravaClientSecretInput = ref("");
const stravaUseCacheInput = ref(false);
const stravaEnrollmentInProgress = ref(false);
const qualityActionActivityId = ref<number | null>(null);
const qualityActionIssueId = ref<string | null>(null);
const qualityBatchActionInProgress = ref(false);
const dataQualitySeverityFilter = ref("all");
const dataQualityActivityFilter = ref("");
const dataQualityFieldFilter = ref("all");
const dataQualityImpactFilter = ref("all");
const dataQualityActionFilter = ref("all");
const selectedQualityIssueId = ref<string | null>(null);
const issueCorrectionPreview = ref<DataQualityCorrectionPreview | null>(null);
const issueCorrectionPreviewLoading = ref(false);
const issueCorrectionPreviewError = ref("");
const safeCorrectionPreview = ref<DataQualityCorrectionPreview | null>(null);
const showSafeCorrectionPreview = ref(false);
const showAllDataQualityIssues = ref(false);
const dataQualityPreviewLimit = 12;
const selectedSourceMode = ref<SourceMode>("STRAVA");
const sourceModePath = ref("");
const sourceModeOptions: Array<{ mode: SourceMode; label: string; icon: string }> = [
  { mode: "STRAVA", label: "Strava", icon: "fa-brands fa-strava" },
  { mode: "FIT", label: "FIT", icon: "fa-solid fa-file-lines" },
  { mode: "GPX", label: "GPX", icon: "fa-solid fa-route" },
];
type GuideFactTone = "warn" | "up" | "down" | "neutral";
type GuideFact = { label: string; value: string; tone?: GuideFactTone; monospace?: boolean };
type GuideStepState = "complete" | "current" | "warn" | "pending";
type GuideStep = { key: string; title: string; detail: string; state: GuideStepState; icon: string };
type StatusTone = "up" | "warn" | "down" | "neutral" | "info";
type StatusActionKey = "start-osrm" | "check-osrm" | "safe-fixes" | "review-quality" | "review-source" | "synchronize" | "refresh";
type StatusActionTone = "primary" | "warn" | "neutral";
type StatusAction = { key: StatusActionKey; label: string; detail: string; icon: string; tone: StatusActionTone; disabled?: boolean };
type StatusItem = { label: string; value: string; detail?: string; tone: StatusTone };

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
const composite = computed(() => asRecord(root.value.composite));
const sourceSync = computed(() => asRecord(root.value.sourceSync));
const fitSourceSync = computed(() => asRecord(sourceSync.value.fit));
const fitDeviceSync = computed(() => asRecord(fitSourceSync.value.deviceSync || fitSourceSync.value.syncModule));
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
const previewedSafeCorrections = computed(() => safeCorrectionPreview.value?.corrections.slice(0, 8) ?? []);
const safeCorrectionPreviewOverflowCount = computed(() => Math.max((safeCorrectionPreview.value?.corrections.length ?? 0) - previewedSafeCorrections.value.length, 0));
const hasFullDataQualityReport = computed(() => diagnosticsStore.dataQualityReport !== null);
const dataQualityFieldOptions = computed(() => {
  const fields = new Map<string, string>();
  for (const issue of dataQualityIssues.value) {
    const value = issue.field || issue.category;
    if (!value) continue;
    fields.set(value, `${formatSourceField(value)} · ${formatDataQualityCategory(issue.category)}`);
  }
  return Array.from(fields.entries())
    .map(([value, label]) => ({ value, label }))
    .sort((left, right) => left.label.localeCompare(right.label));
});
const dataQualityActionGroups = computed(() => {
  const groups = [
    { key: "safe", label: "Safe fixes", icon: "fa-solid fa-wand-magic-sparkles", count: 0 },
    { key: "manual", label: "Manual review", icon: "fa-solid fa-hand", count: 0 },
    { key: "unsupported", label: "Unsupported", icon: "fa-solid fa-ban", count: 0 },
  ];
  for (const issue of dataQualityIssues.value) {
    const group = groups.find((item) => item.key === dataQualityActionForIssue(issue));
    if (group) group.count += 1;
  }
  return groups;
});
const hasDataQualityFilters = computed(() =>
  dataQualitySeverityFilter.value !== "all"
  || dataQualityActivityFilter.value.trim() !== ""
  || dataQualityFieldFilter.value !== "all"
  || dataQualityImpactFilter.value !== "all"
  || dataQualityActionFilter.value !== "all",
);
const filteredDataQualityIssues = computed(() => dataQualityIssues.value.filter((issue) => {
  if (dataQualitySeverityFilter.value !== "all" && issue.severity !== dataQualitySeverityFilter.value) {
    return false;
  }
  const activityQuery = dataQualityActivityFilter.value.trim().toLowerCase();
  if (activityQuery) {
    const haystack = [
      issue.activityName,
      issue.activityId,
      issue.activityType,
      issue.year,
      issue.filePath,
    ].filter(Boolean).join(" ").toLowerCase();
    if (!haystack.includes(activityQuery)) return false;
  }
  if (dataQualityFieldFilter.value !== "all" && issue.field !== dataQualityFieldFilter.value && issue.category !== dataQualityFieldFilter.value) {
    return false;
  }
  if (dataQualityActionFilter.value !== "all" && dataQualityActionForIssue(issue) !== dataQualityActionFilter.value) {
    return false;
  }
  if (dataQualityImpactFilter.value !== "all" && !dataQualityImpactTokens(issue).includes(dataQualityImpactFilter.value)) {
    return false;
  }
  return true;
}));
const displayedDataQualityIssues = computed(() => {
  if (showAllDataQualityIssues.value) {
    return filteredDataQualityIssues.value;
  }
  return filteredDataQualityIssues.value.slice(0, dataQualityPreviewLimit);
});
const dataQualityIssueListLabel = computed(() => {
  const visible = displayedDataQualityIssues.value.length;
  const filtered = filteredDataQualityIssues.value.length;
  const total = hasFullDataQualityReport.value
    ? dataQualityIssues.value.length
    : dataQualitySummary.value?.issueCount ?? dataQualityIssues.value.length;
  if (total === filtered && total <= visible) {
    return `${formatInteger(visible)} issues`;
  }
  if (total === filtered) {
    return `Showing ${formatInteger(visible)} of ${formatInteger(total)} issues`;
  }
  if (filtered <= visible) {
    return `${formatInteger(filtered)} filtered issues out of ${formatInteger(total)} total`;
  }
  return `Showing ${formatInteger(visible)} of ${formatInteger(filtered)} filtered issues`;
});
const canToggleDataQualityIssues = computed(() => hasFullDataQualityReport.value && filteredDataQualityIssues.value.length > dataQualityPreviewLimit);
const issueCorrection = computed(() => issueCorrectionPreview.value?.corrections[0] ?? null);
const issueCorrectionImpactStats = computed(() => {
  const impact = issueCorrection.value?.impact;
  return [
    {
      label: "Distance",
      before: formatDistanceMeters(impact?.distanceMetersBefore),
      after: formatDistanceMeters(impact?.distanceMetersAfter),
      delta: formatSignedDistance(impact?.distanceDeltaMeters),
    },
    {
      label: "Elevation",
      before: formatMeters(impact?.elevationMetersBefore),
      after: formatMeters(impact?.elevationMetersAfter),
      delta: formatSignedMeters(impact?.elevationDeltaMeters),
    },
    {
      label: "Max speed",
      before: formatSpeed(impact?.maxSpeedBefore),
      after: formatSpeed(impact?.maxSpeedAfter),
      delta: formatSignedSpeedDelta(impact?.maxSpeedBefore, impact?.maxSpeedAfter),
    },
    {
      label: "Records",
      before: "n/a",
      after: "n/a",
      delta: issueCorrectionPreview.value?.summary.potentiallyImpactsRecords ? "May change" : "No signal",
    },
  ];
});
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
    { label: "Manual review", value: formatInteger(summary?.manualReviewCount ?? 0), tone: (summary?.manualReviewCount ?? 0) > 0 ? "warn" : "neutral" },
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
const osrmControlEnabled = computed(() =>
  runtimeRouting.value.controlEnabled === undefined ? true : booleanValue(runtimeRouting.value.controlEnabled),
);
const osrmStartDisabledReason = computed(() => {
  if (!osrmControlEnabled.value) return "OSRM control is disabled.";
  if (routingStatus.value === "up" && routingReachable.value) return "OSRM is already online.";
  if (routingStatus.value === "disabled") return "OSRM routing is disabled.";
  if (diagnosticsStore.isStartingOsrm) return "Starting OSRM...";
  if (diagnosticsStore.isLoading) return "Diagnostics are refreshing.";
  return "";
});
const canStartOsrm = computed(() => osrmStartDisabledReason.value === "");
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
  { label: "Annual summary", value: textValue(warmup.value.priority1) || "n/a" },
  { label: "Primary records", value: textValue(warmup.value.priority2) || "n/a" },
  { label: "Advanced metrics", value: textValue(warmup.value.priority3) || "n/a" },
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
const activeProviderValue = computed(() => textValue(root.value.provider) || textValue(runtimeData.value.provider));
const activeConfiguredProviders = computed(() => {
  const compositeProviders = listValues(composite.value.activeProviders);
  if (compositeProviders.length > 0) return compositeProviders;
  return listValues(runtimeData.value.activeProviders);
});
const isCompositeProvider = computed(() => activeProviderValue.value.toLowerCase() === "composite" || booleanValue(composite.value.active));
const compositeSourceRows = computed(() =>
  recordList(composite.value.sources).map((source) => ({
    provider: formatProvider(textValue(source.provider)),
    athleteId: textValue(source.athleteId) || "n/a",
    cacheRoot: textValue(source.cacheRoot) || "n/a",
    activities: formatInteger(numberValue(source.activities)),
    years: displayList(source.availableYearBins),
  })),
);
const compositeConflictRows = computed(() =>
  recordList(composite.value.conflictSamples).map((conflict) => formatCompositeConflict(conflict)),
);
const compositeSummaryItems = computed<Array<{ label: string; value: string; tone?: "warn" | "up" | "neutral" }>>(() => [
  { label: "Active sources", value: activeConfiguredProviders.value.map(formatProvider).join(", ") || "n/a", tone: "up" },
  { label: "Merged", value: formatInteger(numberValue(composite.value.matchedActivities)), tone: "up" },
  { label: "Local only", value: formatInteger(numberValue(composite.value.localOnlyActivities)), tone: "neutral" },
  { label: "Conflicts", value: formatInteger(numberValue(composite.value.conflictCount)), tone: (numberValue(composite.value.conflictCount) ?? 0) > 0 ? "warn" : "up" },
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
  { label: "Strava API base", value: textValue(runtimeData.value.stravaApiBaseUrl) || "n/a", monospace: true },
  { label: "FIT files", value: textValue(runtimeData.value.fitFilesPath) || "n/a", monospace: true },
  { label: "FIT inbox", value: textValue(runtimeData.value.fitInboxPath) || "n/a", monospace: true },
  { label: "Garmin FIT source", value: textValue(runtimeData.value.garminFitSourcePath) || "Auto-detect", monospace: true },
  { label: "GPX files", value: textValue(runtimeData.value.gpxFilesPath) || "n/a", monospace: true },
  { label: "CORS origins", value: displayList(runtimeCors.value.allowedOrigins), monospace: true },
  { label: "CORS headers", value: displayList(runtimeCors.value.allowedHeaders), monospace: true },
  { label: "CORS credentials", value: yesNo(runtimeCors.value.allowCredentials), monospace: false },
  { label: "Listen", value: runtimeListenAddress.value, monospace: true },
  { label: "OSRM base URL", value: textValue(runtimeRouting.value.baseUrl) || "n/a", monospace: true },
  { label: "OSRM enabled", value: yesNo(runtimeRouting.value.enabled), monospace: false },
  { label: "OSRM control", value: yesNo(osrmControlEnabled.value), monospace: false },
  { label: "History bias", value: yesNo(runtimeRouting.value.historyBiasEnabled), monospace: false },
]);
const sourceModePreview = computed(() => diagnosticsStore.sourceModePreview);
const sourceSyncStatusLabel = computed(() => {
  const status = textValue(sourceSync.value.status).toLowerCase();
  if (diagnosticsStore.isSynchronizingSources || status === "running") return "Running";
  if (status === "completed") return "OK";
  if (status === "failed") return "Failed";
  if (status === "skipped") return "Skipped";
  return "Idle";
});
const sourceSyncStatusClass = computed(() => {
  const status = textValue(sourceSync.value.status).toLowerCase();
  if (diagnosticsStore.isSynchronizingSources || status === "running") return "status-chip status-chip--warn";
  if (status === "completed") return "status-chip status-chip--up";
  if (status === "failed") return "status-chip status-chip--down";
  return "status-chip status-chip--neutral";
});
const sourceSyncSummary = computed(() => textValue(sourceSync.value.message) || "No synchronization run yet.");
const sourceSyncItems = computed<Array<{ label: string; value: string; tone?: "warn" | "up" | "neutral"; monospace?: boolean }>>(() => {
  const importedFiles = numberValue(fitSourceSync.value.importedFiles) ?? 0;
  const invalidFiles = numberValue(fitSourceSync.value.invalidFiles) ?? 0;
  const deviceStatus = textValue(fitDeviceSync.value.status);
  const deviceCopiedFiles = numberValue(fitDeviceSync.value.copiedFiles) ?? 0;
  return [
    { label: "Last run", value: formatDateTime(textValue(sourceSync.value.completedAt)), tone: "neutral" },
    { label: "Source type", value: formatFITSourceKind(textValue(fitSourceSync.value.sourceKind)), tone: textValue(fitSourceSync.value.sourceKind) ? "neutral" : "warn" },
    { label: "Source", value: textValue(fitSourceSync.value.sourcePath) || "No FIT inbox or Garmin source", monospace: true, tone: textValue(fitSourceSync.value.sourcePath) ? "neutral" : "warn" },
    { label: "Inbox", value: textValue(fitSourceSync.value.inboxPath) || textValue(runtimeData.value.fitInboxPath) || "n/a", monospace: true },
    { label: "Device sync", value: deviceStatus ? `${formatFITDeviceSyncStatus(deviceStatus)} · ${textValue(fitDeviceSync.value.backend) || "filesystem"}` : "n/a", tone: deviceStatus === "failed" ? "warn" : "neutral" },
    { label: "Copied to inbox", value: formatInteger(deviceCopiedFiles), tone: deviceCopiedFiles > 0 ? "up" : "neutral" },
    { label: "Destination", value: textValue(fitSourceSync.value.destinationPath) || textValue(runtimeData.value.fitFilesPath) || "n/a", monospace: true },
    { label: "Scanned", value: formatInteger(numberValue(fitSourceSync.value.scannedFiles)) },
    { label: "Imported", value: formatInteger(importedFiles), tone: importedFiles > 0 ? "up" : "neutral" },
    { label: "Already present", value: formatInteger(numberValue(fitSourceSync.value.alreadyPresentFiles)) },
    { label: "Invalid", value: formatInteger(invalidFiles), tone: invalidFiles > 0 ? "warn" : "neutral" },
    { label: "Year folders", value: displayList(fitSourceSync.value.createdYearDirectories) },
  ];
});
const sourceSyncKeyItems = computed(() => {
  const priority = new Set(["Last run", "Source type", "Device sync", "Imported", "Invalid"]);
  return sourceSyncItems.value.filter((item) => priority.has(item.label));
});
const activeSourceMode = computed(() => normalizeSourceMode(activeProviderValue.value));
const selectedSourceIsActive = computed(() => {
  if (isCompositeProvider.value) {
    return activeConfiguredProviders.value.map((provider) => provider.toUpperCase()).includes(selectedSourceMode.value);
  }
  return activeSourceMode.value === selectedSourceMode.value;
});
const stravaOAuth = computed<StravaOAuthStatus | null>(() => selectedSourceMode.value === "STRAVA" ? sourceModePreview.value?.stravaOAuth ?? null : null);
const stravaSettingsHref = computed(() => stravaOAuth.value?.settingsUrl || "https://www.strava.com/settings/api");
const stravaEnrollmentCommand = computed(() => stravaOAuth.value?.setupCommand || stravaSetupCommand(sourceModePath.value));
const stravaEnrollmentActionLabel = computed(() => {
  if (stravaEnrollmentInProgress.value) return stravaUseCacheInput.value ? "Saving" : "Starting";
  return stravaUseCacheInput.value ? "Save cache mode" : "Start OAuth";
});
const stravaEnrollmentStatusLabel = computed(() => {
  const status = stravaOAuth.value?.status;
  if (!sourceModePreview.value) return "Check needed";
  if (status === "ready" || status === "ready_unverified_scopes") return "Ready";
  if (status === "cache_only") return "Cache only";
  if (status === "refreshable") return "Refreshable";
  if (status === "scope_incomplete") return "Scopes missing";
  if (status === "token_unreadable" || status === "token_incomplete" || status === "token_expired") return "Token issue";
  if (status === "needs_token") return "OAuth needed";
  return "Credentials needed";
});
const stravaEnrollmentStatusClass = computed(() => {
  const status = stravaOAuth.value?.status;
  if (status === "ready" || status === "ready_unverified_scopes" || status === "cache_only") return "status-chip status-chip--up";
  if (status === "refreshable" || status === "needs_token" || status === "scope_incomplete") return "status-chip status-chip--warn";
  if (status === "token_unreadable" || status === "token_incomplete" || status === "token_expired") return "status-chip status-chip--down";
  return "status-chip status-chip--neutral";
});
const stravaRequiredFields = computed(() => [
  { label: "Client ID", value: stravaOAuth.value?.clientIdPresent ? "Present" : "Required" },
  { label: "Client Secret", value: stravaOAuth.value?.clientSecretPresent ? "Present" : "Required" },
  { label: "Authorization Callback Domain", value: stravaOAuth.value?.callbackDomain || "127.0.0.1" },
]);
const stravaClientIdPlaceholder = computed(() => stravaOAuth.value?.clientIdPresent ? "Reuse saved Client ID" : "Client ID");
const stravaClientSecretPlaceholder = computed(() => stravaOAuth.value?.clientSecretPresent ? "Reuse saved Client Secret" : "Client Secret");
const stravaSavedCredentialsHint = computed(() => {
  const status = stravaOAuth.value;
  if (!sourceModePreview.value) return "Check this cache before starting OAuth.";
  if (status?.credentialsPresent) return "Existing .strava credentials will be reused when these fields are empty.";
  if (status?.credentialsFilePresent) return "A .strava file exists, but at least one credential is missing.";
  return "No .strava credentials detected yet.";
});
const stravaSavedCredentialsHintClass = computed(() => {
  const status = stravaOAuth.value;
  if (status?.credentialsPresent) return "strava-saved-hint strava-saved-hint--ready";
  if (status?.credentialsFilePresent) return "strava-saved-hint strava-saved-hint--warn";
  return "strava-saved-hint";
});
const canStartStravaEnrollment = computed(() => {
  if (stravaEnrollmentInProgress.value || selectedSourceMode.value !== "STRAVA") return false;
  if (!sourceModePath.value.trim()) return false;
  const hasSavedCredentials = stravaOAuth.value?.credentialsPresent ?? false;
  if (stravaUseCacheInput.value) {
    return stravaClientIdInput.value.trim() !== "" || stravaOAuth.value?.clientIdPresent === true;
  }
  return hasSavedCredentials || (stravaClientIdInput.value.trim() !== "" && stravaClientSecretInput.value.trim() !== "");
});
const stravaEnrollmentSteps = computed<Array<{ key: string; title: string; detail: string; state: "complete" | "current" | "warn" | "pending"; icon: string }>>(() => {
  const status = stravaOAuth.value;
  const preview = sourceModePreview.value;
  const credentialsReady = status?.credentialsPresent ?? false;
  const tokenReady = status !== null && ["ready", "ready_unverified_scopes", "refreshable", "cache_only"].includes(status.status);
  const tokenWarn = status !== null && ["needs_token", "scope_incomplete"].includes(status.status);
  const tokenDown = status !== null && ["token_unreadable", "token_incomplete", "token_expired"].includes(status.status);
  return [
    {
      key: "app",
      title: "Strava app",
      detail: credentialsReady ? "Client credentials are available locally." : "Create the app and set the callback domain.",
      state: credentialsReady ? "complete" : "current",
      icon: "fa-solid fa-id-card",
    },
    {
      key: "credentials",
      title: ".strava",
      detail: status?.credentialsFile || "Run the setup assistant to write the credentials file.",
      state: credentialsReady ? "complete" : status?.credentialsFilePresent ? "warn" : "pending",
      icon: "fa-solid fa-key",
    },
    {
      key: "oauth",
      title: "OAuth token",
      detail: status?.message || "Check the Strava cache to inspect OAuth readiness.",
      state: tokenReady ? "complete" : tokenDown ? "warn" : tokenWarn ? "current" : "pending",
      icon: "fa-solid fa-shield-halved",
    },
    {
      key: "activate",
      title: "Active source",
      detail: preview?.active ? "Backend is already using this Strava cache." : preview?.restartNeeded ? "Use this source, then restart normally." : "Check the source, then verify active mode.",
      state: preview?.active ? "complete" : preview ? "warn" : "pending",
      icon: "fa-solid fa-circle-check",
    },
  ];
});
const stravaOAuthFacts = computed<Array<{ label: string; value: string; tone?: "warn" | "down" }>>(() => {
  const status = stravaOAuth.value;
  if (!status) return [];
  return [
    { label: "Credentials", value: status.credentialsPresent ? "Ready" : "Missing", tone: status.credentialsPresent ? undefined : "warn" },
    { label: "Token", value: tokenStatusLabel(status), tone: tokenTone(status) },
    { label: "Scopes", value: scopesStatusLabel(status), tone: status.missingScopes.length > 0 ? "warn" : undefined },
    { label: "Athlete", value: status.athleteName || status.athleteId || "n/a" },
  ];
});
const sourceModePathLabel = computed(() => {
  if (selectedSourceMode.value === "STRAVA") return "Cache path";
  return `${selectedSourceMode.value} directory`;
});
const sourceModeStatusLabel = computed(() => {
  const preview = sourceModePreview.value;
  if (diagnosticsStore.sourceModePreviewError) return "Unavailable";
  if (diagnosticsStore.isPreviewingSourceMode) return "Checking";
  if (!preview) return "Not checked";
  if (preview.active) return "Active";
  if (!preview.supported || preview.errors.length > 0 || !preview.readable || !preview.validStructure) return "Needs attention";
  if (sourceModeSavedForNextStart.value) return "Restart needed";
  if (preview.restartNeeded) return "Ready after restart";
  return "Ready";
});
const sourceModeStatusClass = computed(() => {
  const preview = sourceModePreview.value;
  if (diagnosticsStore.sourceModePreviewError) return "status-chip status-chip--down";
  if (!preview) return "status-chip status-chip--neutral";
  if (preview.active) return "status-chip status-chip--up";
  if (!preview.supported || preview.errors.length > 0 || !preview.readable || !preview.validStructure) return "status-chip status-chip--warn";
  if (sourceModeSavedForNextStart.value) return "status-chip status-chip--warn";
  if (preview.restartNeeded) return "status-chip status-chip--warn";
  return "status-chip status-chip--up";
});
const dataSourceStatusLabel = computed(() => {
  if (isCompositeProvider.value) return "Composite active";
  return sourceModeStatusLabel.value;
});
const dataSourceStatusClass = computed(() => {
  if (isCompositeProvider.value) return "status-chip status-chip--up";
  return sourceModeStatusClass.value;
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
const localSourceGuideTitle = computed(() => `Import ${selectedSourceMode.value}`);
const localSourceGuideCopy = computed(() => {
  if (selectedSourceMode.value === "FIT") return "Local FIT files grouped by year.";
  if (selectedSourceMode.value === "GPX") return "Local GPX files grouped by year.";
  return "";
});
const localSourceGuideFacts = computed<GuideFact[]>(() => {
  const preview = sourceModePreview.value;
  if (!preview) {
    return [
      { label: "Path", value: sourceModePath.value.trim() ? "Set" : "Required", tone: sourceModePath.value.trim() ? undefined : "warn" },
      { label: "Saved", value: sourceModeSavedForNextStart.value ? "Pending restart" : "No", tone: sourceModeSavedForNextStart.value ? "warn" : "neutral" },
      { label: "Verification", value: diagnosticsStore.isPreviewingSourceMode ? "Checking" : "Check first", tone: "warn" },
    ];
  }
  return [
    { label: "Activities", value: formatInteger(preview.activityCount), tone: preview.activityCount > 0 ? "up" : "warn" },
    { label: "Years", value: preview.years.length > 0 ? preview.years.map((year) => year.year).join(", ") : "n/a", tone: preview.years.length > 0 ? undefined : "warn" },
    { label: "Files", value: `${formatInteger(preview.validFileCount)}/${formatInteger(preview.fileCount)}`, tone: preview.validFileCount > 0 ? "up" : "warn" },
    { label: "Invalid", value: formatInteger(preview.invalidFileCount), tone: preview.invalidFileCount > 0 ? "warn" : undefined },
  ];
});
const localSourceGuideNextAction = computed(() => {
  const preview = sourceModePreview.value;
  if (!sourceModePath.value.trim()) return `Select a ${selectedSourceMode.value} directory to inspect.`;
  if (diagnosticsStore.isPreviewingSourceMode) return "Checking the selected directory.";
  if (!preview) return "Check the selected directory before saving it.";
  if (preview.errors.length > 0) return preview.errors[0]?.message ?? "Fix the reported source issue.";
  if (!preview.readable) return "Choose a readable directory.";
  if (!preview.validStructure) return "Fix the directory structure before activating this source.";
  if (preview.invalidFileCount > 0) return "Remove or fix invalid files before relying on this source.";
  if (sourceModeSavedForNextStart.value && preview.restartNeeded) return "Restart the backend normally, then verify active mode.";
  if (preview.restartNeeded) return "Use this source, then restart the backend normally.";
  if (preview.active) return `${selectedSourceMode.value} is active and ready.`;
  return "Check is ready; verify active mode before switching workflows.";
});
const localSourceGuideNextActionClass = computed(() => {
  const preview = sourceModePreview.value;
  if (preview?.active) return "source-guide-next source-guide-next--ready";
  if (preview && preview.errors.length === 0 && preview.readable && preview.validStructure && preview.invalidFileCount === 0) return "source-guide-next source-guide-next--ready";
  if (diagnosticsStore.isPreviewingSourceMode) return "source-guide-next";
  return "source-guide-next source-guide-next--warn";
});
const localSourceGuideSteps = computed<GuideStep[]>(() => {
  const preview = sourceModePreview.value;
  const hasPath = sourceModePath.value.trim().length > 0;
  const sourceLabel = selectedSourceMode.value;
  return [
    {
      key: "folder",
      title: `${sourceLabel} directory`,
      detail: hasPath ? sourceModePath.value : "Directory path is required.",
      state: preview?.readable ? "complete" : hasPath ? "pending" : "current",
      icon: "fa-solid fa-folder-open",
    },
    {
      key: "files",
      title: "Check",
      detail: preview ? `${formatInteger(preview.validFileCount)} valid file(s), ${formatInteger(preview.invalidFileCount)} invalid.` : "Check files before saving.",
      state: preview ? preview.errors.length > 0 || !preview.validStructure || preview.validFileCount === 0 ? "warn" : "complete" : "pending",
      icon: "fa-solid fa-file-lines",
    },
    {
      key: "restart",
      title: "Save",
      detail: sourceModeSavedForNextStart.value ? "Saved for the next backend start." : preview?.active ? "Backend is already using this source." : sourceModeCheckIsReady.value ? "Ready to save for the next backend start." : "Use this source after a successful check.",
      state: sourceModeSavedForNextStart.value || preview?.active ? "complete" : preview ? "current" : "pending",
      icon: "fa-solid fa-floppy-disk",
    },
    {
      key: "verify",
      title: "Active source",
      detail: preview?.active ? "Active mode verified." : "Restart if needed, then verify active mode.",
      state: preview?.active ? "complete" : preview ? "current" : "pending",
      icon: "fa-solid fa-circle-check",
    },
  ];
});
const dataQualityProviderMode = computed<SourceMode | null>(() => {
  const provider = dataQualitySummary.value?.provider?.trim();
  return provider ? normalizeSourceMode(provider) : null;
});
const selectedLocalSourceIsActive = computed(() => selectedSourceMode.value !== "STRAVA" && selectedSourceIsActive.value);
const dataQualityMatchesSelectedSource = computed(() => {
  if (!selectedLocalSourceIsActive.value) return false;
  return dataQualityProviderMode.value === null || dataQualityProviderMode.value === selectedSourceMode.value;
});
const localSourceQualityStatusLabel = computed(() => {
  if (!selectedLocalSourceIsActive.value) return "After restart";
  const status = dataQualitySummary.value?.status ?? "not_applicable";
  if (status === "not_applicable") return "Not checked";
  return dataQualityStatusLabel.value;
});
const localSourceQualityStatusClass = computed(() => {
  if (!selectedLocalSourceIsActive.value) return "status-chip status-chip--neutral";
  const status = dataQualitySummary.value?.status ?? "not_applicable";
  if (status === "not_applicable") return "status-chip status-chip--neutral";
  return dataQualityStatusClass.value;
});
const localSourceQualityCopy = computed(() => {
  const preview = sourceModePreview.value;
  const summary = dataQualitySummary.value;
  if (!selectedLocalSourceIsActive.value) {
    if (sourceModeSavedForNextStart.value) return "Restart the backend normally to run data quality on this import.";
    if (preview?.restartNeeded) return "Use this source to run data quality on this import after restart.";
    return `Data quality will attach here once ${selectedSourceMode.value} is the active source.`;
  }
  if (!dataQualityMatchesSelectedSource.value || !summary || summary.status === "not_applicable") {
    return "Refresh diagnostics after activation to load local data quality checks.";
  }
  if (summary.issueCount <= 0) {
    return `No local data quality issue detected for the active ${selectedSourceMode.value} source.`;
  }
  return `${formatInteger(summary.issueCount)} issue(s) affect ${formatInteger(summary.impactedActivities)} activity(ies); review them before trusting records.`;
});
const localSourceQualityFacts = computed<GuideFact[]>(() => {
  const summary = dataQualitySummary.value;
  if (!selectedLocalSourceIsActive.value) {
    return [
      { label: "Quality", value: "Pending" },
      { label: "Current", value: formatProvider(activeSourceMode.value), tone: activeSourceMode.value === "STRAVA" ? "neutral" : "warn" },
      { label: "Target", value: selectedSourceMode.value, tone: "up" },
      { label: "Next", value: sourceModeSavedForNextStart.value ? "Restart" : sourceModePreview.value?.restartNeeded ? "Use source" : "Check", tone: "warn" },
    ];
  }
  if (!dataQualityMatchesSelectedSource.value || !summary || summary.status === "not_applicable") {
    return [
      { label: "Quality", value: "Not checked", tone: "warn" },
      { label: "Active", value: selectedSourceMode.value, tone: "up" },
      { label: "Issues", value: "n/a" },
      { label: "Next", value: "Refresh", tone: "warn" },
    ];
  }
  const statusTone = summary.status === "critical" ? "down" : summary.issueCount > 0 ? "warn" : "up";
  return [
    { label: "Issues", value: formatInteger(summary.issueCount), tone: statusTone },
    { label: "Affected", value: formatInteger(summary.impactedActivities), tone: summary.impactedActivities > 0 ? "warn" : "up" },
    { label: "Safe fixes", value: formatInteger(summary.safeCorrectionCount ?? 0), tone: (summary.safeCorrectionCount ?? 0) > 0 ? "warn" : undefined },
    { label: "Manual", value: formatInteger(summary.manualReviewCount ?? 0), tone: (summary.manualReviewCount ?? 0) > 0 ? "warn" : undefined },
  ];
});
const localSourceQualityCategories = computed(() => dataQualityMatchesSelectedSource.value ? dataQualityCategories.value.slice(0, 4) : []);
const localSourceQualityTopIssues = computed(() => dataQualityMatchesSelectedSource.value ? dataQualityIssues.value.slice(0, 3) : []);
const localSourceQualityOverflowCount = computed(() => Math.max(dataQualityIssues.value.length - localSourceQualityTopIssues.value.length, 0));
const localSourceQualityCanFix = computed(() =>
  dataQualityMatchesSelectedSource.value && (dataQualitySummary.value?.safeCorrectionCount ?? 0) > 0,
);
const sourceModeActivationCommand = computed(() => sourceModePreview.value?.activationCommand ?? "");
const sourceModeSavedForNextStart = computed(() => {
  if (selectedSourceIsActive.value) return null;
  const result = diagnosticsStore.sourceModeApplyResult;
  if (!result) return null;
  const preview = result.preview;
  if (preview.mode !== selectedSourceMode.value) return null;
  if (preview.path.trim() !== sourceModePath.value.trim()) return null;
  return result;
});
const sourceModeCheckIsReady = computed(() => {
  const preview = sourceModePreview.value;
  return Boolean(preview && preview.supported && preview.readable && preview.validStructure && preview.errors.length === 0);
});
const sourceModeNextStep = computed(() => {
  if (selectedSourceIsActive.value && !sourceModeSavedForNextStart.value) return "Ready";
  if (sourceModeSavedForNextStart.value) return "Restart normally";
  if (sourceModeCheckIsReady.value) return "Use this source";
  return "Check first";
});
const sourceModeNextStepCopy = computed(() => {
  if (selectedSourceIsActive.value && !sourceModeSavedForNextStart.value) {
    return `${selectedSourceMode.value} is running now.`;
  }
  if (sourceModeSavedForNextStart.value) {
    return "Restart the backend with your usual command. The saved .env settings will be loaded automatically.";
  }
  if (sourceModeCheckIsReady.value) {
    return "This source looks usable. Save it for the next backend start.";
  }
  return "Check the selected path before saving it.";
});
const sourceModeNextStepClass = computed(() => {
  if (selectedSourceIsActive.value && !sourceModeSavedForNextStart.value) return "source-next-step source-next-step--ready";
  if (sourceModeSavedForNextStart.value || sourceModeCheckIsReady.value) return "source-next-step source-next-step--warn";
  return "source-next-step";
});
const sourceModeActivationSummary = computed<Array<{ label: string; value: string; tone?: "warn" | "up" | "neutral"; monospace?: boolean }>>(() => {
  const preview = sourceModePreview.value;
  const saved = sourceModeSavedForNextStart.value;
  const savedValue = saved
    ? `${formatProvider(saved.preview.mode)} · ${saved.preview.path}`
    : selectedSourceIsActive.value
      ? "No pending change"
      : "Not saved yet";
  return [
    { label: "Running now", value: formatProvider(activeProviderValue.value), tone: selectedSourceIsActive.value ? "up" : "warn" },
    { label: "Saved for next start", value: savedValue, tone: saved ? "warn" : selectedSourceIsActive.value ? "up" : "neutral", monospace: Boolean(saved) },
    { label: "Next step", value: sourceModeNextStep.value, tone: selectedSourceIsActive.value && !saved ? "up" : preview ? "warn" : "neutral" },
  ];
});
const sourceModeEnvironment = computed(() => sourceModePreview.value?.environment ?? []);
const sourceModeAdvancedAvailable = computed(() => sourceModeEnvironment.value.length > 0 || sourceModeActivationCommand.value.length > 0);
const canApplySourceMode = computed(() => {
  const preview = sourceModePreview.value;
  if (!preview || diagnosticsStore.isPreviewingSourceMode || diagnosticsStore.isApplyingSourceMode) return false;
  if (sourceModeSavedForNextStart.value) return false;
  if (!sourceModePath.value.trim()) return false;
  if (preview.mode !== selectedSourceMode.value || preview.path.trim() !== sourceModePath.value.trim()) return false;
  return preview.supported && preview.readable && preview.validStructure && preview.errors.length === 0;
});
const sourceModeCheckLabel = computed(() => {
  if (diagnosticsStore.isPreviewingSourceMode) return "Checking";
  return selectedSourceMode.value === "STRAVA" ? "Check cache" : "Check directory";
});
const sourceModeApplyLabel = computed(() => {
  if (diagnosticsStore.isApplyingSourceMode) return "Saving";
  if (sourceModeSavedForNextStart.value) return "Saved";
  if (sourceModePreview.value?.active) return "Already active";
  return "Use this source";
});
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
const dataQualityIssueTotal = computed(() => dataQualitySummary.value?.issueCount ?? dataQualityIssues.value.length);
const dataQualitySafeFixTotal = computed(() => dataQualitySummary.value?.safeCorrectionCount ?? 0);
const dataQualityManualReviewTotal = computed(() => dataQualitySummary.value?.manualReviewCount ?? 0);
const compositeConflictTotal = computed(() => numberValue(composite.value.conflictCount) ?? compositeConflictRows.value.length);
const statusOverviewItems = computed<StatusItem[]>(() => {
  const sourceDetail = isCompositeProvider.value
    ? activeConfiguredProviders.value.map(formatProvider).join(", ") || "Composite source"
    : sourcePath.value || "No source path detected";
  const qualityDetail = dataQualityIssueTotal.value > 0
    ? `${formatInteger(dataQualityIssueTotal.value)} issue(s), ${formatInteger(dataQualitySafeFixTotal.value)} safe fix(es)`
    : "No blocking local data quality issue";

  return [
    {
      label: "System",
      value: healthStatusLabel.value,
      detail: diagnosticsStore.error || "Backend health payload is reachable.",
      tone: diagnosticsStore.error ? "down" : healthStatusLabel.value === "Operational" ? "up" : "warn",
    },
    {
      label: "Data source",
      value: dataSourceStatusLabel.value,
      detail: sourceDetail,
      tone: dataSourceStatusClass.value.includes("--up") ? "up" : dataSourceStatusClass.value.includes("--down") ? "down" : "warn",
    },
    {
      label: "Routing",
      value: routingStatusLabel.value,
      detail: textValue(routing.value.error) || textValue(routing.value.baseUrl) || "No routing detail available.",
      tone: routingStatus.value === "up" && routingReachable.value ? "up" : routingStatus.value === "disabled" ? "neutral" : routingStatus.value === "unknown" ? "neutral" : "down",
    },
    {
      label: "Data quality",
      value: dataQualityStatusLabel.value,
      detail: qualityDetail,
      tone: dataQualityStatusClass.value.includes("--up") ? "up" : dataQualityStatusClass.value.includes("--down") ? "down" : dataQualityIssueTotal.value > 0 ? "warn" : "neutral",
    },
  ];
});
const attentionItems = computed<StatusItem[]>(() => {
  const items: StatusItem[] = degradedReasons.value.map((reason) => ({
    label: reason.title,
    value: reason.tone === "down" ? "Needs attention" : "Notice",
    detail: reason.detail,
    tone: reason.tone,
  }));

  if (compositeConflictTotal.value > 0) {
    items.push({
      label: "Source conflicts",
      value: `${formatInteger(compositeConflictTotal.value)} conflict(s)`,
      detail: "Matched activities disagree between sources. Review which source is used for totals.",
      tone: "warn",
    });
  }

  if (dataQualityIssueTotal.value > 0) {
    const safeFixText = dataQualitySafeFixTotal.value > 0
      ? `${formatInteger(dataQualitySafeFixTotal.value)} safe fix(es) available`
      : `${formatInteger(dataQualityManualReviewTotal.value)} manual review item(s)`;
    items.push({
      label: "Data quality",
      value: `${formatInteger(dataQualityIssueTotal.value)} issue(s)`,
      detail: `${safeFixText}. Check this before trusting records, distance, elevation, or sensor charts.`,
      tone: dataQualitySummary.value?.status === "critical" ? "down" : "warn",
    });
  }

  if (items.length === 0) {
    items.push({
      label: "All clear",
      value: "No action required",
      detail: "Core source, routing, cache, and quality signals look usable.",
      tone: "up",
    });
  }

  return items;
});
const statusActions = computed<StatusAction[]>(() => {
  const actions: StatusAction[] = [];
  if (canStartOsrm.value) {
    actions.push({
      key: "start-osrm",
      label: "Start OSRM",
      detail: "Bring route generation online.",
      icon: "fa-solid fa-play",
      tone: "primary",
      disabled: diagnosticsStore.isStartingOsrm,
    });
  } else if (routingStatus.value === "down" || routingStatus.value === "misconfigured") {
    actions.push({
      key: "check-osrm",
      label: "Check OSRM",
      detail: "Refresh routing health.",
      icon: "fa-solid fa-route",
      tone: "warn",
      disabled: diagnosticsStore.isLoading,
    });
  }

  if (dataQualitySafeFixTotal.value > 0) {
    actions.push({
      key: "safe-fixes",
      label: "Review safe fixes",
      detail: "Preview local corrections before applying them.",
      icon: "fa-solid fa-wand-magic-sparkles",
      tone: "primary",
      disabled: qualityBatchActionInProgress.value,
    });
  } else if (dataQualityIssueTotal.value > 0) {
    actions.push({
      key: "review-quality",
      label: "Triage data quality",
      detail: "Inspect the highest-impact issues.",
      icon: "fa-solid fa-list-check",
      tone: "warn",
    });
  }

  if (compositeConflictTotal.value > 0) {
    actions.push({
      key: "review-source",
      label: "Review source conflicts",
      detail: "Compare Strava/FIT values.",
      icon: "fa-solid fa-code-compare",
      tone: "warn",
    });
  }

  actions.push({
    key: "synchronize",
    label: "Synchronize",
    detail: "Import new local source files.",
    icon: "fa-solid fa-arrows-rotate",
    tone: "neutral",
    disabled: diagnosticsStore.isSynchronizingSources || diagnosticsStore.isLoading,
  });

  return actions.slice(0, 4);
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

watch(
  [() => diagnosticsStore.hasHealth, sourceModePath, selectedSourceMode],
  () => {
    if (!diagnosticsStore.hasHealth || !sourceModeInitialized.value || sourceModePathEdited.value) {
      return;
    }
    if (!sourceModePath.value.trim()) {
      return;
    }
    const previewKey = `${selectedSourceMode.value}:${sourceModePath.value.trim()}`;
    if (lastSourceModeAutoPreviewKey.value === previewKey) {
      return;
    }
    lastSourceModeAutoPreviewKey.value = previewKey;
    void previewSourceMode(true);
  },
  { immediate: true, flush: "post" },
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

function sourceSyncChangedActivityData(result: unknown): boolean {
  const sync = asRecord(result);
  const fit = asRecord(sync.fit);
  return booleanValue(sync.reloaded) || (numberValue(fit.importedFiles) ?? 0) > 0;
}

function listValues(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((item) => textValue(item))
    .filter((item) => item.length > 0);
}

function recordList(value: unknown): HealthRecord[] {
  if (!Array.isArray(value)) return [];
  return value.map((item) => asRecord(item));
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
  if (normalized === "composite") return "Composite";
  if (normalized === "") return "Unknown";
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}

function formatFITSourceKind(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "fit_inbox") return "FIT inbox";
  if (normalized === "garmin_usb") return "Garmin USB";
  return value || "No source";
}

function formatFITDeviceSyncStatus(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "ok") return "OK";
  if (normalized === "no_device") return "No device";
  if (normalized === "not_configured") return "Not configured";
  if (normalized === "failed") return "Failed";
  return value || "n/a";
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

function formatMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  return `${value.toFixed(0)} m`;
}

function formatSignedMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(0)} m`;
}

function formatDistanceMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  return `${(value / 1000).toFixed(2)} km`;
}

function formatSignedDistance(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : "";
  return `${sign}${(value / 1000).toFixed(2)} km`;
}

function formatDurationSeconds(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const totalSeconds = Math.max(0, Math.round(value));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts: string[] = [];
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0 || hours > 0) parts.push(`${minutes}m`);
  if (seconds > 0 || parts.length === 0) parts.push(`${seconds}s`);
  return parts.join(" ");
}

function formatSignedDurationSeconds(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${formatDurationSeconds(Math.abs(value))}`;
}

function formatSpeed(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  return `${(value * 3.6).toFixed(1)} km/h`;
}

function formatSignedSpeedDelta(before: number | null | undefined, after: number | null | undefined): string {
  if (before === null || before === undefined || after === null || after === undefined) return "n/a";
  const delta = (after - before) * 3.6;
  const sign = delta > 0 ? "+" : "";
  return `${sign}${delta.toFixed(1)} km/h`;
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

function defaultSourceModePath(mode: SourceMode): string {
  if (mode === "FIT") {
    return textValue(root.value.fitDirectory) || textValue(runtimeData.value.fitFilesPath) || "";
  }
  if (mode === "GPX") {
    return textValue(root.value.gpxDirectory) || textValue(runtimeData.value.gpxFilesPath) || "";
  }
  return textValue(root.value.cacheRoot) || textValue(runtimeData.value.stravaCachePath) || "strava-cache";
}

function stravaSetupCommand(path: string): string {
  const trimmedPath = path.trim();
  if (!trimmedPath) return "node scripts/setup-strava-oauth.mjs";
  return `node scripts/setup-strava-oauth.mjs --cache ${shellQuote(trimmedPath)}`;
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`;
}

function tokenStatusLabel(status: StravaOAuthStatus): string {
  if (status.cacheOnly) return "Cache only";
  if (!status.tokenPresent) return "Missing";
  if (!status.tokenReadable) return "Unreadable";
  if (!status.accessTokenPresent || !status.refreshTokenPresent) return "Incomplete";
  if (status.tokenExpired && status.refreshTokenPresent) return "Refreshable";
  if (status.tokenExpired) return "Expired";
  return status.tokenExpiresAt ? `Until ${formatDateTime(status.tokenExpiresAt)}` : "Present";
}

function tokenTone(status: StravaOAuthStatus): "warn" | "down" | undefined {
  if (status.cacheOnly) return undefined;
  if (!status.tokenPresent || status.tokenExpired) return "warn";
  if (!status.tokenReadable || !status.accessTokenPresent || !status.refreshTokenPresent) return "down";
  return undefined;
}

function scopesStatusLabel(status: StravaOAuthStatus): string {
  if (status.cacheOnly) return "n/a";
  if (!status.scopesVerified) return "Not recorded";
  if (status.missingScopes.length > 0) return `Missing ${status.missingScopes.join(", ")}`;
  return status.grantedScopes.join(", ");
}

function stravaStepClass(state: "complete" | "current" | "warn" | "pending"): string {
  return `strava-step strava-step--${state}`;
}

function sourceGuideStepClass(state: "complete" | "current" | "warn" | "pending"): string {
  return `source-guide-step source-guide-step--${state}`;
}

function sourceGuideFactClass(fact: GuideFact): string {
  return fact.tone && fact.tone !== "neutral" ? `source-guide-fact source-guide-fact--${fact.tone}` : "source-guide-fact";
}

function statusOverviewItemClass(tone: StatusTone): string {
  return `status-overview-item status-overview-item--${tone}`;
}

function attentionItemClass(tone: StatusTone): string {
  return `attention-item attention-item--${tone}`;
}

function scrollToSection(sectionId: string) {
  document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

function openDetails(selector: string) {
  const details = document.querySelector<HTMLDetailsElement>(selector);
  if (details) {
    details.open = true;
  }
}

function runStatusAction(actionKey: StatusActionKey) {
  if (actionKey === "start-osrm") {
    void startOsrm();
    return;
  }
  if (actionKey === "check-osrm") {
    void checkRouting();
    return;
  }
  if (actionKey === "safe-fixes") {
    void applySafeCorrections();
    return;
  }
  if (actionKey === "review-quality") {
    openDetails(".quality-issue-details");
    scrollToSection("data-quality-section");
    return;
  }
  if (actionKey === "review-source") {
    openDetails(".source-conflicts-details");
    scrollToSection("data-source-section");
    return;
  }
  if (actionKey === "synchronize") {
    void synchronizeSources();
    return;
  }
  void refreshDiagnostics();
}

function selectSourceMode(mode: SourceMode) {
  selectedSourceMode.value = mode;
  sourceModePathEdited.value = false;
  sourceModePath.value = defaultSourceModePath(mode);
  lastSourceModeAutoPreviewKey.value = "";
  clearSourceModePreview();
}

function markSourceModePathEdited() {
  sourceModePathEdited.value = true;
  lastSourceModeAutoPreviewKey.value = "";
  clearSourceModePreview();
}

function clearSourceModePreview() {
  diagnosticsStore.sourceModePreview = null;
  diagnosticsStore.sourceModeApplyResult = null;
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
    distance: "Distance",
    moving_time: "Moving time",
    start_latlng: "Start point",
  };
  return labels[value] || value;
}

function parseConflictNumber(value: string): number | null {
  const match = value.replace(",", ".").match(/-?\d+(?:\.\d+)?/);
  if (!match) return null;
  const parsed = Number.parseFloat(match[0]);
  return Number.isFinite(parsed) ? parsed : null;
}

function compositePrimaryProviderLabel(otherSource: string): string {
  const other = formatProvider(otherSource);
  return activeConfiguredProviders.value
    .map((provider) => formatProvider(provider))
    .find((provider) => provider !== other) || "Primary source";
}

function compositeConflictTitle(field: string): string {
  const labels: Record<string, string> = {
    distance: "Distance differs between sources",
    moving_time: "Moving time differs between sources",
    start_latlng: "Start point differs between sources",
  };
  return labels[field] || `${formatSourceField(field)} differs between sources`;
}

function formatCompositeConflictValue(field: string, value: string): string {
  const numeric = parseConflictNumber(value);
  if (numeric === null) return value || "n/a";
  if (field === "moving_time") return formatDurationSeconds(numeric);
  if (field === "distance") return formatDistanceMeters(numeric);
  if (field === "start_latlng") return formatMeters(numeric);
  return formatInteger(numeric);
}

function formatCompositeConflictDelta(field: string, primary: string, other: string): string {
  const primaryValue = parseConflictNumber(primary);
  const otherValue = parseConflictNumber(other);
  if (primaryValue === null || otherValue === null) return "n/a";
  const delta = otherValue - primaryValue;
  if (field === "moving_time") return formatSignedDurationSeconds(delta);
  if (field === "distance") return formatSignedDistance(delta);
  if (field === "start_latlng") return formatSignedMeters(delta);
  const sign = delta > 0 ? "+" : "";
  return `${sign}${formatInteger(delta)}`;
}

function compositeConflictSummary(field: string, primaryProvider: string, otherProvider: string, delta: string): string {
  const fieldLabel = formatSourceField(field).toLowerCase();
  if (delta === "n/a") {
    return `${otherProvider} reports a different ${fieldLabel}; ${primaryProvider} keeps the value used in totals.`;
  }
  return `${otherProvider} differs from ${primaryProvider} by ${delta} for ${fieldLabel}; ${primaryProvider} keeps the value used in totals.`;
}

function formatCompositeConflict(conflict: HealthRecord) {
  const rawField = textValue(conflict.field);
  const otherProvider = formatProvider(textValue(conflict.source));
  const primaryProvider = compositePrimaryProviderLabel(otherProvider);
  const rawPrimary = textValue(conflict.primary) || "n/a";
  const rawOther = textValue(conflict.other) || "n/a";
  const delta = formatCompositeConflictDelta(rawField, rawPrimary, rawOther);

  return {
    id: `${otherProvider}-${rawField}-${rawPrimary}-${rawOther}`,
    title: compositeConflictTitle(rawField),
    summary: compositeConflictSummary(rawField, primaryProvider, otherProvider, delta),
    primaryLabel: primaryProvider,
    primaryValue: formatCompositeConflictValue(rawField, rawPrimary),
    otherLabel: otherProvider,
    otherValue: formatCompositeConflictValue(rawField, rawOther),
    delta,
    rawField: rawField || "unknown",
    rawPrimary,
    rawOther,
  };
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

function dataQualityActionForIssue(issue: DataQualityIssue): string {
  if (issue.correction?.available && issue.correction.safety === "safe") return "safe";
  if (issue.correction?.available && issue.correction.safety === "manual") return "manual";
  return "unsupported";
}

function dataQualityActionLabel(issue: DataQualityIssue): string {
  const action = dataQualityActionForIssue(issue);
  if (action === "safe") return "Safe local fix";
  if (action === "manual") return "Manual review";
  return "Unsupported";
}

function dataQualityImpactTokens(issue: DataQualityIssue): string[] {
  const text = `${issue.category} ${issue.field} ${issue.message}`.toLowerCase();
  const tokens = new Set<string>();
  if (issue.severity === "critical" || issue.severity === "warning") {
    tokens.add("records");
  }
  if (text.includes("distance") || text.includes("gps") || text.includes("latlng")) {
    tokens.add("distance");
  }
  if (text.includes("elevation") || text.includes("altitude")) {
    tokens.add("elevation");
  }
  if (text.includes("speed") || text.includes("time") || text.includes("moving")) {
    tokens.add("speed");
  }
  if (text.includes("stream") || text.includes("heart") || text.includes("power") || text.includes("watts") || text.includes("cadence")) {
    tokens.add("sensor");
  }
  if (tokens.size === 0) {
    tokens.add("records");
  }
  return Array.from(tokens);
}

function formatDataQualityImpactToken(value: string): string {
  const labels: Record<string, string> = {
    records: "Records",
    distance: "Distance",
    elevation: "Elevation",
    speed: "Speed",
    sensor: "Sensor",
  };
  return labels[value] || value;
}

function clearDataQualityFilters() {
  dataQualitySeverityFilter.value = "all";
  dataQualityActivityFilter.value = "";
  dataQualityFieldFilter.value = "all";
  dataQualityImpactFilter.value = "all";
  dataQualityActionFilter.value = "all";
  showAllDataQualityIssues.value = false;
}

async function previewIssueCorrection(issue: DataQualityIssue) {
  selectedQualityIssueId.value = issue.id;
  issueCorrectionPreview.value = null;
  issueCorrectionPreviewError.value = "";
  issueCorrectionPreviewLoading.value = true;
  try {
    issueCorrectionPreview.value = await diagnosticsStore.previewCorrection(issue.id);
  } catch (error) {
    issueCorrectionPreviewError.value = error instanceof Error ? error.message : "Unable to preview correction impact.";
  } finally {
    issueCorrectionPreviewLoading.value = false;
  }
}

function closeIssueCorrectionPreview() {
  selectedQualityIssueId.value = null;
  issueCorrectionPreview.value = null;
  issueCorrectionPreviewError.value = "";
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
    closeIssueCorrectionPreview();
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
    safeCorrectionPreview.value = preview;
    if (preview.summary.safeCorrectionCount <= 0) {
      uiStore.showToast({
        id: `quality-fix-empty-${Date.now()}`,
        type: ToastTypeEnum.NORMAL,
        message: "No safe local correction available.",
        timeout: 2600,
      });
      return;
    }
    showSafeCorrectionPreview.value = true;
  } catch (error) {
    uiStore.showToast({
      id: `quality-fix-batch-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to preview safe corrections.",
      timeout: 3600,
    });
  } finally {
    qualityBatchActionInProgress.value = false;
  }
}

function closeSafeCorrectionPreview() {
  if (qualityBatchActionInProgress.value) {
    return;
  }
  showSafeCorrectionPreview.value = false;
}

async function confirmSafeCorrectionPreview() {
  if (!safeCorrectionPreview.value || safeCorrectionPreview.value.summary.safeCorrectionCount <= 0) {
    return;
  }
  qualityBatchActionInProgress.value = true;
  try {
    await diagnosticsStore.applySafeCorrections();
    showSafeCorrectionPreview.value = false;
    safeCorrectionPreview.value = null;
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

async function synchronizeSources() {
  try {
    const result = await diagnosticsStore.synchronizeSources();
    const imported = numberValue(asRecord(result.fit).importedFiles) ?? 0;
    if (sourceSyncChangedActivityData(result)) {
      contextStore.invalidateActivityDerivedCaches();
    }
    uiStore.showToast({
      id: `source-sync-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: imported > 0
        ? `${formatInteger(imported)} FIT file(s) imported. Activity views will reload with the latest data.`
        : textValue(result.message) || "Synchronization completed.",
      timeout: 3600,
    });
  } catch (error) {
    uiStore.showToast({
      id: `source-sync-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to synchronize sources.",
      timeout: 4600,
    });
  }
}

async function checkRouting() {
  await diagnosticsStore.refreshDiagnostics();
}

async function startOsrm() {
  try {
    const result = await diagnosticsStore.startOsrm();
    uiStore.showToast({
      id: `osrm-start-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: result.message || "OSRM start requested.",
      timeout: 3200,
    });
  } catch (error) {
    uiStore.showToast({
      id: `osrm-start-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to start OSRM.",
      timeout: 4200,
    });
  }
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
      message: "Manual command copied.",
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

async function copyStravaEnrollmentCommand() {
  if (!stravaEnrollmentCommand.value) {
    return;
  }
  try {
    await navigator.clipboard.writeText(stravaEnrollmentCommand.value);
    uiStore.showToast({
      id: `strava-enrollment-command-copy-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Strava setup command copied.",
      timeout: 2400,
    });
  } catch {
    uiStore.showToast({
      id: `strava-enrollment-command-copy-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: "Unable to copy Strava setup command.",
      timeout: 3200,
    });
  }
}

async function startStravaEnrollment() {
  let popup: Window | null = null;
  if (!stravaUseCacheInput.value) {
    popup = window.open("about:blank", "mystravastats-strava-oauth");
  }
  stravaEnrollmentInProgress.value = true;
  try {
    const result = await diagnosticsStore.startStravaOAuthEnrollment({
      path: sourceModePath.value,
      clientId: stravaClientIdInput.value,
      clientSecret: stravaClientSecretInput.value,
      useCache: stravaUseCacheInput.value,
    });
    if (result.authorizeUrl) {
      if (popup) {
        popup.location.href = result.authorizeUrl;
      } else {
        window.open(result.authorizeUrl, "_blank", "noreferrer");
      }
    } else if (popup) {
      popup.close();
    }
    stravaClientSecretInput.value = "";
    await previewSourceMode(true);
    uiStore.showToast({
      id: `strava-enrollment-start-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: result.message,
      timeout: 3200,
    });
  } catch (error) {
    if (popup) {
      popup.close();
    }
    uiStore.showToast({
      id: `strava-enrollment-start-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to start Strava OAuth.",
      timeout: 4200,
    });
  } finally {
    stravaEnrollmentInProgress.value = false;
  }
}

async function applySourceMode() {
  if (!canApplySourceMode.value) {
    return;
  }
  try {
    const result = await diagnosticsStore.applySourceMode({
      mode: selectedSourceMode.value,
      path: sourceModePath.value,
    });
    uiStore.showToast({
      id: `source-mode-apply-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: result.restartNeeded
        ? `${formatProvider(result.preview.mode)} source saved. Restart the backend normally to load it.`
        : `${formatProvider(result.preview.mode)} source saved.`,
      timeout: 4200,
    });
  } catch (error) {
    uiStore.showToast({
      id: `source-mode-apply-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to save data source.",
      timeout: 4200,
    });
  }
}

async function previewSourceMode(silent = false) {
  try {
    const preview = await diagnosticsStore.previewSourceMode({
      mode: selectedSourceMode.value,
      path: sourceModePath.value,
    });
    if (!silent) {
      uiStore.showToast({
        id: `source-mode-preview-${Date.now()}`,
        type: preview.errors.length > 0 ? ToastTypeEnum.WARN : ToastTypeEnum.NORMAL,
        message: preview.errors.length > 0 ? "Data source needs attention." : "Data source checked.",
        timeout: 2800,
      });
    }
  } catch {
    if (!silent) {
      uiStore.showToast({
        id: `source-mode-preview-failed-${Date.now()}`,
        type: ToastTypeEnum.WARN,
        message: diagnosticsStore.sourceModePreviewError || "Unable to preview data source.",
        timeout: 3600,
      });
    }
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
          class="btn btn-primary"
          :disabled="diagnosticsStore.isLoading"
          @click="refreshDiagnostics"
        >
          <i class="fa-solid fa-rotate-right" aria-hidden="true" />
          {{ diagnosticsStore.isLoading ? "Refreshing" : "Refresh" }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="diagnosticsStore.isSynchronizingSources || diagnosticsStore.isLoading"
          @click="synchronizeSources"
        >
          <i class="fa-solid fa-arrows-rotate" aria-hidden="true" />
          {{ diagnosticsStore.isSynchronizingSources ? "Synchronizing" : "Synchronize" }}
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
      <section class="diagnostics-panel diagnostics-panel--summary status-overview-panel">
        <div class="status-overview-header">
          <div class="summary-main">
            <span :class="healthStatusClass">{{ healthStatusLabel }}</span>
            <h2>{{ providerLabel }}</h2>
            <p>{{ formatInteger(activityCount) }} activities · {{ availableYears.length }} years</p>
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
        </div>

        <div class="status-overview-grid">
          <div
            v-for="item in statusOverviewItems"
            :key="item.label"
            :class="statusOverviewItemClass(item.tone)"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <small>{{ item.detail }}</small>
          </div>
        </div>

        <div class="status-action-strip">
          <button
            v-for="action in statusActions"
            :key="action.key"
            type="button"
            :class="['status-action', `status-action--${action.tone}`]"
            :disabled="action.disabled"
            @click="runStatusAction(action.key)"
          >
            <i :class="action.icon" aria-hidden="true" />
            <span>
              <strong>{{ action.label }}</strong>
              <small>{{ action.detail }}</small>
            </span>
          </button>
        </div>

        <div class="attention-list">
          <article
            v-for="item in attentionItems"
            :key="`${item.label}-${item.value}`"
            :class="attentionItemClass(item.tone)"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <small>{{ item.detail }}</small>
          </article>
        </div>
      </section>

      <section
        v-if="degradedReasons.length > 0"
        id="degraded-section"
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

      <section
        id="data-source-section"
        class="diagnostics-panel diagnostics-panel--wide"
      >
        <div class="panel-heading">
          <h2>Data Source</h2>
          <span :class="dataSourceStatusClass">{{ dataSourceStatusLabel }}</span>
        </div>
        <div class="source-mode-layout">
          <div class="source-sync-overview">
            <div class="source-sync-heading">
              <div>
                <strong>Synchronization</strong>
                <small>{{ sourceSyncSummary }}</small>
              </div>
              <span :class="sourceSyncStatusClass">{{ sourceSyncStatusLabel }}</span>
            </div>
            <div class="source-preview-metrics source-preview-metrics--compact">
              <div
                v-for="item in sourceSyncKeyItems"
                :key="item.label"
                :class="['source-preview-metric', item.tone === 'warn' ? 'source-preview-metric--warn' : '', item.tone === 'up' ? 'source-preview-metric--up' : '']"
              >
                <span>{{ item.label }}</span>
                <strong :class="{ monospace: item.monospace }">{{ item.value }}</strong>
              </div>
            </div>
            <details class="source-sync-details">
              <summary>
                <i class="fa-solid fa-sliders" aria-hidden="true" />
                Synchronization details
              </summary>
              <div class="source-preview-metrics source-preview-metrics--compact">
                <div
                  v-for="item in sourceSyncItems"
                  :key="`details-${item.label}`"
                  :class="['source-preview-metric', item.tone === 'warn' ? 'source-preview-metric--warn' : '', item.tone === 'up' ? 'source-preview-metric--up' : '']"
                >
                  <span>{{ item.label }}</span>
                  <strong :class="{ monospace: item.monospace }">{{ item.value }}</strong>
                </div>
              </div>
            </details>
          </div>

          <div
            v-if="isCompositeProvider"
            class="composite-source-overview"
          >
            <div class="source-preview-metrics">
              <div
                v-for="item in compositeSummaryItems"
                :key="item.label"
                :class="['source-preview-metric', item.tone === 'warn' ? 'source-preview-metric--warn' : '']"
              >
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
              </div>
            </div>
            <div
              v-if="compositeSourceRows.length > 0"
              class="composite-source-table"
            >
              <div class="composite-source-row composite-source-row--head">
                <span>Source</span>
                <span>Activities</span>
                <span>Years</span>
                <span>Cache</span>
              </div>
              <div
                v-for="source in compositeSourceRows"
                :key="`${source.provider}-${source.athleteId}-${source.cacheRoot}`"
                class="composite-source-row"
              >
                <span>{{ source.provider }}</span>
                <span>{{ source.activities }}</span>
                <span>{{ source.years }}</span>
                <span class="monospace">{{ source.cacheRoot }}</span>
              </div>
            </div>
            <details
              v-if="compositeConflictRows.length > 0"
              class="source-conflicts-details"
            >
              <summary>
                <span>
                  <strong>{{ formatInteger(compositeConflictTotal) }} source conflict(s)</strong>
                  <small>Review matched activities where source values disagree.</small>
                </span>
                <i class="fa-solid fa-chevron-down" aria-hidden="true" />
              </summary>
              <div class="source-message-list source-message-list--warnings source-conflict-list">
                <p class="source-conflict-intro">
                  Matched activities can disagree by source. The primary source stays in charge of summary fields; the secondary value is shown for comparison.
                </p>
                <div
                  v-for="conflict in compositeConflictRows"
                  :key="conflict.id"
                  class="source-message-item source-conflict-item"
                >
                  <strong>{{ conflict.title }}</strong>
                  <span>{{ conflict.summary }}</span>
                  <div class="source-conflict-values">
                    <span>
                      <small>{{ conflict.primaryLabel }}</small>
                      <strong>{{ conflict.primaryValue }}</strong>
                    </span>
                    <span>
                      <small>{{ conflict.otherLabel }}</small>
                      <strong>{{ conflict.otherValue }}</strong>
                    </span>
                    <span>
                      <small>Difference</small>
                      <strong>{{ conflict.delta }}</strong>
                    </span>
                  </div>
                  <small class="source-conflict-raw">
                    Raw: {{ conflict.rawField }} · {{ conflict.rawPrimary }} → {{ conflict.rawOther }}
                  </small>
                </div>
              </div>
            </details>
          </div>
          <details class="source-configuration-details">
            <summary>
              <span>
                <strong>Change data source</strong>
                <small>Check Strava, FIT, or GPX paths and save pending source changes.</small>
              </span>
              <i class="fa-solid fa-chevron-down" aria-hidden="true" />
            </summary>
            <div class="source-configuration-body">
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
                @click="() => previewSourceMode()"
              >
                <i class="fa-solid fa-magnifying-glass" aria-hidden="true" />
                {{ sourceModeCheckLabel }}
              </button>
            </div>
            <div
              v-if="selectedSourceMode === 'STRAVA'"
              class="strava-guide"
            >
              <div class="strava-guide-heading">
                <div>
                  <strong>Connect Strava</strong>
                  <span :class="stravaEnrollmentStatusClass">{{ stravaEnrollmentStatusLabel }}</span>
                </div>
                <a
                  class="btn btn-outline-secondary btn-sm"
                  :href="stravaSettingsHref"
                  target="_blank"
                  rel="noreferrer"
                >
                  <i class="fa-brands fa-strava" aria-hidden="true" />
                  Settings API
                </a>
              </div>
              <div class="strava-required-grid">
                <div
                  v-for="field in stravaRequiredFields"
                  :key="field.label"
                  class="strava-required-item"
                >
                  <span>{{ field.label }}</span>
                  <strong>{{ field.value }}</strong>
                </div>
              </div>
              <div class="strava-enrollment-form">
                <label>
                  <span>Client ID</span>
                  <input
                    v-model="stravaClientIdInput"
                    type="text"
                    inputmode="numeric"
                    autocomplete="off"
                    class="form-control"
                    :placeholder="stravaClientIdPlaceholder"
                  >
                </label>
                <label>
                  <span>Client Secret</span>
                  <input
                    v-model="stravaClientSecretInput"
                    type="password"
                    autocomplete="off"
                    class="form-control"
                    :placeholder="stravaClientSecretPlaceholder"
                  >
                </label>
                <label class="strava-cache-toggle">
                  <input
                    v-model="stravaUseCacheInput"
                    type="checkbox"
                  >
                  <span>Cache only</span>
                </label>
                <button
                  type="button"
                  class="btn btn-primary"
                  :disabled="!canStartStravaEnrollment"
                  @click="startStravaEnrollment"
                >
                  <i
                    :class="stravaUseCacheInput ? 'fa-solid fa-floppy-disk' : 'fa-brands fa-strava'"
                    aria-hidden="true"
                  />
                  {{ stravaEnrollmentActionLabel }}
                </button>
              </div>
              <p :class="stravaSavedCredentialsHintClass">
                <i class="fa-solid fa-circle-info" aria-hidden="true" />
                {{ stravaSavedCredentialsHint }}
              </p>
              <div class="strava-steps">
                <div
                  v-for="step in stravaEnrollmentSteps"
                  :key="step.key"
                  :class="stravaStepClass(step.state)"
                >
                  <i :class="step.icon" aria-hidden="true" />
                  <span>
                    <strong>{{ step.title }}</strong>
                    <small>{{ step.detail }}</small>
                  </span>
                </div>
              </div>
              <div
                v-if="stravaOAuthFacts.length > 0"
                class="strava-oauth-facts"
              >
                <div
                  v-for="fact in stravaOAuthFacts"
                  :key="fact.label"
                  :class="['strava-oauth-fact', fact.tone === 'warn' ? 'strava-oauth-fact--warn' : '', fact.tone === 'down' ? 'strava-oauth-fact--down' : '']"
                >
                  <span>{{ fact.label }}</span>
                  <strong>{{ fact.value }}</strong>
                </div>
              </div>
              <div class="source-command strava-command">
                <code>{{ stravaEnrollmentCommand }}</code>
                <button
                  type="button"
                  class="btn btn-outline-secondary btn-sm"
                  @click="copyStravaEnrollmentCommand"
                >
                  <i class="fa-solid fa-copy" aria-hidden="true" />
                  Copy
                </button>
              </div>
              <p
                v-if="stravaOAuth?.tokenError"
                class="source-mode-error"
              >
                {{ stravaOAuth.tokenError }}
              </p>
            </div>
            <div
              v-if="selectedSourceMode !== 'STRAVA'"
              class="source-guide"
            >
              <div class="source-guide-heading">
                <div>
                  <strong>{{ localSourceGuideTitle }}</strong>
                  <span :class="sourceModeStatusClass">{{ sourceModeStatusLabel }}</span>
                </div>
              </div>
              <p class="source-guide-copy">
                {{ localSourceGuideCopy }}
              </p>
              <div class="source-guide-facts">
                <div
                  v-for="fact in localSourceGuideFacts"
                  :key="fact.label"
                  :class="sourceGuideFactClass(fact)"
                >
                  <span>{{ fact.label }}</span>
                  <strong :class="{ monospace: fact.monospace }">{{ fact.value }}</strong>
                </div>
              </div>
              <p :class="localSourceGuideNextActionClass">
                <i class="fa-solid fa-circle-info" aria-hidden="true" />
                {{ localSourceGuideNextAction }}
              </p>
              <div class="source-guide-steps">
                <div
                  v-for="step in localSourceGuideSteps"
                  :key="step.key"
                  :class="sourceGuideStepClass(step.state)"
                >
                  <i :class="step.icon" aria-hidden="true" />
                  <span>
                    <strong>{{ step.title }}</strong>
                    <small>{{ step.detail }}</small>
                  </span>
                </div>
              </div>
              <div class="source-guide-quality">
                <div class="source-guide-quality-heading">
                  <div>
                    <strong>Data quality</strong>
                    <span :class="localSourceQualityStatusClass">{{ localSourceQualityStatusLabel }}</span>
                  </div>
                  <button
                    v-if="localSourceQualityCanFix"
                    type="button"
                    class="btn btn-sm btn-primary"
                    :disabled="qualityBatchActionInProgress"
                    @click="applySafeCorrections"
                  >
                    <i class="fa-solid fa-wand-magic-sparkles" aria-hidden="true" />
                    Fix safe issues
                  </button>
                </div>
                <p class="source-guide-quality-copy">
                  {{ localSourceQualityCopy }}
                </p>
                <div class="source-guide-quality-facts">
                  <div
                    v-for="fact in localSourceQualityFacts"
                    :key="fact.label"
                    :class="sourceGuideFactClass(fact)"
                  >
                    <span>{{ fact.label }}</span>
                    <strong :class="{ monospace: fact.monospace }">{{ fact.value }}</strong>
                  </div>
                </div>
                <div
                  v-if="localSourceQualityCategories.length > 0"
                  class="source-guide-quality-categories"
                >
                  <span
                    v-for="item in localSourceQualityCategories"
                    :key="item.category"
                    class="quality-category"
                    :title="dataQualityCategoryTooltip(item.category)"
                  >
                    {{ formatDataQualityCategory(item.category) }} · {{ item.count }}
                    <TooltipHint :text="dataQualityCategoryTooltip(item.category)" />
                  </span>
                </div>
                <div
                  v-if="localSourceQualityTopIssues.length > 0"
                  class="source-guide-quality-issues"
                >
                  <div
                    v-for="issue in localSourceQualityTopIssues"
                    :key="issue.id"
                    class="source-guide-quality-issue"
                  >
                    <span :class="dataQualitySeverityClass(issue.severity)">{{ issue.severity }}</span>
                    <strong>{{ formatDataQualityCategory(issue.category) }}</strong>
                    <small>{{ [issue.activityName || issue.activityId, issue.message].filter(Boolean).join(" · ") }}</small>
                  </div>
                  <small
                    v-if="localSourceQualityOverflowCount > 0"
                    class="source-guide-quality-overflow"
                  >
                    +{{ formatInteger(localSourceQualityOverflowCount) }} more in Data Quality.
                  </small>
                </div>
              </div>
            </div>
            <div class="source-activation">
              <div class="source-activation-summary">
                <div
                  v-for="item in sourceModeActivationSummary"
                  :key="item.label"
                  :class="['source-activation-item', item.tone === 'warn' ? 'source-activation-item--warn' : '', item.tone === 'up' ? 'source-activation-item--up' : '']"
                >
                  <span>{{ item.label }}</span>
                  <strong :class="{ monospace: item.monospace }">{{ item.value }}</strong>
                </div>
              </div>
              <p :class="sourceModeNextStepClass">
                <i class="fa-solid fa-circle-info" aria-hidden="true" />
                {{ sourceModeNextStepCopy }}
              </p>
              <div class="source-activation-actions">
                <button
                  type="button"
                  class="btn btn-primary btn-sm source-save-button"
                  :disabled="!canApplySourceMode"
                  @click="applySourceMode"
                >
                  <i class="fa-solid fa-floppy-disk" aria-hidden="true" />
                  {{ sourceModeApplyLabel }}
                </button>
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
              <details
                v-if="sourceModeAdvancedAvailable"
                class="source-advanced"
              >
                <summary>
                  <i class="fa-solid fa-sliders" aria-hidden="true" />
                  Advanced
                </summary>
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
              </details>
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
          </details>
        </div>
      </section>

      <section
        v-if="dataQualitySummary"
        id="data-quality-section"
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
            class="quality-triage"
          >
            <div class="quality-action-groups">
              <button
                v-for="group in dataQualityActionGroups"
                :key="group.key"
                type="button"
                :class="['quality-action-group', { 'quality-action-group--active': dataQualityActionFilter === group.key }]"
                @click="dataQualityActionFilter = dataQualityActionFilter === group.key ? 'all' : group.key"
              >
                <i :class="group.icon" aria-hidden="true" />
                <span>{{ group.label }}</span>
                <strong>{{ formatInteger(group.count) }}</strong>
              </button>
            </div>
            <div class="quality-filters">
              <label>
                <span>Severity</span>
                <select
                  v-model="dataQualitySeverityFilter"
                  class="form-control"
                >
                  <option value="all">All severities</option>
                  <option value="critical">Critical</option>
                  <option value="warning">Warning</option>
                  <option value="info">Info</option>
                </select>
              </label>
              <label>
                <span>Activity</span>
                <input
                  v-model="dataQualityActivityFilter"
                  type="search"
                  class="form-control"
                  placeholder="Name or ID"
                >
              </label>
              <label>
                <span>Field</span>
                <select
                  v-model="dataQualityFieldFilter"
                  class="form-control"
                >
                  <option value="all">All fields</option>
                  <option
                    v-for="field in dataQualityFieldOptions"
                    :key="field.value"
                    :value="field.value"
                  >
                    {{ field.label }}
                  </option>
                </select>
              </label>
              <label>
                <span>Impact</span>
                <select
                  v-model="dataQualityImpactFilter"
                  class="form-control"
                >
                  <option value="all">All impacts</option>
                  <option value="records">Records</option>
                  <option value="distance">Distance</option>
                  <option value="elevation">Elevation</option>
                  <option value="speed">Speed</option>
                  <option value="sensor">Sensor data</option>
                </select>
              </label>
              <button
                type="button"
                class="btn btn-sm btn-outline-secondary"
                :disabled="!hasDataQualityFilters"
                @click="clearDataQualityFilters"
              >
                <i class="fa-solid fa-filter-circle-xmark" aria-hidden="true" />
                Clear
              </button>
            </div>
          </div>

          <details
            v-if="dataQualityIssues.length > 0"
            class="quality-issue-details"
          >
            <summary>
              <span>
                <strong>Issue list</strong>
                <small>{{ dataQualityIssueListLabel }}</small>
              </span>
              <i class="fa-solid fa-chevron-down" aria-hidden="true" />
            </summary>

            <div
              v-if="filteredDataQualityIssues.length > 0"
              class="quality-table"
            >
              <div class="quality-table-toolbar">
                <div>
                  <strong>Visible issues</strong>
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
                <span>Field</span>
                <span>Impact</span>
                <span>Action</span>
              </div>
              <template
                v-for="issue in displayedDataQualityIssues"
                :key="issue.id"
              >
                <div class="quality-row">
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
                      {{ dataQualityActionLabel(issue) }}
                    </small>
                  </span>
                  <span>
                    <strong class="monospace">{{ formatSourceField(issue.field) }}</strong>
                    <small>{{ issue.rawValue || issue.field }}</small>
                  </span>
                  <span class="quality-impact-cell">
                    <small
                      v-for="token in dataQualityImpactTokens(issue)"
                      :key="token"
                      :class="['quality-impact-chip', `quality-impact-chip--${token}`]"
                    >
                      {{ formatDataQualityImpactToken(token) }}
                    </small>
                  </span>
                  <span class="quality-action-cell">
                    <button
                      v-if="issue.correction?.available"
                      type="button"
                      class="btn btn-sm btn-outline-secondary"
                      :disabled="issueCorrectionPreviewLoading && selectedQualityIssueId === issue.id"
                      @click="previewIssueCorrection(issue)"
                    >
                      <i class="fa-solid fa-eye" aria-hidden="true" />
                      Review
                    </button>
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
                    <small
                      v-else-if="issue.correction?.available && issue.correction.safety === 'manual'"
                      class="quality-manual-note"
                    >
                      {{ issue.correction.description || "Manual review required" }}
                    </small>
                    <small v-else>{{ issue.suggestion || "Review source" }}</small>
                  </span>
                </div>
                <div
                  v-if="selectedQualityIssueId === issue.id"
                  class="quality-row-preview"
                >
                  <div class="quality-row-preview-heading">
                    <div>
                      <strong>{{ issue.activityName || issue.activityId || "Selected issue" }}</strong>
                      <small>{{ issue.message }}</small>
                    </div>
                    <button
                      type="button"
                      class="btn btn-sm btn-outline-secondary"
                      @click="closeIssueCorrectionPreview"
                    >
                      <i class="fa-solid fa-xmark" aria-hidden="true" />
                      Close
                    </button>
                  </div>
                  <p
                    v-if="issueCorrectionPreviewLoading"
                    class="quality-preview-inline-state"
                  >
                    Loading correction impact.
                  </p>
                  <p
                    v-else-if="issueCorrectionPreviewError"
                    class="quality-preview-inline-state quality-preview-inline-state--warn"
                  >
                    {{ issueCorrectionPreviewError }}
                  </p>
                  <template v-else-if="issueCorrectionPreview">
                    <div class="quality-impact-grid">
                      <div
                        v-for="stat in issueCorrectionImpactStats"
                        :key="stat.label"
                      >
                        <span>{{ stat.label }}</span>
                        <strong>{{ stat.delta }}</strong>
                        <small>{{ stat.before }} → {{ stat.after }}</small>
                      </div>
                    </div>
                    <div
                      v-if="issueCorrection"
                      class="quality-preview-fields"
                    >
                      <span>{{ dataQualityActionLabel(issue) }}</span>
                      <span>{{ correctionLabel(issueCorrection.type) }}</span>
                      <span
                        v-for="field in issueCorrection.modifiedFields"
                        :key="field"
                      >
                        {{ formatSourceField(field) }}
                      </span>
                    </div>
                    <div
                      v-if="issueCorrectionPreview.warnings.length > 0 || issueCorrectionPreview.blockingReasons.length > 0"
                      class="quality-preview-review"
                    >
                      <span
                        v-for="warning in issueCorrectionPreview.warnings"
                        :key="`warning-${warning}`"
                      >
                        {{ warning }}
                      </span>
                      <span
                        v-for="reason in issueCorrectionPreview.blockingReasons"
                        :key="`blocking-${reason}`"
                      >
                        {{ reason }}
                      </span>
                    </div>
                    <div class="quality-preview-actions">
                      <button
                        v-if="issueCorrection?.safety === 'safe'"
                        type="button"
                        class="btn btn-sm btn-primary"
                        :disabled="qualityActionIssueId === issue.id"
                        @click="applyIssueCorrection(issue)"
                      >
                        <i class="fa-solid fa-wand-magic-sparkles" aria-hidden="true" />
                        Apply safe fix
                      </button>
                    </div>
                  </template>
                </div>
              </template>
            </div>

            <div
              v-if="filteredDataQualityIssues.length === 0"
              class="quality-empty"
            >
              No issue matches the current triage filters.
            </div>
          </details>

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

      <details class="diagnostics-technical">
        <summary class="technical-summary">
          <span>
            <strong>Technical details</strong>
            <small>Runtime config, cache files, warmup, API limits, routing details, and raw payload.</small>
          </span>
          <i class="fa-solid fa-chevron-down" aria-hidden="true" />
        </summary>
        <div class="technical-grid">
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
      </details>
    </div>

    <div
      v-if="showSafeCorrectionPreview && safeCorrectionPreview"
      class="quality-preview-modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="safe-correction-preview-title"
      @click.self="closeSafeCorrectionPreview"
    >
      <section class="quality-preview-panel">
        <div class="quality-preview-heading">
          <div>
            <p>Local corrections</p>
            <h2 id="safe-correction-preview-title">Review safe fixes</h2>
          </div>
          <button
            type="button"
            class="btn btn-sm btn-outline-secondary"
            :disabled="qualityBatchActionInProgress"
            @click="closeSafeCorrectionPreview"
          >
            <i class="fa-solid fa-xmark" aria-hidden="true" />
          </button>
        </div>

        <div class="quality-preview-stats">
          <div>
            <span>Safe fixes</span>
            <strong>{{ formatInteger(safeCorrectionPreview.summary.safeCorrectionCount) }}</strong>
          </div>
          <div>
            <span>Activities</span>
            <strong>{{ formatInteger(safeCorrectionPreview.summary.activityCount) }}</strong>
          </div>
          <div>
            <span>Distance</span>
            <strong>{{ formatSignedDistance(safeCorrectionPreview.summary.distanceDeltaMeters) }}</strong>
          </div>
          <div>
            <span>Elevation</span>
            <strong>{{ formatSignedMeters(safeCorrectionPreview.summary.elevationDeltaMeters) }}</strong>
          </div>
        </div>

        <div
          v-if="safeCorrectionPreview.summary.manualReviewCount > 0 || safeCorrectionPreview.summary.unsupportedIssueCount > 0"
          class="quality-preview-review"
        >
          <span>{{ formatInteger(safeCorrectionPreview.summary.manualReviewCount) }} manual review</span>
          <span>{{ formatInteger(safeCorrectionPreview.summary.unsupportedIssueCount) }} unsupported</span>
        </div>

        <div class="quality-preview-list">
          <div
            v-for="correction in previewedSafeCorrections"
            :key="correction.id"
            class="quality-preview-row"
          >
            <span>
              <strong>{{ correction.activityName || correction.activityId }}</strong>
              <small>{{ correctionLabel(correction.type) }}</small>
            </span>
            <span>{{ correction.modifiedFields.join(", ") }}</span>
            <span>
              {{ formatSignedDistance(correction.impact.distanceDeltaMeters) }}
              <small>{{ formatSignedMeters(correction.impact.elevationDeltaMeters) }}</small>
            </span>
          </div>
          <small
            v-if="safeCorrectionPreviewOverflowCount > 0"
            class="quality-preview-overflow"
          >
            +{{ formatInteger(safeCorrectionPreviewOverflowCount) }} more
          </small>
        </div>

        <div
          v-if="safeCorrectionPreview.summary.modifiedFields.length > 0"
          class="quality-preview-fields"
        >
          <span
            v-for="field in safeCorrectionPreview.summary.modifiedFields"
            :key="field"
          >
            {{ field }}
          </span>
        </div>

        <div class="quality-preview-actions">
          <button
            type="button"
            class="btn btn-outline-secondary"
            :disabled="qualityBatchActionInProgress"
            @click="closeSafeCorrectionPreview"
          >
            Cancel
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="qualityBatchActionInProgress"
            @click="confirmSafeCorrectionPreview"
          >
            <i class="fa-solid fa-wand-magic-sparkles" aria-hidden="true" />
            {{ qualityBatchActionInProgress ? "Applying" : "Apply safe fixes" }}
          </button>
        </div>
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
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 12px;
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
  font-size: 1.35rem;
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
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.diagnostics-panel {
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 12px;
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
}

.status-overview-panel {
  display: grid;
  gap: 14px;
}

.status-overview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.status-overview-grid,
.attention-list {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.status-overview-item,
.attention-item {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 10px;
}

.status-overview-item span,
.attention-item span,
.status-action small {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 800;
}

.status-overview-item strong,
.attention-item strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.status-overview-item small,
.attention-item small {
  display: block;
  margin-top: 4px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.status-overview-item--up,
.attention-item--up {
  border-color: #99d6b0;
  background: #f1fbf5;
}

.status-overview-item--warn,
.attention-item--warn {
  border-color: #f3d17e;
  background: #fff8e3;
}

.status-overview-item--down,
.attention-item--down {
  border-color: #efa4a4;
  background: #fff0f0;
}

.status-overview-item--info,
.attention-item--info,
.status-overview-item--neutral,
.attention-item--neutral {
  border-color: #c8d2e1;
  background: #eef4fb;
}

.status-action-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 8px;
}

.status-action {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 9px;
  align-items: center;
  min-width: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text);
  padding: 10px;
  text-align: left;
}

.status-action i {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #eef4fb;
  color: #31506f;
}

.status-action strong {
  display: block;
  line-height: 1.25;
}

.status-action--primary {
  border-color: #f6b18a;
  background: #fff7f2;
  color: var(--ms-primary-strong);
}

.status-action--primary i {
  background: #ffe4d5;
  color: var(--ms-primary-strong);
}

.status-action--warn {
  border-color: #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.status-action--warn i {
  background: #fff0bf;
  color: #805d05;
}

.status-action:disabled {
  cursor: not-allowed;
  opacity: 0.62;
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
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
}

.source-mode-form,
.source-mode-preview,
.source-sync-overview,
.composite-source-overview {
  grid-column: 1 / -1;
  min-width: 0;
}

.source-sync-overview {
  display: grid;
  gap: 10px;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #fafbfe;
  padding: 12px;
}

.source-sync-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.source-sync-heading strong,
.source-sync-heading small {
  display: block;
}

.source-sync-heading small {
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-weight: 600;
}

.source-sync-details,
.source-conflicts-details,
.source-configuration-details {
  min-width: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px 10px;
}

.source-sync-details summary,
.source-conflicts-details summary,
.source-configuration-details summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: var(--ms-text-muted);
  cursor: pointer;
  font-size: 0.82rem;
  font-weight: 800;
}

.source-sync-details summary {
  justify-content: flex-start;
}

.source-sync-details[open],
.source-conflicts-details[open],
.source-configuration-details[open] {
  display: grid;
  gap: 8px;
}

.source-sync-details[open] summary,
.source-conflicts-details[open] summary,
.source-configuration-details[open] summary {
  margin-bottom: 2px;
}

.source-conflicts-details summary span,
.source-conflicts-details summary strong,
.source-conflicts-details summary small,
.source-configuration-details summary span,
.source-configuration-details summary strong,
.source-configuration-details summary small {
  display: block;
  min-width: 0;
}

.source-conflicts-details summary small,
.source-configuration-details summary small {
  margin-top: 2px;
  font-size: 0.76rem;
  font-weight: 700;
}

.source-configuration-body {
  display: grid;
  gap: 12px;
}

.composite-source-overview {
  display: grid;
  gap: 10px;
}

.composite-source-table {
  display: grid;
  gap: 1px;
  overflow-x: auto;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
}

.composite-source-row {
  display: grid;
  grid-template-columns: minmax(90px, 0.6fr) minmax(86px, 0.4fr) minmax(120px, 0.7fr) minmax(180px, 1.3fr);
  gap: 8px;
  min-width: 680px;
  padding: 8px 10px;
  background: #ffffff;
}

.composite-source-row span {
  min-width: 0;
  overflow-wrap: anywhere;
}

.composite-source-row--head {
  background: #f3f6fa;
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
  text-transform: uppercase;
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

.strava-guide,
.source-guide {
  display: grid;
  gap: 10px;
  margin-top: 12px;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #fbfcff;
  padding: 10px;
}

.strava-guide-heading,
.source-guide-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.strava-guide-heading > div,
.source-guide-heading > div {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.strava-guide-heading strong,
.source-guide-heading strong {
  font-size: 0.92rem;
}

.strava-guide-heading .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.source-guide-copy {
  margin: 0;
  color: var(--ms-text-muted);
  font-size: 0.82rem;
  font-weight: 700;
}

.source-guide-facts,
.source-guide-quality-facts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 7px;
}

.source-guide-fact {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px 9px;
}

.source-guide-fact--up {
  border-color: #b8e5c8;
  background: #f1fbf5;
}

.source-guide-fact--warn {
  border-color: #f3d17e;
  background: #fff8e3;
}

.source-guide-fact--down {
  border-color: #efa4a4;
  background: #fff0f0;
}

.source-guide-fact span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
}

.source-guide-fact strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.source-guide-next {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  width: fit-content;
  margin: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
  padding: 7px 9px;
}

.source-guide-next--ready {
  border-color: #b8e5c8;
  background: #f1fbf5;
  color: #23713b;
}

.source-guide-next--warn {
  border-color: #f6db9a;
  background: #fff8e5;
  color: #7a5b15;
}

.source-next-step {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  width: fit-content;
  max-width: 100%;
  margin: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text-muted);
  font-size: 0.82rem;
  font-weight: 700;
  padding: 8px 10px;
}

.source-next-step--ready {
  border-color: #b8e5c8;
  background: #f1fbf5;
  color: #23713b;
}

.source-next-step--warn {
  border-color: #f6db9a;
  background: #fff8e5;
  color: #7a5b15;
}

.source-guide-quality {
  display: grid;
  gap: 8px;
  border-top: 1px solid #e5e7ee;
  padding-top: 10px;
}

.source-guide-quality-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.source-guide-quality-heading > div {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.source-guide-quality-heading strong {
  font-size: 0.9rem;
}

.source-guide-quality-heading .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.source-guide-quality-copy {
  margin: 0;
  color: var(--ms-text-muted);
  font-size: 0.8rem;
  font-weight: 700;
}

.source-guide-quality-categories {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.source-guide-quality-issues {
  display: grid;
  gap: 6px;
}

.source-guide-quality-issue {
  display: grid;
  grid-template-columns: auto minmax(100px, 0.3fr) minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.source-guide-quality-issue strong,
.source-guide-quality-issue small {
  min-width: 0;
  overflow-wrap: anywhere;
}

.source-guide-quality-issue small,
.source-guide-quality-overflow {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 700;
}

.strava-required-grid,
.strava-oauth-facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
}

.strava-enrollment-form {
  display: grid;
  grid-template-columns: minmax(160px, 0.85fr) minmax(220px, 1fr) auto auto;
  gap: 8px;
  align-items: end;
}

.strava-enrollment-form label {
  display: grid;
  gap: 5px;
  min-width: 0;
  margin: 0;
}

.strava-enrollment-form label > span {
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
}

.strava-enrollment-form .strava-cache-toggle {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 38px;
  border: 1px solid #c8d2e1;
  border-radius: 8px;
  padding: 7px 9px;
  background: #ffffff;
  white-space: nowrap;
}

.strava-cache-toggle input {
  margin: 0;
}

.strava-enrollment-form .btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 38px;
  white-space: nowrap;
}

.strava-saved-hint {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  width: fit-content;
  margin: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
  padding: 7px 9px;
}

.strava-saved-hint--ready {
  border-color: #b8e5c8;
  background: #f1fbf5;
  color: #23713b;
}

.strava-saved-hint--warn {
  border-color: #f6db9a;
  background: #fff8e5;
  color: #7a5b15;
}

.strava-required-item,
.strava-oauth-fact {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px 9px;
}

.strava-oauth-fact--warn {
  border-color: #f3d17e;
  background: #fff8e3;
}

.strava-oauth-fact--down {
  border-color: #efa4a4;
  background: #fff0f0;
}

.strava-required-item span,
.strava-oauth-fact span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
}

.strava-required-item strong,
.strava-oauth-fact strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.strava-steps,
.source-guide-steps {
  display: grid;
  gap: 7px;
}

.strava-step,
.source-guide-step {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px 9px;
}

.strava-step i,
.source-guide-step i {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #eef4fb;
  color: #31506f;
}

.strava-step strong,
.strava-step small,
.source-guide-step strong,
.source-guide-step small {
  display: block;
}

.strava-step small,
.source-guide-step small {
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.strava-step--complete,
.source-guide-step--complete {
  border-color: #99d6b0;
  background: #f3fbf5;
}

.strava-step--complete i,
.source-guide-step--complete i {
  background: #dff4e6;
  color: #176a37;
}

.strava-step--current,
.strava-step--warn,
.source-guide-step--current,
.source-guide-step--warn {
  border-color: #f3d17e;
  background: #fffaf0;
}

.strava-step--current i,
.strava-step--warn i,
.source-guide-step--current i,
.source-guide-step--warn i {
  background: #fff0bf;
  color: #805d05;
}

.strava-step--pending,
.source-guide-step--pending {
  color: var(--ms-text-muted);
}

.strava-command {
  margin-top: 0;
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
.source-save-button,
.source-verify-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.source-activation-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.source-advanced {
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px 10px;
}

.source-advanced summary {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--ms-text-muted);
  cursor: pointer;
  font-size: 0.82rem;
  font-weight: 800;
}

.source-advanced[open] {
  display: grid;
  gap: 8px;
}

.source-advanced[open] summary {
  margin-bottom: 2px;
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

.source-preview-metrics--compact {
  grid-template-columns: repeat(4, minmax(0, 1fr));
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

.source-preview-metric--up {
  border-color: #99d6b0;
  background: #eaf8ef;
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

.source-message-list--warnings .source-message-item {
  border-color: #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.source-conflict-list {
  gap: 8px;
}

.source-conflict-intro {
  margin: 0;
  color: #6f5311;
  font-size: 0.82rem;
  font-weight: 700;
}

.source-conflict-item {
  gap: 7px;
}

.source-conflict-values {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 7px;
}

.source-conflict-values > span {
  display: grid;
  gap: 2px;
  border: 1px solid rgba(128, 93, 5, 0.18);
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.65);
  padding: 6px 8px;
  min-width: 0;
}

.source-conflict-values small,
.source-conflict-raw {
  color: #6f5311;
  font-size: 0.72rem;
  font-weight: 800;
}

.source-conflict-values strong {
  color: #684900;
  overflow-wrap: anywhere;
}

.source-conflict-raw {
  opacity: 0.8;
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

.quality-triage {
  display: grid;
  gap: 10px;
}

.quality-action-groups {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.quality-action-group {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  min-width: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  color: var(--ms-text);
  padding: 9px 10px;
  text-align: left;
}

.quality-action-group i {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #eef4fb;
  color: #31506f;
}

.quality-action-group span,
.quality-action-group strong {
  overflow-wrap: anywhere;
}

.quality-action-group span {
  font-weight: 800;
}

.quality-action-group--active {
  border-color: var(--ms-primary);
  background: #eef4ff;
  color: #244fbd;
}

.quality-filters {
  display: grid;
  grid-template-columns: 0.8fr minmax(160px, 1fr) minmax(180px, 1.2fr) 0.8fr auto;
  gap: 8px;
  align-items: end;
}

.quality-filters label {
  display: grid;
  gap: 5px;
  min-width: 0;
  margin: 0;
}

.quality-filters label > span {
  color: var(--ms-text-muted);
  font-size: 0.72rem;
  font-weight: 800;
}

.quality-filters .btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 38px;
}

.quality-issue-details {
  display: grid;
  min-width: 0;
  border: 1px solid #d7e1ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 10px;
}

.quality-issue-details:not([open]) > :not(summary) {
  display: none;
}

.quality-issue-details summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--ms-text);
  cursor: pointer;
  font-weight: 800;
}

.quality-issue-details summary span,
.quality-issue-details summary strong,
.quality-issue-details summary small {
  display: block;
  min-width: 0;
}

.quality-issue-details summary small {
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
}

.quality-issue-details[open] {
  gap: 10px;
}

.quality-issue-details[open] summary {
  margin-bottom: 2px;
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
  grid-template-columns: minmax(88px, 0.5fr) minmax(170px, 1fr) minmax(220px, 1.35fr) minmax(110px, 0.65fr) minmax(130px, 0.8fr) minmax(230px, 1.35fr);
  gap: 10px;
  min-width: 1080px;
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

.quality-impact-cell {
  display: flex;
  align-content: start;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 5px;
}

.quality-impact-chip {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  border: 1px solid #c8d2e1;
  border-radius: 999px;
  background: #eef4fb;
  color: #31506f;
  padding: 2px 8px;
  font-size: 0.72rem;
  font-weight: 900;
}

.quality-impact-chip--records {
  border-color: #f3d17e;
  background: #fff8e3;
  color: #805d05;
}

.quality-impact-chip--distance,
.quality-impact-chip--elevation,
.quality-impact-chip--speed {
  border-color: #b7c9f7;
  background: #f0f5ff;
  color: #244fbd;
}

.quality-action-cell .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  font-weight: 800;
}

.quality-manual-note {
  max-width: 220px;
}

.quality-row-preview {
  display: grid;
  gap: 10px;
  min-width: 1080px;
  border-top: 1px solid #e5e7ee;
  background: #fbfcff;
  padding: 12px;
}

.quality-row-preview-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.quality-row-preview-heading strong,
.quality-row-preview-heading small {
  display: block;
}

.quality-row-preview-heading small,
.quality-preview-inline-state {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
}

.quality-row-preview-heading .btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.quality-preview-inline-state {
  margin: 0;
}

.quality-preview-inline-state--warn {
  color: #9b1c1c;
}

.quality-impact-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.quality-impact-grid div {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #ffffff;
  padding: 9px;
}

.quality-impact-grid span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 800;
}

.quality-impact-grid strong,
.quality-impact-grid small {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.quality-impact-grid small {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 700;
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

.quality-preview-modal {
  position: fixed;
  inset: 0;
  z-index: 1040;
  display: grid;
  place-items: center;
  padding: 18px;
  background: rgba(15, 23, 42, 0.46);
}

.quality-preview-panel {
  display: grid;
  gap: 12px;
  width: min(760px, 100%);
  max-height: min(760px, calc(100vh - 36px));
  overflow: auto;
  border: 1px solid #c8d2e1;
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 22px 70px rgba(15, 23, 42, 0.24);
  padding: 16px;
}

.quality-preview-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.quality-preview-heading p {
  margin: 0 0 2px;
  color: var(--ms-text-muted);
  font-size: 0.75rem;
  font-weight: 900;
  text-transform: uppercase;
}

.quality-preview-heading h2 {
  margin: 0;
  font-size: 1.16rem;
}

.quality-preview-heading .btn,
.quality-preview-actions .btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.quality-preview-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.quality-preview-stats div {
  min-width: 0;
  border: 1px solid #e5e7ee;
  border-radius: 8px;
  background: #fafbfe;
  padding: 9px;
}

.quality-preview-stats span {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 800;
}

.quality-preview-stats strong {
  display: block;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.quality-preview-review,
.quality-preview-fields {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.quality-preview-review span,
.quality-preview-fields span {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  border: 1px solid #f3d17e;
  border-radius: 999px;
  background: #fff8e3;
  color: #805d05;
  padding: 2px 9px;
  font-size: 0.76rem;
  font-weight: 800;
}

.quality-preview-fields span {
  border-color: #c8d2e1;
  background: #eef4fb;
  color: #31506f;
}

.quality-preview-list {
  display: grid;
  gap: 1px;
  overflow-x: auto;
  border: 1px solid var(--ms-border);
  border-radius: 8px;
}

.quality-preview-row {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(220px, 1fr) minmax(110px, 0.6fr);
  gap: 10px;
  min-width: 620px;
  background: #ffffff;
  padding: 9px 10px;
}

.quality-preview-row span {
  min-width: 0;
  overflow-wrap: anywhere;
}

.quality-preview-row small,
.quality-preview-overflow {
  display: block;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
}

.quality-preview-overflow {
  padding: 9px 10px;
}

.quality-preview-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
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

.diagnostics-technical {
  grid-column: 1 / -1;
  display: grid;
  gap: 10px;
}

.technical-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  cursor: pointer;
  padding: 12px;
}

.technical-summary span,
.technical-summary strong,
.technical-summary small {
  display: block;
  min-width: 0;
}

.technical-summary strong {
  font-size: 1rem;
}

.technical-summary small {
  margin-top: 2px;
  color: var(--ms-text-muted);
  font-size: 0.8rem;
  font-weight: 700;
}

.technical-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.diagnostics-technical[open] .technical-summary {
  margin-bottom: 2px;
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
  .diagnostics-panel--summary,
  .status-overview-header {
    align-items: stretch;
    flex-direction: column;
  }

  .diagnostics-actions {
    justify-content: flex-start;
  }

  .diagnostics-grid,
  .summary-list,
  .status-overview-grid,
  .attention-list,
  .detail-list--columns,
  .warmup-steps,
  .runtime-config-grid,
  .technical-grid,
  .quality-metrics,
  .quality-action-groups,
  .quality-filters,
  .quality-impact-grid,
  .quality-preview-stats,
  .source-mode-layout,
  .source-activation-summary,
  .source-env-row,
  .source-command,
  .source-conflict-values,
  .source-preview-metrics,
  .source-guide-facts,
  .source-guide-quality-facts,
  .source-guide-quality-issue,
  .strava-required-grid,
  .strava-oauth-facts,
  .strava-enrollment-form {
    grid-template-columns: 1fr;
  }

  .strava-guide-heading,
  .source-guide-heading,
  .source-guide-quality-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .degraded-item {
    flex-direction: column;
  }

  .degraded-item span {
    text-align: left;
  }

  .quality-table {
    gap: 8px;
    overflow: visible;
    border: 0;
  }

  .quality-table-toolbar,
  .quality-row,
  .quality-row-preview {
    min-width: 0;
    border: 1px solid var(--ms-border);
    border-radius: 8px;
  }

  .quality-table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .quality-table-actions {
    justify-content: flex-start;
  }

  .quality-row {
    grid-template-columns: 1fr;
  }

  .quality-row--head {
    display: none;
  }

  .quality-action-cell {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
  }

  .quality-action-cell small {
    flex-basis: 100%;
  }

  .quality-row-preview,
  .quality-impact-grid {
    grid-template-columns: 1fr;
  }

  .technical-summary,
  .source-conflicts-details summary,
  .source-configuration-details summary,
  .quality-issue-details summary {
    align-items: flex-start;
  }
}
</style>

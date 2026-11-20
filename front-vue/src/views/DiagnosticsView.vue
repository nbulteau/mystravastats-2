<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useContextStore } from "@/stores/context";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import { useDiagnosticsDataQuality } from "@/composables/useDiagnosticsDataQuality";
import { useDiagnosticsSourceMode } from "@/composables/useDiagnosticsSourceMode";
import TooltipHint from "@/components/TooltipHint.vue";
import { ToastTypeEnum } from "@/models/toast.model";
import type { HealthRecord } from "@/models/health.model";
import type { SourceMode } from "@/models/source-mode.model";
import { RouterLink } from "vue-router";
import { asRecord, numberRecord, textValue, numberValue, booleanValue, sourceSyncChangedActivityData, listValues, recordList, displayList, yesNo, inferProvider, formatProvider, formatFITSourceKind, formatFITDeviceSyncStatus, formatDateTime, formatEpochMs, formatBytes, formatInteger, formatSignedMeters, formatSignedDistance, formatDurationSeconds, formatSignedDurationSeconds, statusClass, fileStatusClass, normalizeSourceMode } from "@/services/diagnostics-formatters";
import { correctionLabel, dataQualityActionLabel, dataQualityCategoryTooltip, dataQualityImpactTokens, dataQualitySeverityClass, dataQualityStatClass, formatCompositeConflict, formatDataQualityCategory, formatDataQualityImpactToken, formatSourceField } from "@/services/diagnostics-presentation";

const contextStore = useContextStore();
const diagnosticsStore = useDiagnosticsStore();
const uiStore = useUiStore();
const showRawPayload = ref(false);
type GuideFactTone = "warn" | "up" | "down" | "neutral";
type GuideFact = { label: string; value: string; tone?: GuideFactTone; monospace?: boolean };
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
const dataQuality = useDiagnosticsDataQuality(diagnosticsStore, uiStore);
const {
  qualityActionActivityId,
  qualityActionIssueId,
  qualityBatchActionInProgress,
  dataQualitySeverityFilter,
  dataQualityActivityFilter,
  dataQualityFieldFilter,
  dataQualityImpactFilter,
  dataQualityActionFilter,
  selectedQualityIssueId,
  issueCorrectionPreview,
  issueCorrectionPreviewLoading,
  issueCorrectionPreviewError,
  safeCorrectionPreview,
  showSafeCorrectionPreview,
  showAllDataQualityIssues,
  dataQualityPreviewLimit,
  dataQualitySummary,
  dataQualityIssues,
  dataQualityCorrections,
  activeDataQualityCorrections,
  previewedSafeCorrections,
  safeCorrectionPreviewOverflowCount,
  hasFullDataQualityReport,
  dataQualityFieldOptions,
  dataQualityActionGroups,
  hasDataQualityFilters,
  filteredDataQualityIssues,
  displayedDataQualityIssues,
  dataQualityIssueListLabel,
  canToggleDataQualityIssues,
  issueCorrection,
  issueCorrectionImpactStats,
  dataQualityStatusLabel,
  dataQualityStatusClass,
  dataQualityStats,
  dataQualityCategories,
  dataQualityIssueTotal,
  dataQualitySafeFixTotal,
  dataQualityManualReviewTotal,
  clearDataQualityFilters,
  previewIssueCorrection,
  closeIssueCorrectionPreview,
  toggleStatsExclusion,
  applyIssueCorrection,
  applySafeCorrections,
  closeSafeCorrectionPreview,
  confirmSafeCorrectionPreview,
  revertCorrection,
} = dataQuality;

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
const sourceMode = useDiagnosticsSourceMode({
  root,
  runtimeData,
  activeProviderValue,
  activeConfiguredProviders,
  isCompositeProvider,
}, diagnosticsStore, uiStore);
const {
  sourceModeOptions,
  selectedSourceMode,
  sourceModePath,
  sourceModePreview,
  activeSourceMode,
  selectedSourceIsActive,
  stravaClientIdInput,
  stravaClientSecretInput,
  stravaUseCacheInput,
  stravaEnrollmentInProgress,
  stravaOAuth,
  stravaSettingsHref,
  stravaEnrollmentCommand,
  stravaEnrollmentActionLabel,
  stravaEnrollmentStatusLabel,
  stravaEnrollmentStatusClass,
  stravaRequiredFields,
  stravaClientIdPlaceholder,
  stravaClientSecretPlaceholder,
  stravaSavedCredentialsHint,
  stravaSavedCredentialsHintClass,
  canStartStravaEnrollment,
  stravaEnrollmentSteps,
  stravaOAuthFacts,
  sourceModePathLabel,
  sourceModeStatusLabel,
  sourceModeStatusClass,
  sourceModePreviewStats,
  localSourceGuideTitle,
  localSourceGuideCopy,
  localSourceGuideFacts,
  localSourceGuideNextAction,
  localSourceGuideNextActionClass,
  localSourceGuideSteps,
  sourceModeActivationCommand,
  sourceModeSavedForNextStart,
  sourceModeCheckIsReady,
  sourceModeNextStep,
  sourceModeNextStepCopy,
  sourceModeNextStepClass,
  sourceModeActivationSummary,
  sourceModeEnvironment,
  sourceModeAdvancedAvailable,
  canApplySourceMode,
  sourceModeCheckLabel,
  sourceModeApplyLabel,
  selectSourceMode,
  markSourceModePathEdited,
  copySourceModeCommand,
  copyStravaEnrollmentCommand,
  startStravaEnrollment,
  applySourceMode,
  previewSourceMode,
  stravaStepClass,
  sourceGuideStepClass,
  sourceGuideFactClass,
} = sourceMode;
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
  recordList(composite.value.conflictSamples).map((conflict) => formatCompositeConflict(conflict, activeConfiguredProviders.value)),
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
const dataSourceStatusLabel = computed(() => {
  if (isCompositeProvider.value) return "Composite active";
  return sourceModeStatusLabel.value;
});
const dataSourceStatusClass = computed(() => {
  if (isCompositeProvider.value) return "status-chip status-chip--up";
  return sourceModeStatusClass.value;
});
const dataQualityProviderMode = computed<SourceMode | null>(() => {
  const provider = dataQualitySummary.value?.provider?.trim();
  return provider ? normalizeSourceMode(provider) : null;
});
const selectedLocalSourceIsActive = computed(() => selectedSourceMode.value !== "STRAVA" && selectedSourceIsActive.value);
const dataQualityMatchesSelectedSource = computed(() => {
  if (!selectedLocalSourceIsActive.value) return false;
  if (dataQualitySummary.value?.provider?.trim().toLowerCase() === "composite") return true;
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

<style scoped src="../assets/views/diagnostics-view.css"></style>

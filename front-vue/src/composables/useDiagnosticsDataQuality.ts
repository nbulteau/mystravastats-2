import { computed, ref } from "vue";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import type { DataQualityCorrection, DataQualityCorrectionPreview, DataQualityIssue, DataQualityReport, DataQualitySummary } from "@/models/data-quality.model";
import type { HealthDetailsPayload } from "@/models/health.model";
import type { Toast } from "@/models/toast.model";
import { asRecord, formatDistanceMeters, formatInteger, formatMeters, formatSignedDistance, formatSignedMeters, formatSignedSpeedDelta, formatSpeed, normalizeDataQualitySummary } from "@/services/diagnostics-formatters";
import { dataQualityActionForIssue, dataQualityCategoryPriority, dataQualityImpactTokens, formatDataQualityCategory, formatSourceField } from "@/services/diagnostics-presentation";

const dataQualityPreviewLimit = 12;

export interface DiagnosticsDataQualityStore {
  health: HealthDetailsPayload | null;
  dataQualityReport: DataQualityReport | null;
  includeActivityInStats(activityId: number): Promise<DataQualityReport>;
  excludeActivityFromStats(activityId: number, reason?: string): Promise<DataQualityReport>;
  previewCorrection(issueId: string): Promise<DataQualityCorrectionPreview>;
  applyCorrection(issueId: string): Promise<DataQualityReport>;
  previewSafeCorrections(): Promise<DataQualityCorrectionPreview>;
  applySafeCorrections(): Promise<DataQualityReport>;
  revertCorrection(correctionId: string): Promise<DataQualityReport>;
}

export interface DiagnosticsDataQualityUiStore {
  showToast(toast: Toast): void;
}

export function useDiagnosticsDataQuality(
  diagnosticsStore: DiagnosticsDataQualityStore = useDiagnosticsStore(),
  uiStore: DiagnosticsDataQualityUiStore = useUiStore(),
) {
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

  const dataQualitySummary = computed<DataQualitySummary | null>(() => {
    if (diagnosticsStore.dataQualityReport) {
      return diagnosticsStore.dataQualityReport.summary;
    }
    return normalizeDataQualitySummary(asRecord(diagnosticsStore.health).dataQuality);
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
  const displayedDataQualityIssues = computed(() => showAllDataQualityIssues.value
    ? filteredDataQualityIssues.value
    : filteredDataQualityIssues.value.slice(0, dataQualityPreviewLimit));
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
  const dataQualityIssueTotal = computed(() => dataQualitySummary.value?.issueCount ?? dataQualityIssues.value.length);
  const dataQualitySafeFixTotal = computed(() => dataQualitySummary.value?.safeCorrectionCount ?? 0);
  const dataQualityManualReviewTotal = computed(() => dataQualitySummary.value?.manualReviewCount ?? 0);

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

  return {
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
  };
}

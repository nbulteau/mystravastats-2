import { computed, ref, watch, type Ref } from "vue";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import type { Toast } from "@/models/toast.model";
import type { HealthRecord } from "@/models/health.model";
import type { SourceMode, SourceModeApplyResult, SourceModePreview, SourceModePreviewRequest, StravaOAuthStatus, StravaOAuthStartRequest } from "@/models/source-mode.model";
import { formatDateTime, formatInteger, formatProvider, normalizeSourceMode, textValue } from "@/services/diagnostics-formatters";

export type DiagnosticsGuideStepState = "complete" | "current" | "warn" | "pending";
export type DiagnosticsGuideFactTone = "warn" | "up" | "down" | "neutral";

export interface DiagnosticsSourceModeContext {
  root: Readonly<Ref<HealthRecord>>;
  runtimeData: Readonly<Ref<HealthRecord>>;
  activeProviderValue: Readonly<Ref<string>>;
  activeConfiguredProviders: Readonly<Ref<string[]>>;
  isCompositeProvider: Readonly<Ref<boolean>>;
}

export interface DiagnosticsSourceModeStore {
  hasHealth: boolean;
  sourceModePreview: SourceModePreview | null;
  sourceModeApplyResult: SourceModeApplyResult | null;
  sourceModePreviewError: string | null;
  isPreviewingSourceMode: boolean;
  isApplyingSourceMode: boolean;
  previewSourceMode(request: SourceModePreviewRequest): Promise<SourceModePreview>;
  applySourceMode(request: { mode: SourceMode; path: string }): Promise<SourceModeApplyResult>;
  startStravaOAuthEnrollment(request: StravaOAuthStartRequest): Promise<{ authorizeUrl: string; message: string }>;
}

export interface DiagnosticsSourceModeUiStore {
  showToast(toast: Toast): void;
}

export function useDiagnosticsSourceMode(
  context: DiagnosticsSourceModeContext,
  diagnosticsStore: DiagnosticsSourceModeStore = useDiagnosticsStore(),
  uiStore: DiagnosticsSourceModeUiStore = useUiStore(),
) {
  const sourceModePathEdited = ref(false);
  const sourceModeInitialized = ref(false);
  const lastSourceModeAutoPreviewKey = ref("");
  const stravaClientIdInput = ref("");
  const stravaClientSecretInput = ref("");
  const stravaUseCacheInput = ref(false);
  const stravaEnrollmentInProgress = ref(false);
  const selectedSourceMode = ref<SourceMode>("STRAVA");
  const sourceModePath = ref("");
  const sourceModeOptions: Array<{ mode: SourceMode; label: string; icon: string }> = [
    { mode: "STRAVA", label: "Strava", icon: "fa-brands fa-strava" },
    { mode: "FIT", label: "FIT", icon: "fa-solid fa-file-lines" },
    { mode: "GPX", label: "GPX", icon: "fa-solid fa-route" },
  ];

  const sourceModePreview = computed(() => diagnosticsStore.sourceModePreview);
  const activeSourceMode = computed(() => normalizeSourceMode(context.activeProviderValue.value));
  const selectedSourceIsActive = computed(() => {
    if (context.isCompositeProvider.value) {
      return context.activeConfiguredProviders.value.map((provider) => provider.toUpperCase()).includes(selectedSourceMode.value);
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
  const stravaEnrollmentSteps = computed<Array<{ key: string; title: string; detail: string; state: DiagnosticsGuideStepState; icon: string }>>(() => {
    const status = stravaOAuth.value;
    const preview = sourceModePreview.value;
    const credentialsReady = status?.credentialsPresent ?? false;
    const tokenReady = status !== null && ["ready", "ready_unverified_scopes", "refreshable", "cache_only"].includes(status.status);
    const tokenWarn = status !== null && ["needs_token", "scope_incomplete"].includes(status.status);
    const tokenDown = status !== null && ["token_unreadable", "token_incomplete", "token_expired"].includes(status.status);
    return [
      { key: "app", title: "Strava app", detail: credentialsReady ? "Client credentials are available locally." : "Create the app and set the callback domain.", state: credentialsReady ? "complete" : "current", icon: "fa-solid fa-id-card" },
      { key: "credentials", title: ".strava", detail: status?.credentialsFile || "Run the setup assistant to write the credentials file.", state: credentialsReady ? "complete" : status?.credentialsFilePresent ? "warn" : "pending", icon: "fa-solid fa-key" },
      { key: "oauth", title: "OAuth token", detail: status?.message || "Check the Strava cache to inspect OAuth readiness.", state: tokenReady ? "complete" : tokenDown ? "warn" : tokenWarn ? "current" : "pending", icon: "fa-solid fa-shield-halved" },
      { key: "activate", title: "Active source", detail: preview?.active ? "Backend is already using this Strava cache." : preview?.restartNeeded ? "Use this source, then restart normally." : "Check the source, then verify active mode.", state: preview?.active ? "complete" : preview ? "warn" : "pending", icon: "fa-solid fa-circle-check" },
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
  const sourceModePathLabel = computed(() => selectedSourceMode.value === "STRAVA" ? "Cache path" : `${selectedSourceMode.value} directory`);
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
    if (sourceModeSavedForNextStart.value || preview.restartNeeded) return "status-chip status-chip--warn";
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
  const localSourceGuideTitle = computed(() => `Import ${selectedSourceMode.value}`);
  const localSourceGuideCopy = computed(() => selectedSourceMode.value === "FIT" ? "Local FIT files grouped by year." : selectedSourceMode.value === "GPX" ? "Local GPX files grouped by year." : "");
  const localSourceGuideFacts = computed<Array<{ label: string; value: string; tone?: DiagnosticsGuideFactTone; monospace?: boolean }>>(() => {
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
  const localSourceGuideSteps = computed<Array<{ key: string; title: string; detail: string; state: DiagnosticsGuideStepState; icon: string }>>(() => {
    const preview = sourceModePreview.value;
    const hasPath = sourceModePath.value.trim().length > 0;
    const sourceLabel = selectedSourceMode.value;
    return [
      { key: "folder", title: `${sourceLabel} directory`, detail: hasPath ? sourceModePath.value : "Directory path is required.", state: preview?.readable ? "complete" : hasPath ? "pending" : "current", icon: "fa-solid fa-folder-open" },
      { key: "files", title: "Check", detail: preview ? `${formatInteger(preview.validFileCount)} valid file(s), ${formatInteger(preview.invalidFileCount)} invalid.` : "Check files before saving.", state: preview ? preview.errors.length > 0 || !preview.validStructure || preview.validFileCount === 0 ? "warn" : "complete" : "pending", icon: "fa-solid fa-file-lines" },
      { key: "restart", title: "Save", detail: sourceModeSavedForNextStart.value ? "Saved for the next backend start." : preview?.active ? "Backend is already using this source." : sourceModeCheckIsReady.value ? "Ready to save for the next backend start." : "Use this source after a successful check.", state: sourceModeSavedForNextStart.value || preview?.active ? "complete" : preview ? "current" : "pending", icon: "fa-solid fa-floppy-disk" },
      { key: "verify", title: "Active source", detail: preview?.active ? "Active mode verified." : "Restart if needed, then verify active mode.", state: preview?.active ? "complete" : preview ? "current" : "pending", icon: "fa-solid fa-circle-check" },
    ];
  });
  const sourceModeActivationCommand = computed(() => sourceModePreview.value?.activationCommand ?? "");
  const sourceModeSavedForNextStart = computed(() => {
    if (selectedSourceIsActive.value) return null;
    const result = diagnosticsStore.sourceModeApplyResult;
    if (!result) return null;
    if (result.preview.mode !== selectedSourceMode.value) return null;
    if (result.preview.path.trim() !== sourceModePath.value.trim()) return null;
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
    if (selectedSourceIsActive.value && !sourceModeSavedForNextStart.value) return `${selectedSourceMode.value} is running now.`;
    if (sourceModeSavedForNextStart.value) return "Restart the backend with your usual command. The saved .env settings will be loaded automatically.";
    if (sourceModeCheckIsReady.value) return "This source looks usable. Save it for the next backend start.";
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
    const savedValue = saved ? `${formatProvider(saved.preview.mode)} · ${saved.preview.path}` : selectedSourceIsActive.value ? "No pending change" : "Not saved yet";
    return [
      { label: "Running now", value: formatProvider(context.activeProviderValue.value), tone: selectedSourceIsActive.value ? "up" : "warn" },
      { label: "Saved for next start", value: savedValue, tone: saved ? "warn" : selectedSourceIsActive.value ? "up" : "neutral", monospace: Boolean(saved) },
      { label: "Next step", value: sourceModeNextStep.value, tone: selectedSourceIsActive.value && !saved ? "up" : preview ? "warn" : "neutral" },
    ];
  });
  const sourceModeEnvironment = computed(() => sourceModePreview.value?.environment ?? []);
  const sourceModeAdvancedAvailable = computed(() => sourceModeEnvironment.value.length > 0 || sourceModeActivationCommand.value.length > 0);
  const canApplySourceMode = computed(() => {
    const preview = sourceModePreview.value;
    if (!preview || diagnosticsStore.isPreviewingSourceMode || diagnosticsStore.isApplyingSourceMode) return false;
    if (sourceModeSavedForNextStart.value || !sourceModePath.value.trim()) return false;
    if (preview.mode !== selectedSourceMode.value || preview.path.trim() !== sourceModePath.value.trim()) return false;
    return preview.supported && preview.readable && preview.validStructure && preview.errors.length === 0;
  });
  const sourceModeCheckLabel = computed(() => diagnosticsStore.isPreviewingSourceMode ? "Checking" : selectedSourceMode.value === "STRAVA" ? "Check cache" : "Check directory");
  const sourceModeApplyLabel = computed(() => {
    if (diagnosticsStore.isApplyingSourceMode) return "Saving";
    if (sourceModeSavedForNextStart.value) return "Saved";
    if (sourceModePreview.value?.active) return "Already active";
    return "Use this source";
  });

  watch(context.runtimeData, () => {
    if (!sourceModeInitialized.value) {
      selectedSourceMode.value = normalizeSourceMode(textValue(context.runtimeData.value.provider));
      sourceModePath.value = defaultSourceModePath(selectedSourceMode.value);
      sourceModeInitialized.value = true;
      return;
    }
    if (!sourceModePathEdited.value) sourceModePath.value = defaultSourceModePath(selectedSourceMode.value);
  }, { immediate: true });

  watch([() => diagnosticsStore.hasHealth, sourceModePath, selectedSourceMode], () => {
    if (!diagnosticsStore.hasHealth || !sourceModeInitialized.value || sourceModePathEdited.value) return;
    if (!sourceModePath.value.trim()) return;
    const previewKey = `${selectedSourceMode.value}:${sourceModePath.value.trim()}`;
    if (lastSourceModeAutoPreviewKey.value === previewKey) return;
    lastSourceModeAutoPreviewKey.value = previewKey;
    void previewSourceMode(true);
  }, { immediate: true, flush: "post" });

  function defaultSourceModePath(mode: SourceMode): string {
    if (mode === "FIT") return textValue(context.root.value.fitDirectory) || textValue(context.runtimeData.value.fitFilesPath) || "";
    if (mode === "GPX") return textValue(context.root.value.gpxDirectory) || textValue(context.runtimeData.value.gpxFilesPath) || "";
    return textValue(context.root.value.cacheRoot) || textValue(context.runtimeData.value.stravaCachePath) || "strava-cache";
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

  async function copySourceModeCommand() {
    if (!sourceModeActivationCommand.value) return;
    try {
      await navigator.clipboard.writeText(sourceModeActivationCommand.value);
      uiStore.showToast({ id: `source-mode-command-copy-${Date.now()}`, type: ToastTypeEnum.NORMAL, message: "Manual command copied.", timeout: 2400 });
    } catch {
      uiStore.showToast({ id: `source-mode-command-copy-failed-${Date.now()}`, type: ToastTypeEnum.WARN, message: "Unable to copy restart command.", timeout: 3200 });
    }
  }

  async function copyStravaEnrollmentCommand() {
    if (!stravaEnrollmentCommand.value) return;
    try {
      await navigator.clipboard.writeText(stravaEnrollmentCommand.value);
      uiStore.showToast({ id: `strava-enrollment-command-copy-${Date.now()}`, type: ToastTypeEnum.NORMAL, message: "Strava setup command copied.", timeout: 2400 });
    } catch {
      uiStore.showToast({ id: `strava-enrollment-command-copy-failed-${Date.now()}`, type: ToastTypeEnum.WARN, message: "Unable to copy Strava setup command.", timeout: 3200 });
    }
  }

  async function startStravaEnrollment() {
    let popup: Window | null = null;
    if (!stravaUseCacheInput.value) popup = window.open("about:blank", "mystravastats-strava-oauth");
    stravaEnrollmentInProgress.value = true;
    try {
      const result = await diagnosticsStore.startStravaOAuthEnrollment({ path: sourceModePath.value, clientId: stravaClientIdInput.value, clientSecret: stravaClientSecretInput.value, useCache: stravaUseCacheInput.value });
      if (result.authorizeUrl) {
        if (popup) popup.location.href = result.authorizeUrl;
        else window.open(result.authorizeUrl, "_blank", "noreferrer");
      } else if (popup) popup.close();
      stravaClientSecretInput.value = "";
      await previewSourceMode(true);
      uiStore.showToast({ id: `strava-enrollment-start-${Date.now()}`, type: ToastTypeEnum.NORMAL, message: result.message, timeout: 3200 });
    } catch (error) {
      if (popup) popup.close();
      uiStore.showToast({ id: `strava-enrollment-start-failed-${Date.now()}`, type: ToastTypeEnum.WARN, message: error instanceof Error ? error.message : "Unable to start Strava OAuth.", timeout: 4200 });
    } finally {
      stravaEnrollmentInProgress.value = false;
    }
  }

  async function applySourceMode() {
    if (!canApplySourceMode.value) return;
    try {
      const result = await diagnosticsStore.applySourceMode({ mode: selectedSourceMode.value, path: sourceModePath.value });
      uiStore.showToast({
        id: `source-mode-apply-${Date.now()}`,
        type: ToastTypeEnum.NORMAL,
        message: result.restartNeeded ? `${formatProvider(result.preview.mode)} source saved. Restart the backend normally to load it.` : `${formatProvider(result.preview.mode)} source saved.`,
        timeout: 4200,
      });
    } catch (error) {
      uiStore.showToast({ id: `source-mode-apply-failed-${Date.now()}`, type: ToastTypeEnum.WARN, message: error instanceof Error ? error.message : "Unable to save data source.", timeout: 4200 });
    }
  }

  async function previewSourceMode(silent = false) {
    try {
      const preview = await diagnosticsStore.previewSourceMode({ mode: selectedSourceMode.value, path: sourceModePath.value });
      if (!silent) uiStore.showToast({ id: `source-mode-preview-${Date.now()}`, type: preview.errors.length > 0 ? ToastTypeEnum.WARN : ToastTypeEnum.NORMAL, message: preview.errors.length > 0 ? "Data source needs attention." : "Data source checked.", timeout: 2800 });
    } catch {
      if (!silent) uiStore.showToast({ id: `source-mode-preview-failed-${Date.now()}`, type: ToastTypeEnum.WARN, message: diagnosticsStore.sourceModePreviewError || "Unable to preview data source.", timeout: 3600 });
    }
  }

  return {
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
    clearSourceModePreview,
    copySourceModeCommand,
    copyStravaEnrollmentCommand,
    startStravaEnrollment,
    applySourceMode,
    previewSourceMode,
    stravaStepClass,
    sourceGuideStepClass,
    sourceGuideFactClass,
  };
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

function stravaStepClass(state: DiagnosticsGuideStepState): string {
  return `strava-step strava-step--${state}`;
}

function sourceGuideStepClass(state: DiagnosticsGuideStepState): string {
  return `source-guide-step source-guide-step--${state}`;
}

function sourceGuideFactClass(fact: { tone?: DiagnosticsGuideFactTone }): string {
  return fact.tone && fact.tone !== "neutral" ? `source-guide-fact source-guide-fact--${fact.tone}` : "source-guide-fact";
}

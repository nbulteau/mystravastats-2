import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { nextTick, reactive, ref } from "vue";
import { useDiagnosticsSourceMode } from "@/composables/useDiagnosticsSourceMode";
import type {
  DiagnosticsSourceModeContext,
  DiagnosticsSourceModeStore,
  DiagnosticsSourceModeUiStore,
} from "@/composables/useDiagnosticsSourceMode";
import type { SourceMode, SourceModeApplyResult, SourceModePreview, SourceModePreviewRequest } from "@/models/source-mode.model";

describe("useDiagnosticsSourceMode", () => {
  let context: DiagnosticsSourceModeContext;
  let diagnosticsStore: DiagnosticsSourceModeStore;
  let uiStore: DiagnosticsSourceModeUiStore;

  beforeEach(() => {
    context = sourceContext({ provider: "FIT", fitFilesPath: "/runtime/fit", gpxFilesPath: "/runtime/gpx" });
    diagnosticsStore = sourceStore();
    uiStore = { showToast: vi.fn() };
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("initializes mode and path from runtime, follows runtime paths until the user edits", async () => {
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    await settleWatchers();

    expect(mode.selectedSourceMode.value).toBe("FIT");
    expect(mode.sourceModePath.value).toBe("/runtime/fit");

    context.runtimeData.value = { provider: "FIT", fitFilesPath: "/runtime/fit-updated" };
    await settleWatchers();
    expect(mode.sourceModePath.value).toBe("/runtime/fit-updated");

    mode.sourceModePath.value = "/chosen/by-user";
    mode.markSourceModePathEdited();
    context.runtimeData.value = { provider: "FIT", fitFilesPath: "/runtime/fit-later" };
    await settleWatchers();
    expect(mode.sourceModePath.value).toBe("/chosen/by-user");
  });

  it("invalidates stale preview and apply state when mode or path changes", async () => {
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    diagnosticsStore.sourceModePreview = preview("FIT", "/runtime/fit");
    diagnosticsStore.sourceModeApplyResult = applyResult(preview("FIT", "/runtime/fit"));
    diagnosticsStore.sourceModePreviewError = "old error";

    mode.selectSourceMode("GPX");

    expect(mode.selectedSourceMode.value).toBe("GPX");
    expect(mode.sourceModePath.value).toBe("/runtime/gpx");
    expect(diagnosticsStore.sourceModePreview).toBeNull();
    expect(diagnosticsStore.sourceModeApplyResult).toBeNull();
    expect(diagnosticsStore.sourceModePreviewError).toBeNull();

    diagnosticsStore.sourceModePreview = preview("GPX", "/runtime/gpx");
    diagnosticsStore.sourceModeApplyResult = applyResult(preview("GPX", "/runtime/gpx"));
    mode.sourceModePath.value = "/user/gpx";
    mode.markSourceModePathEdited();
    expect(diagnosticsStore.sourceModePreview).toBeNull();
    expect(diagnosticsStore.sourceModeApplyResult).toBeNull();
  });

  it("auto-previews once per mode/path key and does not auto-preview after a manual edit", async () => {
    const store = diagnosticsStore as DiagnosticsSourceModeStore & { previewSourceMode: ReturnType<typeof vi.fn> };
    store.previewSourceMode = vi.fn().mockResolvedValue(preview("FIT", "/runtime/fit"));
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    await settleWatchers();

    expect(store.previewSourceMode).toHaveBeenCalledTimes(1);
    expect(store.previewSourceMode).toHaveBeenLastCalledWith({ mode: "FIT", path: "/runtime/fit" });

    context.runtimeData.value = { provider: "FIT", fitFilesPath: "/runtime/fit" };
    await settleWatchers();
    expect(store.previewSourceMode).toHaveBeenCalledTimes(1);

    context.runtimeData.value = { provider: "FIT", fitFilesPath: "/runtime/fit-next" };
    await settleWatchers();
    expect(store.previewSourceMode).toHaveBeenCalledTimes(2);

    mode.sourceModePath.value = "/manual/fit";
    mode.markSourceModePathEdited();
    await settleWatchers();
    expect(store.previewSourceMode).toHaveBeenCalledTimes(2);

    context.runtimeData.value = { provider: "FIT", fitFilesPath: "/runtime/after-edit" };
    await settleWatchers();
    expect(store.previewSourceMode).toHaveBeenCalledTimes(2);
  });

  it("gates apply on a current valid preview and applies only after readiness", async () => {
    const apply = vi.fn().mockResolvedValue(applyResult(preview("FIT", "/runtime/fit")));
    diagnosticsStore.applySourceMode = apply;
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    await settleWatchers();

    expect(mode.canApplySourceMode.value).toBe(false);
    await mode.applySourceMode();
    expect(apply).not.toHaveBeenCalled();

    diagnosticsStore.sourceModePreview = preview("FIT", "/old/path");
    expect(mode.canApplySourceMode.value).toBe(false);
    diagnosticsStore.sourceModePreview = preview("FIT", "/runtime/fit", { errors: [{ path: "ride.fit", message: "Invalid" }] });
    expect(mode.canApplySourceMode.value).toBe(false);
    diagnosticsStore.sourceModePreview = preview("FIT", "/runtime/fit", { readable: false });
    expect(mode.canApplySourceMode.value).toBe(false);
    diagnosticsStore.sourceModePreview = preview("FIT", "/runtime/fit");
    expect(mode.canApplySourceMode.value).toBe(true);

    await mode.applySourceMode();
    expect(apply).toHaveBeenCalledWith({ mode: "FIT", path: "/runtime/fit" });
  });

  it("requires the appropriate Strava fields while allowing saved credentials or cache-only setup", async () => {
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    await settleWatchers();
    mode.selectSourceMode("STRAVA");

    expect(mode.canStartStravaEnrollment.value).toBe(false);
    mode.stravaClientIdInput.value = "client-id";
    expect(mode.canStartStravaEnrollment.value).toBe(false);

    mode.stravaClientSecretInput.value = "client-secret";
    expect(mode.canStartStravaEnrollment.value).toBe(true);

    mode.stravaClientSecretInput.value = "";
    mode.stravaUseCacheInput.value = true;
    expect(mode.canStartStravaEnrollment.value).toBe(true);

    diagnosticsStore.sourceModePreview = preview("STRAVA", "strava-cache", {
      stravaOAuth: oauthStatus({ credentialsPresent: true }),
    });
    mode.stravaClientIdInput.value = "";
    expect(mode.canStartStravaEnrollment.value).toBe(true);
  });

  it("opens OAuth before enrollment, navigates the popup, clears the secret and reports success", async () => {
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    await settleWatchers();
    mode.selectSourceMode("STRAVA");
    mode.stravaClientIdInput.value = "client-id";
    mode.stravaClientSecretInput.value = "secret-value";
    const popup = { location: { href: "about:blank" }, close: vi.fn() } as unknown as Window;
    const calls: string[] = [];
    const open = vi.fn(() => {
      calls.push("open");
      return popup;
    });
    vi.stubGlobal("window", { open });
    const start = vi.fn(() => {
      calls.push("enroll");
      return Promise.resolve({ authorizeUrl: "https://strava.example/authorize", message: "OAuth started." });
    });
    const previewAction = vi.fn().mockResolvedValue(preview("STRAVA", "strava-cache"));
    diagnosticsStore.startStravaOAuthEnrollment = start;
    diagnosticsStore.previewSourceMode = previewAction;

    await mode.startStravaEnrollment();

    expect(open).toHaveBeenCalledWith("about:blank", "mystravastats-strava-oauth");
    expect(calls).toEqual(["open", "enroll"]);
    expect(start).toHaveBeenCalledWith({ path: "strava-cache", clientId: "client-id", clientSecret: "secret-value", useCache: false });
    expect((popup as unknown as { location: { href: string } }).location.href).toBe("https://strava.example/authorize");
    expect(mode.stravaClientSecretInput.value).toBe("");
    expect(previewAction).toHaveBeenCalledWith({ mode: "STRAVA", path: "strava-cache" });
    expect(uiStore.showToast).toHaveBeenCalledWith(expect.objectContaining({ type: "NORMAL", message: "OAuth started." }));
    expect(mode.stravaEnrollmentInProgress.value).toBe(false);
  });

  it("closes the OAuth popup and reports enrollment errors without treating them as success", async () => {
    const mode = useDiagnosticsSourceMode(context, diagnosticsStore, uiStore);
    await settleWatchers();
    mode.selectSourceMode("STRAVA");
    mode.stravaClientIdInput.value = "client-id";
    mode.stravaClientSecretInput.value = "secret-value";
    const popup = { location: { href: "about:blank" }, close: vi.fn() } as unknown as Window;
    vi.stubGlobal("window", { open: vi.fn().mockReturnValue(popup) });
    diagnosticsStore.startStravaOAuthEnrollment = vi.fn().mockRejectedValue(new Error("OAuth setup failed."));

    await mode.startStravaEnrollment();

    expect(popup.close).toHaveBeenCalledOnce();
    expect(mode.stravaClientSecretInput.value).toBe("secret-value");
    expect(uiStore.showToast).toHaveBeenCalledWith(expect.objectContaining({ type: "WARN", message: "OAuth setup failed." }));
    expect(mode.stravaEnrollmentInProgress.value).toBe(false);
  });
});

function sourceContext(runtime: Record<string, unknown>): DiagnosticsSourceModeContext {
  return {
    root: ref({ fitDirectory: "", gpxDirectory: "", cacheRoot: "", ...runtime }) as DiagnosticsSourceModeContext["root"],
    runtimeData: ref(runtime) as DiagnosticsSourceModeContext["runtimeData"],
    activeProviderValue: ref("STRAVA"),
    activeConfiguredProviders: ref(["STRAVA"]),
    isCompositeProvider: ref(false),
  };
}

function sourceStore(): DiagnosticsSourceModeStore {
  return reactive({
    hasHealth: true,
    sourceModePreview: null as SourceModePreview | null,
    sourceModeApplyResult: null as SourceModeApplyResult | null,
    sourceModePreviewError: null as string | null,
    isPreviewingSourceMode: false,
    isApplyingSourceMode: false,
    previewSourceMode: vi.fn(async (_request: SourceModePreviewRequest) => preview("FIT", "/runtime/fit")),
    applySourceMode: vi.fn(async () => applyResult(preview("FIT", "/runtime/fit"))),
    startStravaOAuthEnrollment: vi.fn(async () => ({ authorizeUrl: "", message: "OAuth started." })),
  });
}

function preview(mode: SourceMode, path: string, overrides: Partial<SourceModePreview> = {}): SourceModePreview {
  return {
    mode,
    activeMode: "STRAVA",
    path,
    configKey: "DATA_PROVIDER",
    supported: true,
    active: false,
    configured: false,
    readable: true,
    validStructure: true,
    restartNeeded: true,
    activationCommand: "docker compose restart backend",
    fileCount: 1,
    validFileCount: 1,
    invalidFileCount: 0,
    activityCount: 1,
    years: [],
    missingFields: [],
    environment: [],
    errors: [],
    recommendations: [],
    stravaOAuth: null,
    ...overrides,
  };
}

function applyResult(resultPreview: SourceModePreview): SourceModeApplyResult {
  return { status: "saved", message: "Saved", envFile: ".env", restartNeeded: true, preview: resultPreview };
}

function oauthStatus(overrides: Partial<NonNullable<SourceModePreview["stravaOAuth"]>> = {}): NonNullable<SourceModePreview["stravaOAuth"]> {
  return {
    status: "ready",
    message: "Credentials are ready.",
    settingsUrl: "https://www.strava.com/settings/api",
    callbackDomain: "127.0.0.1",
    oauthCallbackUrl: "http://127.0.0.1/callback",
    setupCommand: "node scripts/setup-strava-oauth.mjs",
    credentialsFile: ".strava",
    tokenFile: ".strava-token.json",
    credentialsFilePresent: true,
    credentialsPresent: true,
    clientIdPresent: true,
    clientSecretPresent: true,
    cacheOnly: false,
    tokenPresent: true,
    tokenReadable: true,
    accessTokenPresent: true,
    refreshTokenPresent: true,
    tokenExpired: false,
    tokenExpiresAt: "",
    athleteId: "",
    athleteName: "",
    scopesVerified: false,
    grantedScopes: [],
    requiredScopes: [],
    missingScopes: [],
    tokenError: "",
    ...overrides,
  };
}

async function settleWatchers(): Promise<void> {
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

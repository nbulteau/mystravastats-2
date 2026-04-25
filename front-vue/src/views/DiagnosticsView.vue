<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useContextStore } from "@/stores/context";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import type { HealthRecord } from "@/models/health.model";

const contextStore = useContextStore();
const diagnosticsStore = useDiagnosticsStore();
const uiStore = useUiStore();
const showRawPayload = ref(false);

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
  { label: "Priority 1", value: textValue(warmup.value.priority1) || "n/a" },
  { label: "Priority 2", value: textValue(warmup.value.priority2) || "n/a" },
  { label: "Priority 3", value: textValue(warmup.value.priority3) || "n/a" },
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

function asRecord(value: unknown): HealthRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? value as HealthRecord
    : {};
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
  .runtime-config-grid {
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

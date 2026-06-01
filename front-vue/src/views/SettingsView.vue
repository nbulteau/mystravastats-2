<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useAthleteStore } from "@/stores/athlete";
import { useContextStore } from "@/stores/context";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";
import {
  emptyAthletePerformanceSettings,
  isIsoDate,
  normalizeAthletePerformanceSettings,
  type AthletePerformanceSettings,
  type AthleteFtpSetting,
} from "@/models/athlete-performance-settings.model";

const contextStore = useContextStore();
const athleteStore = useAthleteStore();
const uiStore = useUiStore();

const isLoading = ref(false);
const isSaving = ref(false);
const loadError = ref("");
const saveError = ref("");
const draftSettings = reactive<AthletePerformanceSettings>(emptyAthletePerformanceSettings());
const draftFtp = ref<number | null>(null);
const draftEffectiveFrom = ref(todayKey());

const manualWeightInput = computed<number | "">({
  get() {
    return draftSettings.weightKg && draftSettings.weightKg > 0
      ? draftSettings.weightKg
      : "";
  },
  set(value) {
    if (value === "" || value === null || value === undefined) {
      draftSettings.weightKg = null;
      return;
    }
    const parsed = Number(value);
    draftSettings.weightKg = Number.isFinite(parsed) && parsed > 0 ? parsed : null;
  },
});

const sortedFtpHistory = computed(() =>
  [...draftSettings.ftpHistory].sort((left, right) => right.effectiveFrom.localeCompare(left.effectiveFrom)),
);

const latestManualFtp = computed(() => sortedFtpHistory.value[0] ?? null);
const canSaveFtpEntry = computed(() =>
  draftFtp.value !== null &&
  Number.isFinite(draftFtp.value) &&
  draftFtp.value > 0 &&
  isIsoDate(draftEffectiveFrom.value),
);

const sourceRows = computed(() => [
  {
    label: "Manual FTP",
    value: latestManualFtp.value ? `${latestManualFtp.value.ftp} W` : "Not set",
    hint: latestManualFtp.value ? `Since ${latestManualFtp.value.effectiveFrom}` : "Fallback will use Strava or estimated FTP",
  },
  {
    label: "Strava FTP",
    value: athleteStore.athleteFtp > 0 ? `${Math.round(athleteStore.athleteFtp)} W` : "Unavailable",
    hint: "Read from Strava profile when exposed by the API",
  },
  {
    label: "Manual weight",
    value: draftSettings.weightKg && draftSettings.weightKg > 0 ? `${draftSettings.weightKg.toFixed(1)} kg` : "Not set",
    hint: "Used before Strava weight in local calculations",
  },
  {
    label: "Strava weight",
    value: athleteStore.athleteWeight > 0 ? `${athleteStore.athleteWeight.toFixed(1)} kg` : "Unavailable",
    hint: "Read from Strava profile when available",
  },
]);

watch(
  () => athleteStore.performanceSettings,
  (settings) => {
    const normalized = normalizeAthletePerformanceSettings(settings);
    draftSettings.ftpHistory = normalized.ftpHistory;
    draftSettings.weightKg = normalized.weightKg;
  },
  { immediate: true, deep: true },
);

onMounted(async () => {
  contextStore.updateCurrentView("settings");
  isLoading.value = true;
  loadError.value = "";
  try {
    await Promise.all([
      athleteStore.fetchAthlete(),
      athleteStore.fetchPerformanceSettings(),
    ]);
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : "Unable to load settings.";
  } finally {
    isLoading.value = false;
  }
});

function addFtpEntry() {
  if (!canSaveFtpEntry.value) {
    saveError.value = "Enter a valid FTP and effective date.";
    return;
  }

  const nextEntry: AthleteFtpSetting = {
    effectiveFrom: draftEffectiveFrom.value,
    ftp: Math.trunc(Number(draftFtp.value)),
  };
  draftSettings.ftpHistory = normalizeAthletePerformanceSettings({
    ...draftSettings,
    ftpHistory: [
      ...draftSettings.ftpHistory.filter((entry) => entry.effectiveFrom !== nextEntry.effectiveFrom),
      nextEntry,
    ],
  }).ftpHistory;
  draftFtp.value = null;
  saveError.value = "";
}

function removeFtpEntry(effectiveFrom: string) {
  draftSettings.ftpHistory = draftSettings.ftpHistory.filter((entry) => entry.effectiveFrom !== effectiveFrom);
}

async function saveSettings() {
  isSaving.value = true;
  saveError.value = "";
  try {
    await athleteStore.savePerformanceSettings(draftSettings);
    uiStore.showToast({
      id: `settings-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Performance settings saved.",
      timeout: 2400,
    });
  } catch (error) {
    saveError.value = error instanceof Error ? error.message : "Unable to save settings.";
  } finally {
    isSaving.value = false;
  }
}

function resetDraft() {
  const normalized = normalizeAthletePerformanceSettings(athleteStore.performanceSettings);
  draftSettings.ftpHistory = normalized.ftpHistory;
  draftSettings.weightKg = normalized.weightKg;
  draftFtp.value = null;
  draftEffectiveFrom.value = todayKey();
  saveError.value = "";
}

function todayKey(): string {
  const now = new Date();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}
</script>

<template>
  <div class="settings-page">
    <header class="settings-header">
      <div>
        <p class="settings-kicker">Athlete</p>
        <h1>Settings</h1>
      </div>
      <div class="settings-actions">
        <button
          type="button"
          class="btn btn-outline-secondary"
          :disabled="isSaving"
          @click="resetDraft"
        >
          <i class="fa-solid fa-rotate-left" aria-hidden="true" />
          Reset
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="isSaving"
          @click="saveSettings"
        >
          <i class="fa-solid fa-floppy-disk" aria-hidden="true" />
          {{ isSaving ? "Saving..." : "Save" }}
        </button>
      </div>
    </header>

    <div
      v-if="loadError"
      class="settings-alert settings-alert--error"
      role="alert"
    >
      {{ loadError }}
    </div>
    <div
      v-if="saveError"
      class="settings-alert settings-alert--error"
      role="alert"
    >
      {{ saveError }}
    </div>

    <div
      v-if="isLoading"
      class="settings-state"
    >
      Loading settings...
    </div>

    <template v-else>
      <section class="settings-summary">
        <article
          v-for="row in sourceRows"
          :key="row.label"
          class="settings-summary-item"
        >
          <span>{{ row.label }}</span>
          <strong>{{ row.value }}</strong>
          <small>{{ row.hint }}</small>
        </article>
      </section>

      <section class="settings-grid">
        <article class="settings-panel">
          <header>
            <h2>FTP History</h2>
          </header>

          <div class="settings-form-grid">
            <label>
              <span>Effective from</span>
              <input
                v-model="draftEffectiveFrom"
                type="date"
                class="form-control"
              >
            </label>
            <label>
              <span>FTP</span>
              <input
                v-model.number="draftFtp"
                type="number"
                min="1"
                step="1"
                class="form-control"
                placeholder="160"
              >
            </label>
            <button
              type="button"
              class="btn btn-outline-primary settings-add-btn"
              :disabled="!canSaveFtpEntry"
              @click="addFtpEntry"
            >
              <i class="fa-solid fa-plus" aria-hidden="true" />
              Add
            </button>
          </div>

          <div
            v-if="sortedFtpHistory.length > 0"
            class="settings-table"
          >
            <div class="settings-table-row settings-table-row--head">
              <span>Date</span>
              <span>FTP</span>
              <span />
            </div>
            <div
              v-for="entry in sortedFtpHistory"
              :key="entry.effectiveFrom"
              class="settings-table-row"
            >
              <span>{{ entry.effectiveFrom }}</span>
              <strong>{{ entry.ftp }} W</strong>
              <button
                type="button"
                class="btn btn-outline-secondary btn-sm settings-icon-btn"
                :aria-label="`Remove FTP from ${entry.effectiveFrom}`"
                :title="`Remove FTP from ${entry.effectiveFrom}`"
                @click="removeFtpEntry(entry.effectiveFrom)"
              >
                <i class="fa-solid fa-trash" aria-hidden="true" />
              </button>
            </div>
          </div>
          <p
            v-else
            class="settings-empty"
          >
            No manual FTP entries.
          </p>
        </article>

        <article class="settings-panel">
          <header>
            <h2>Body Metrics</h2>
          </header>
          <div class="settings-form-grid settings-form-grid--single">
            <label>
              <span>Manual weight</span>
              <input
                v-model="manualWeightInput"
                type="number"
                min="1"
                step="0.1"
                class="form-control"
                placeholder="72.5"
              >
            </label>
          </div>
          <dl class="settings-priority">
            <div>
              <dt>FTP priority</dt>
              <dd>Manual by activity date, Strava profile, estimated from power stream</dd>
            </div>
            <div>
              <dt>Weight priority</dt>
              <dd>Manual setting, Strava profile</dd>
            </div>
          </dl>
        </article>
      </section>
    </template>
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-header {
  align-items: center;
  background: var(--ms-surface-strong);
  border: 1px solid var(--ms-border);
  border-radius: 12px;
  box-shadow: var(--ms-shadow-soft);
  display: flex;
  justify-content: space-between;
  padding: 18px 20px;
}

.settings-kicker {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0;
  margin: 0 0 4px;
  text-transform: uppercase;
}

.settings-header h1 {
  font-size: 1.9rem;
  margin: 0;
}

.settings-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

.settings-actions .btn,
.settings-add-btn {
  align-items: center;
  display: inline-flex;
  gap: 8px;
}

.settings-alert,
.settings-state,
.settings-empty {
  background: #fffaf7;
  border: 1px solid #ffd7c5;
  border-radius: 10px;
  color: var(--ms-text-muted);
  padding: 12px 14px;
}

.settings-alert--error {
  background: #fff6f6;
  border-color: #ffc9c9;
  color: #9f2020;
}

.settings-summary {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.settings-summary-item,
.settings-panel {
  background: var(--ms-surface-strong);
  border: 1px solid var(--ms-border);
  border-radius: 12px;
  box-shadow: var(--ms-shadow-soft);
}

.settings-summary-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 116px;
  padding: 14px 16px;
}

.settings-summary-item span,
.settings-form-grid label span {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

.settings-summary-item strong {
  color: var(--ms-text);
  font-size: 1.45rem;
  line-height: 1.15;
}

.settings-summary-item small {
  color: var(--ms-text-muted);
}

.settings-grid {
  display: grid;
  gap: 14px;
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
}

.settings-panel {
  padding: 16px;
}

.settings-panel header {
  align-items: center;
  display: flex;
  justify-content: space-between;
  margin-bottom: 14px;
}

.settings-panel h2 {
  font-size: 1.2rem;
  margin: 0;
}

.settings-form-grid {
  align-items: end;
  display: grid;
  gap: 10px;
  grid-template-columns: minmax(0, 1fr) minmax(120px, 0.7fr) auto;
  margin-bottom: 14px;
}

.settings-form-grid--single {
  grid-template-columns: minmax(0, 260px);
}

.settings-form-grid label {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.settings-table {
  border: 1px solid var(--ms-border);
  border-radius: 10px;
  overflow: hidden;
}

.settings-table-row {
  align-items: center;
  border-top: 1px solid var(--ms-border);
  display: grid;
  gap: 10px;
  grid-template-columns: minmax(0, 1fr) minmax(90px, auto) 42px;
  min-height: 48px;
  padding: 8px 10px;
}

.settings-table-row:first-child {
  border-top: 0;
}

.settings-table-row--head {
  background: #f8f9fc;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 800;
  text-transform: uppercase;
}

.settings-icon-btn {
  align-items: center;
  aspect-ratio: 1;
  display: inline-flex;
  justify-content: center;
  padding: 0;
}

.settings-priority {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 18px 0 0;
}

.settings-priority div {
  border-top: 1px solid var(--ms-border);
  padding-top: 10px;
}

.settings-priority dt {
  color: var(--ms-text);
  font-weight: 800;
}

.settings-priority dd {
  color: var(--ms-text-muted);
  margin: 3px 0 0;
}

@media (max-width: 992px) {
  .settings-header,
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .settings-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .settings-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .settings-summary,
  .settings-form-grid,
  .settings-form-grid--single {
    grid-template-columns: 1fr;
  }

  .settings-add-btn {
    justify-content: center;
  }
}
</style>

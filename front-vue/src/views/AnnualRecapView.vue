<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink } from "vue-router";
import { useAthleteStore } from "@/stores/athlete";
import { useContextStore } from "@/stores/context";
import { useDashboardStore } from "@/stores/dashboard";
import { useMapStore } from "@/stores/map";
import {
  ALL_ACTIVITY_TYPE_FILTER,
  CYCLING_ACTIVITY_TYPES,
  HIKING_ACTIVITY_TYPES,
  RUNNING_ACTIVITY_TYPES,
} from "@/utils/activityTypes";
import { formatActivityTypeLabel } from "@/utils/formatters";
import {
  ANNUAL_RECAP_FORMATS,
  annualRecapSvgToPng,
  buildAnnualRecapSvg,
  type AnnualRecapFormat,
  type AnnualRecapTheme,
} from "@/utils/annual-recap";

const contextStore = useContextStore();
const dashboardStore = useDashboardStore();
const athleteStore = useAthleteStore();
const mapStore = useMapStore();
const theme = ref<AnnualRecapTheme>("light");
const format = ref<AnnualRecapFormat>("portrait");
const includeMap = ref(false);
const includeName = ref(true);
const isExporting = ref(false);
const exportError = ref("");

const sportOptions = [
  { label: "All sports", value: ALL_ACTIVITY_TYPE_FILTER },
  { label: "Cycling", value: [...CYCLING_ACTIVITY_TYPES].sort().join("_") },
  { label: "Running", value: [...RUNNING_ACTIVITY_TYPES].sort().join("_") },
  { label: "Hiking", value: [...HIKING_ACTIVITY_TYPES].sort().join("_") },
];

const selectedYear = computed({
  get: () => contextStore.currentYear,
  set: (year: string) => void contextStore.updateCurrentYear(year),
});
const selectedSport = computed({
  get: () => contextStore.currentActivityType,
  set: (activityType: string) => void contextStore.updateCurrentActivityType(activityType),
});
const availableYears = computed(() => {
  const candidates = [
    ...contextStore.availableYears,
    ...Object.keys(dashboardStore.dashboardData.nbActivitiesByYear),
  ];
  return [...new Set(candidates)]
    .filter((year) => /^\d{4}$/.test(year))
    .sort((left, right) => Number(right) - Number(left));
});
const currentFormat = computed(() => (
  ANNUAL_RECAP_FORMATS.find((candidate) => candidate.id === format.value) ?? ANNUAL_RECAP_FORMATS[0]
));
const activityLabel = computed(() => contextStore.currentActivityType
  .split("_")
  .map((activityType) => formatActivityTypeLabel(activityType))
  .join(" · "));
const metrics = computed(() => {
  const year = selectedYear.value;
  const data = dashboardStore.dashboardData;
  return {
    activities: positiveValue(data.nbActivitiesByYear[year]),
    activeDays: positiveValue(data.activeDaysByYear[year]),
    distanceKm: positiveValue(data.totalDistanceByYear[year]),
    elevationM: positiveValue(data.totalElevationByYear[year]),
    movingTimeSeconds: positiveValue(data.movingTimeByYear[year]),
    longestActivityKm: positiveValue(data.maxDistanceByYear[year]),
    longestActivityDate: data.maxDistanceDateByYear[year],
  };
});
const hasData = computed(() => selectedYear.value !== "All years" && metrics.value.activities > 0);
const previewSvg = computed(() => buildAnnualRecapSvg({
  year: selectedYear.value === "All years" ? "YEAR" : selectedYear.value,
  athleteName: includeName.value ? athleteStore.athleteName : "",
  activityLabel: activityLabel.value || "All activities",
  metrics: metrics.value,
  theme: theme.value,
  format: format.value,
  includeMap: includeMap.value,
  tracks: includeMap.value ? mapStore.mapTracks : [],
}));

onMounted(() => contextStore.updateCurrentView("annual-recap"));

watch(
  () => [includeMap.value, contextStore.currentFiltersKey] as const,
  ([enabled]) => {
    if (enabled) void mapStore.ensureLoaded();
  },
);

async function downloadPng() {
  if (!hasData.value || isExporting.value) return;
  isExporting.value = true;
  exportError.value = "";
  try {
    const blob = await annualRecapSvgToPng(
      previewSvg.value,
      currentFormat.value.width,
      currentFormat.value.height,
    );
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `mystravastats-${selectedYear.value}-${format.value}.png`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  } catch (error) {
    exportError.value = error instanceof Error ? error.message : "Unable to export the annual recap.";
  } finally {
    isExporting.value = false;
  }
}

function positiveValue(value: number | undefined): number {
  return typeof value === "number" && Number.isFinite(value) && value > 0 ? value : 0;
}
</script>

<template>
  <div class="recap-page">
    <header class="recap-heading">
      <div>
        <p class="recap-kicker">Share studio</p>
        <h1>Annual recap</h1>
        <p>Create a social-ready image from your activity data. Rendering and export stay in your browser.</p>
      </div>
      <RouterLink class="btn btn-outline-secondary btn-sm" to="/dashboard">
        <i class="fa-solid fa-arrow-left" aria-hidden="true" />
        Dashboard
      </RouterLink>
    </header>

    <div class="recap-layout">
      <aside class="recap-controls" aria-label="Annual recap settings">
        <section>
          <span class="step-number">1</span>
          <div>
            <h2>Choose your data</h2>
            <label for="recap-year">Year</label>
            <select id="recap-year" v-model="selectedYear" class="form-select">
              <option v-for="year in availableYears" :key="year" :value="year">{{ year }}</option>
              <option v-if="selectedYear === 'All years'" value="All years">Select a year</option>
            </select>
            <label>Sports</label>
            <div class="choice-grid">
              <button
                v-for="option in sportOptions"
                :key="option.value"
                type="button"
                :class="{ active: selectedSport === option.value }"
                :aria-pressed="selectedSport === option.value"
                @click="selectedSport = option.value"
              >
                {{ option.label }}
              </button>
            </div>
            <p v-if="!sportOptions.some((option) => option.value === selectedSport)" class="selection-note">
              Custom selection from the activity filter is active.
            </p>
          </div>
        </section>

        <section>
          <span class="step-number">2</span>
          <div>
            <h2>Choose a format</h2>
            <div class="choice-grid">
              <button
                v-for="option in ANNUAL_RECAP_FORMATS"
                :key="option.id"
                type="button"
                :class="{ active: format === option.id }"
                :aria-pressed="format === option.id"
                @click="format = option.id"
              >
                {{ option.label }}
              </button>
            </div>
            <label>Theme</label>
            <div class="theme-switch" role="group" aria-label="Recap theme">
              <button type="button" :class="{ active: theme === 'light' }" :aria-pressed="theme === 'light'" @click="theme = 'light'">Light</button>
              <button type="button" :class="{ active: theme === 'dark' }" :aria-pressed="theme === 'dark'" @click="theme = 'dark'">Dark</button>
            </div>
          </div>
        </section>

        <section>
          <span class="step-number">3</span>
          <div>
            <h2>Privacy</h2>
            <label class="check-row">
              <input v-model="includeName" type="checkbox">
              <span><strong>Show athlete name</strong><small>Turn this off for an anonymous recap.</small></span>
            </label>
            <label class="check-row">
              <input v-model="includeMap" type="checkbox">
              <span><strong>Add activity fingerprint</strong><small>Off by default. The export contains abstract paths, no coordinate labels or basemap.</small></span>
            </label>
            <p v-if="includeMap && mapStore.isLoading" class="selection-note">Loading GPS traces…</p>
            <p v-else-if="includeMap && mapStore.error" class="selection-note selection-note--error">{{ mapStore.error }}</p>
          </div>
        </section>

        <p v-if="dashboardStore.error" class="recap-error">{{ dashboardStore.error }}</p>
        <p v-else-if="selectedYear === 'All years'" class="recap-error">Choose a specific year to create a recap.</p>
        <p v-else-if="!dashboardStore.isLoading && !hasData" class="recap-error">No activities are available for this selection.</p>
        <p v-if="exportError" class="recap-error">{{ exportError }}</p>

        <button
          type="button"
          class="btn btn-primary recap-download"
          :disabled="dashboardStore.isLoading || !hasData || isExporting"
          @click="downloadPng"
        >
          <i class="fa-solid fa-download" aria-hidden="true" />
          {{ isExporting ? "Creating PNG…" : `Download PNG · ${currentFormat.width} × ${currentFormat.height}` }}
        </button>
      </aside>

      <main class="recap-preview-column">
        <div class="preview-heading">
          <div><span>Live preview</span><strong>{{ currentFormat.label }}</strong></div>
          <span>{{ includeMap ? "Map enabled" : "Privacy-first" }}</span>
        </div>
        <div
          class="recap-preview"
          :class="`recap-preview--${format}`"
          :aria-busy="dashboardStore.isLoading"
          v-html="previewSvg"
        />
      </main>
    </div>
  </div>
</template>

<style scoped>
.recap-page { display: flex; flex-direction: column; gap: 18px; }
.recap-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; padding: 20px 22px; border: 1px solid var(--ms-border); border-radius: 16px; background: linear-gradient(135deg, #fff 0%, #fff7f1 100%); }
.recap-heading h1 { margin: 2px 0 5px; font-size: 1.55rem; font-weight: 850; }
.recap-heading p { max-width: 720px; margin: 0; color: var(--ms-text-muted); }
.recap-kicker { color: var(--ms-primary) !important; font-size: .68rem; font-weight: 850; letter-spacing: .1em; text-transform: uppercase; }
.recap-layout { display: grid; grid-template-columns: minmax(300px, 390px) minmax(0, 1fr); align-items: start; gap: 18px; }
.recap-controls { display: flex; flex-direction: column; gap: 12px; }
.recap-controls section { display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 10px; padding: 15px; border: 1px solid var(--ms-border); border-radius: 14px; background: #fff; }
.recap-controls h2 { margin: 2px 0 13px; font-size: .9rem; }
.recap-controls label:not(.check-row) { display: block; margin: 12px 0 5px; color: var(--ms-text-muted); font-size: .68rem; font-weight: 800; text-transform: uppercase; }
.step-number { display: inline-flex; width: 30px; height: 30px; align-items: center; justify-content: center; border-radius: 50%; color: #fff; background: var(--ms-primary); font-size: .75rem; font-weight: 850; }
.choice-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }
.choice-grid button,.theme-switch button { min-height: 36px; padding: 7px 8px; border: 1px solid #dde1e7; border-radius: 8px; color: #565c66; background: #fafbfc; font-size: .72rem; font-weight: 750; }
.choice-grid button.active,.theme-switch button.active { border-color: #f2a17e; color: #a73c0d; background: #fff0e8; box-shadow: inset 0 0 0 1px #ffd6c4; }
.theme-switch { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.check-row { display: flex; align-items: flex-start; gap: 9px; padding: 9px 0; border-top: 1px solid #eef0f3; }
.check-row input { margin-top: 3px; accent-color: var(--ms-primary); }
.check-row span { display: flex; flex-direction: column; gap: 2px; }
.check-row strong { font-size: .75rem; }
.check-row small,.selection-note { color: var(--ms-text-muted); font-size: .64rem; }
.selection-note { margin: 8px 0 0; }
.selection-note--error,.recap-error { color: #9f2f23; }
.recap-error { margin: 0; padding: 9px 11px; border: 1px solid #efc1b9; border-radius: 9px; background: #fff5f3; font-size: .7rem; }
.recap-download { width: 100%; min-height: 44px; font-weight: 800; }
.recap-preview-column { position: sticky; top: 64px; min-width: 0; padding: 15px; border: 1px solid var(--ms-border); border-radius: 16px; background: #edf0f4; }
.preview-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; color: var(--ms-text-muted); font-size: .68rem; }
.preview-heading div { display: flex; gap: 8px; }
.preview-heading strong { color: var(--ms-text); }
.recap-preview { width: min(100%, 640px); overflow: hidden; margin: 0 auto; border-radius: 8px; box-shadow: 0 18px 42px rgb(31 41 55 / 18%); line-height: 0; }
.recap-preview :deep(svg) { display: block; width: 100%; height: auto; }
.recap-preview--story { width: min(100%, 430px); }
.recap-preview--square { width: min(100%, 650px); }
@media (max-width: 900px) { .recap-layout { grid-template-columns: 1fr; }.recap-preview-column { position: static; }.recap-controls { order: 2; }.recap-preview-column { order: 1; } }
@media (max-width: 560px) { .recap-heading { flex-direction: column; }.recap-layout { gap: 12px; }.choice-grid { grid-template-columns: 1fr; }.recap-preview-column { padding: 8px; }.preview-heading > span { display: none; } }
</style>

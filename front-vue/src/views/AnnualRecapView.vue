<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink, useRoute } from "vue-router";
import { useAthleteStore } from "@/stores/athlete";
import { useContextStore } from "@/stores/context";
import { useDashboardStore } from "@/stores/dashboard";
import { useMapStore } from "@/stores/map";
import type { DashboardData } from "@/models/dashboard-data.model";
import type { Activity } from "@/models/activity.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import {
  ALL_ACTIVITY_TYPE_FILTER,
  CYCLING_ACTIVITY_TYPES,
  HIKING_ACTIVITY_TYPES,
  RUNNING_ACTIVITY_TYPES,
} from "@/utils/activityTypes";
import { formatActivityTypeLabel } from "@/utils/formatters";
import {
  ANNUAL_RECAP_FORMATS,
  ANNUAL_RECAP_PAGES,
  annualRecapSvgToPng,
  buildAnnualRecapHighlights,
  buildAnnualRecapSvg,
  type AnnualRecapFormat,
  type AnnualRecapMetrics,
  type AnnualRecapPage,
  type AnnualRecapTheme,
} from "@/utils/annual-recap";
import {
  COMMUTE_RECAP_PAGES,
  buildCommuteRecapStats,
  buildCommuteRecapSvg,
  type CommuteImpactConfig,
  type CommuteRecapPage,
} from "@/utils/commute-recap";

const route = useRoute();
const contextStore = useContextStore();
const dashboardStore = useDashboardStore();
const athleteStore = useAthleteStore();
const mapStore = useMapStore();
const theme = ref<AnnualRecapTheme>("light");
const format = ref<AnnualRecapFormat>("portrait");
const includeMap = ref(false);
const includeName = ref(true);
const selectedPage = ref<AnnualRecapPage | CommuteRecapPage>("overview");
const isExporting = ref(false);
const exportError = ref("");
const exportNotice = ref("");
const allYearsData = ref<DashboardData | null>(null);
const allYearsLoading = ref(false);
const allYearsError = ref("");
const supportsNativeShare = ref(false);
const commuteActivities = ref<Activity[]>([]);
const commuteLoading = ref(false);
const commuteError = ref("");
const impactEnabled = ref(false);
const impactConfig = ref<CommuteImpactConfig>({
  motorizedSharePercent: 100,
  fuelConsumptionLitresPer100Km: 6.5,
  fuelPricePerLitre: 1.85,
  co2KgPerKm: 0.192,
});
let allYearsRequestId = 0;
let commuteRequestId = 0;

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
const isCommuteMode = computed(() => route.name === "commute-recap");
const pageDefinitions = computed(() => isCommuteMode.value ? COMMUTE_RECAP_PAGES : ANNUAL_RECAP_PAGES);
const selectedSport = computed({
  get: () => contextStore.currentActivityType,
  set: (activityType: string) => void contextStore.updateCurrentActivityType(activityType),
});
const availableYears = computed(() => {
  const candidates = [
    ...contextStore.availableYears,
    ...Object.keys(allYearsData.value?.nbActivitiesByYear ?? {}),
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
  return metricsForYear(allYearsData.value ?? dashboardStore.dashboardData, year);
});
const previousYear = computed(() => String(Number(selectedYear.value) - 1));
const yearToDate = computed(() => selectedYear.value === String(new Date().getFullYear()));
const previousMetrics = computed(() => metricsForYear(allYearsData.value ?? dashboardStore.dashboardData, previousYear.value));
const hasPreviousData = computed(() => previousMetrics.value.activities > 0);
const consistencyPercent = computed(() => positiveValue((allYearsData.value ?? dashboardStore.dashboardData).consistencyByYear[selectedYear.value]));
const previousConsistencyPercent = computed(() => positiveValue((allYearsData.value ?? dashboardStore.dashboardData).consistencyByYear[previousYear.value]));
const highlights = computed(() => buildAnnualRecapHighlights(
  metrics.value,
  hasPreviousData.value ? previousMetrics.value : undefined,
  consistencyPercent.value,
  yearToDate.value,
));
const commuteStats = computed(() => buildCommuteRecapStats(commuteActivities.value, selectedYear.value));
const previousCommuteStats = computed(() => buildCommuteRecapStats(commuteActivities.value, previousYear.value));
const hasPreviousCommuteData = computed(() => previousCommuteStats.value.trips > 0);
const hasData = computed(() => selectedYear.value !== "All years" && (
  isCommuteMode.value ? commuteStats.value.trips > 0 : metrics.value.activities > 0
));
const pageSvgs = computed(() => new Map(pageDefinitions.value.map((page) => [
  page.id,
  isCommuteMode.value
    ? buildCommuteRecapSvg(commuteRecapInput(page.id as CommuteRecapPage))
    : buildAnnualRecapSvg(recapInput(page.id as AnnualRecapPage)),
])));
const previewSvg = computed(() => pageSvgs.value.get(selectedPage.value) ?? "");
const selectedPageIndex = computed(() => pageDefinitions.value.findIndex((page) => page.id === selectedPage.value));
const selectedPageLabel = computed(() => pageDefinitions.value[selectedPageIndex.value]?.label ?? "Overview");
const isLoading = computed(() => dashboardStore.isLoading || allYearsLoading.value || (isCommuteMode.value && commuteLoading.value));

onMounted(() => {
  contextStore.updateCurrentView("annual-recap");
  supportsNativeShare.value = typeof navigator.share === "function";
});

watch(isCommuteMode, (commuteMode) => {
  selectedPage.value = "overview";
  exportError.value = "";
  exportNotice.value = "";
  if (commuteMode) {
    void contextStore.updateCurrentActivityType("Commute");
    void fetchCommuteActivities();
  }
}, { immediate: true });

watch(
  () => contextStore.currentActivityType,
  () => void fetchAllYearsData(),
  { immediate: true },
);

watch(
  () => [includeMap.value, contextStore.currentFiltersKey] as const,
  ([enabled]) => {
    if (enabled) void mapStore.ensureLoaded();
  },
);

async function downloadCurrentPng() {
  if (!hasData.value || isExporting.value) return;
  isExporting.value = true;
  exportError.value = "";
  exportNotice.value = "";
  try {
    downloadBlob(await renderPageBlob(selectedPage.value), fileName(selectedPage.value));
    exportNotice.value = `${selectedPageLabel.value} card downloaded.`;
  } catch (error) {
    exportError.value = error instanceof Error ? error.message : "Unable to export the annual recap.";
  } finally {
    isExporting.value = false;
  }
}

async function downloadAllPngs() {
  if (!hasData.value || isExporting.value) return;
  isExporting.value = true;
  exportError.value = "";
  exportNotice.value = "";
  try {
    for (const page of pageDefinitions.value) {
      downloadBlob(await renderPageBlob(page.id), fileName(page.id));
      await new Promise((resolve) => window.setTimeout(resolve, 120));
    }
    exportNotice.value = `${pageDefinitions.value.length} recap cards downloaded.`;
  } catch (error) {
    exportError.value = error instanceof Error ? error.message : "Unable to export all recap cards.";
  } finally {
    isExporting.value = false;
  }
}

async function shareCurrentCard() {
  if (!supportsNativeShare.value || !hasData.value || isExporting.value) return;
  isExporting.value = true;
  exportError.value = "";
  exportNotice.value = "";
  try {
    const blob = await renderPageBlob(selectedPage.value);
    const file = new File([blob], fileName(selectedPage.value), { type: "image/png" });
    if (typeof navigator.canShare === "function" && !navigator.canShare({ files: [file] })) {
      throw new Error("This browser cannot share generated image files.");
    }
    await navigator.share({
      title: `${selectedYear.value} ${isCommuteMode.value ? "commute" : "activity"} recap`,
      text: `${selectedPageLabel.value} · ${selectedYear.value} · MyStravaStats`,
      files: [file],
    });
    exportNotice.value = "Share sheet opened successfully.";
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") return;
    exportError.value = error instanceof Error ? error.message : "Unable to share this recap card.";
  } finally {
    isExporting.value = false;
  }
}

function recapInput(page: AnnualRecapPage) {
  return {
    year: selectedYear.value === "All years" ? "YEAR" : selectedYear.value,
    athleteName: includeName.value ? athleteStore.athleteName : "",
    activityLabel: activityLabel.value || "All activities",
    metrics: metrics.value,
    previousMetrics: hasPreviousData.value ? previousMetrics.value : undefined,
    consistencyPercent: consistencyPercent.value,
    previousConsistencyPercent: previousConsistencyPercent.value,
    highlights: highlights.value,
    yearToDate: yearToDate.value,
    daysInScope: yearToDate.value ? elapsedDaysThisYear() : undefined,
    theme: theme.value,
    format: format.value,
    includeMap: includeMap.value,
    tracks: includeMap.value ? mapStore.mapTracks : [],
    page,
  } as const;
}

function commuteRecapInput(page: CommuteRecapPage) {
  return {
    page,
    year: selectedYear.value === "All years" ? "YEAR" : selectedYear.value,
    athleteName: includeName.value ? athleteStore.athleteName : "",
    stats: commuteStats.value,
    previousStats: hasPreviousCommuteData.value ? previousCommuteStats.value : undefined,
    theme: theme.value,
    format: format.value,
    impactEnabled: impactEnabled.value,
    impactConfig: impactConfig.value,
    includeMap: includeMap.value,
    tracks: includeMap.value ? mapStore.mapTracks : [],
    yearToDate: yearToDate.value,
  } as const;
}

async function fetchCommuteActivities() {
  const requestId = ++commuteRequestId;
  commuteLoading.value = true;
  commuteError.value = "";
  try {
    const activities = await requestJson<Activity[]>(buildFilteredApiUrl("activities", "Commute", "All years"));
    if (requestId === commuteRequestId) commuteActivities.value = activities;
  } catch (error) {
    if (requestId === commuteRequestId) {
      commuteError.value = error instanceof Error ? error.message : "Unable to load commute activities.";
    }
  } finally {
    if (requestId === commuteRequestId) commuteLoading.value = false;
  }
}

async function fetchAllYearsData() {
  const requestId = ++allYearsRequestId;
  allYearsLoading.value = true;
  allYearsError.value = "";
  try {
    const data = await requestJson<DashboardData>(buildFilteredApiUrl(
      "dashboard",
      contextStore.currentActivityType,
      "All years",
    ));
    if (requestId === allYearsRequestId) allYearsData.value = data;
  } catch (error) {
    if (requestId === allYearsRequestId) {
      allYearsError.value = error instanceof Error ? error.message : "Unable to load year-over-year data.";
    }
  } finally {
    if (requestId === allYearsRequestId) allYearsLoading.value = false;
  }
}

function metricsForYear(data: DashboardData, year: string): AnnualRecapMetrics {
  return {
    activities: positiveValue(data.nbActivitiesByYear[year]),
    activeDays: positiveValue(data.activeDaysByYear[year]),
    distanceKm: positiveValue(data.totalDistanceByYear[year]),
    elevationM: positiveValue(data.totalElevationByYear[year]),
    movingTimeSeconds: positiveValue(data.movingTimeByYear[year]),
    longestActivityKm: positiveValue(data.maxDistanceByYear[year]),
    longestActivityDate: data.maxDistanceDateByYear[year],
  };
}

function selectAdjacentPage(offset: number) {
  const count = pageDefinitions.value.length;
  const nextIndex = (selectedPageIndex.value + offset + count) % count;
  selectedPage.value = pageDefinitions.value[nextIndex]?.id ?? "overview";
}

function renderPageBlob(page: AnnualRecapPage | CommuteRecapPage): Promise<Blob> {
  const svg = pageSvgs.value.get(page) ?? (isCommuteMode.value
    ? buildCommuteRecapSvg(commuteRecapInput(page as CommuteRecapPage))
    : buildAnnualRecapSvg(recapInput(page as AnnualRecapPage)));
  return annualRecapSvgToPng(
    svg,
    currentFormat.value.width,
    currentFormat.value.height,
  );
}

function downloadBlob(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function fileName(page: AnnualRecapPage | CommuteRecapPage): string {
  const index = pageDefinitions.value.findIndex((candidate) => candidate.id === page) + 1;
  const kind = isCommuteMode.value ? "commute-recap" : "recap";
  return `mystravastats-${selectedYear.value}-${kind}-${String(index).padStart(2, "0")}-${page}-${format.value}.png`;
}

function elapsedDaysThisYear(): number {
  const now = new Date();
  const start = new Date(now.getFullYear(), 0, 1);
  return Math.floor((now.getTime() - start.getTime()) / 86_400_000) + 1;
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
        <h1>{{ isCommuteMode ? "Commute recap" : "Annual recap · Version 2" }}</h1>
        <p v-if="isCommuteMode">Turn explicitly tagged commutes into a five-card story about distance, regularity, progress and optional impact estimates.</p>
        <p v-else>Create a five-card story with year-over-year progress, consistency, exploration and automatic highlights.</p>
      </div>
      <div class="recap-heading__actions">
        <RouterLink
          class="btn btn-outline-primary btn-sm"
          :to="isCommuteMode ? '/annual-recap' : '/commute-recap'"
          @click="isCommuteMode ? selectedSport = ALL_ACTIVITY_TYPE_FILTER : undefined"
        >
          <i :class="isCommuteMode ? 'fa-solid fa-person-running' : 'fa-solid fa-briefcase'" aria-hidden="true" />
          {{ isCommuteMode ? "Sport recap" : "Commute recap" }}
        </RouterLink>
        <RouterLink class="btn btn-outline-secondary btn-sm" to="/dashboard">
          <i class="fa-solid fa-arrow-left" aria-hidden="true" /> Dashboard
        </RouterLink>
      </div>
    </header>

    <div class="recap-layout">
      <aside class="recap-controls" :aria-label="isCommuteMode ? 'Commute recap settings' : 'Annual recap settings'">
        <section>
          <span class="step-number">1</span>
          <div>
            <h2>Choose your data</h2>
            <label for="recap-year">Year</label>
            <select id="recap-year" v-model="selectedYear" class="form-select">
              <option v-for="year in availableYears" :key="year" :value="year">{{ year }}</option>
              <option v-if="selectedYear === 'All years'" value="All years">Select a year</option>
            </select>
            <template v-if="!isCommuteMode">
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
            </template>
            <div v-else class="commute-source">
              <i class="fa-solid fa-circle-check" aria-hidden="true" />
              <span><strong>Explicit commutes only</strong><small>Uses Strava activities marked “Commute”; no inferred home or workplace.</small></span>
            </div>
          </div>
        </section>

        <section>
          <span class="step-number">2</span>
          <div>
            <h2>Choose a card</h2>
            <div class="card-choice-grid">
              <button
                v-for="(page, index) in pageDefinitions"
                :key="page.id"
                type="button"
                :class="{ active: selectedPage === page.id }"
                :aria-pressed="selectedPage === page.id"
                @click="selectedPage = page.id"
              >
                <span>{{ index + 1 }}</span>
                {{ page.label }}
              </button>
            </div>
            <p class="selection-note">All five cards are included when you choose “Download all”.</p>
          </div>
        </section>

        <section>
          <span class="step-number">3</span>
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
          <span class="step-number">4</span>
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
            <template v-if="isCommuteMode">
              <label class="check-row">
                <input v-model="impactEnabled" type="checkbox">
                <span><strong>Enable impact estimates</strong><small>Off by default. Use only for trips that replaced a motorized journey.</small></span>
              </label>
              <div v-if="impactEnabled" class="impact-settings">
                <label>Motorized trips replaced (%)<input v-model.number="impactConfig.motorizedSharePercent" type="number" min="0" max="100" step="5"></label>
                <label>Fuel use (L/100 km)<input v-model.number="impactConfig.fuelConsumptionLitresPer100Km" type="number" min="0" step="0.1"></label>
                <label>Fuel price (€/L)<input v-model.number="impactConfig.fuelPricePerLitre" type="number" min="0" step="0.01"></label>
                <label>CO₂ (kg/km)<input v-model.number="impactConfig.co2KgPerKm" type="number" min="0" step="0.001"></label>
              </div>
            </template>
            <p v-if="includeMap && mapStore.isLoading" class="selection-note">Loading GPS traces…</p>
            <p v-else-if="includeMap && mapStore.error" class="selection-note selection-note--error">{{ mapStore.error }}</p>
          </div>
        </section>

        <p v-if="dashboardStore.error" class="recap-error">{{ dashboardStore.error }}</p>
        <p v-else-if="allYearsError" class="recap-error">{{ allYearsError }}</p>
        <p v-else-if="isCommuteMode && commuteError" class="recap-error">{{ commuteError }}</p>
        <p v-else-if="selectedYear === 'All years'" class="recap-error">Choose a specific year to create a recap.</p>
        <p v-else-if="!isLoading && !hasData" class="recap-error">No {{ isCommuteMode ? "commutes" : "activities" }} are available for this selection.</p>
        <p v-if="exportError" class="recap-error">{{ exportError }}</p>
        <p v-if="exportNotice" class="recap-notice" role="status">{{ exportNotice }}</p>

        <div class="recap-actions">
          <button
            type="button"
            class="btn btn-primary recap-download"
            :disabled="isLoading || !hasData || isExporting"
            @click="downloadCurrentPng"
          >
            <i class="fa-solid fa-download" aria-hidden="true" />
            {{ isExporting ? "Creating…" : `Download ${selectedPageLabel}` }}
          </button>
          <button
            type="button"
            class="btn btn-outline-primary recap-download"
            :disabled="isLoading || !hasData || isExporting"
            @click="downloadAllPngs"
          >
            <i class="fa-solid fa-images" aria-hidden="true" />
            Download all · {{ pageDefinitions.length }} PNGs
          </button>
          <button
            v-if="supportsNativeShare"
            type="button"
            class="btn btn-outline-secondary recap-download"
            :disabled="isLoading || !hasData || isExporting"
            @click="shareCurrentCard"
          >
            <i class="fa-solid fa-share-nodes" aria-hidden="true" />
            Share current card
          </button>
        </div>
      </aside>

      <main class="recap-preview-column">
        <div class="preview-heading">
          <div><span>Card {{ selectedPageIndex + 1 }} of {{ pageDefinitions.length }}</span><strong>{{ selectedPageLabel }}</strong></div>
          <span>{{ currentFormat.label }}</span>
        </div>
        <div class="carousel-stage">
          <button type="button" class="carousel-arrow carousel-arrow--previous" aria-label="Previous recap card" @click="selectAdjacentPage(-1)">
            <i class="fa-solid fa-chevron-left" aria-hidden="true" />
          </button>
          <div
            class="recap-preview"
            :class="`recap-preview--${format}`"
            :aria-busy="isLoading"
            v-html="previewSvg"
          />
          <button type="button" class="carousel-arrow carousel-arrow--next" aria-label="Next recap card" @click="selectAdjacentPage(1)">
            <i class="fa-solid fa-chevron-right" aria-hidden="true" />
          </button>
        </div>
        <div class="carousel-dots" role="group" aria-label="Recap cards">
          <button
            v-for="page in pageDefinitions"
            :key="page.id"
            type="button"
            :class="{ active: selectedPage === page.id }"
            :aria-label="`Show ${page.label} card`"
            :aria-pressed="selectedPage === page.id"
            @click="selectedPage = page.id"
          />
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.recap-page { display: flex; flex-direction: column; gap: 18px; }
.recap-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; padding: 20px 22px; border: 1px solid var(--ms-border); border-radius: 16px; background: linear-gradient(135deg, #fff 0%, #fff7f1 100%); }
.recap-heading h1 { margin: 2px 0 5px; font-size: 1.55rem; font-weight: 850; }
.recap-heading p { max-width: 720px; margin: 0; color: var(--ms-text-muted); }
.recap-heading__actions { display: flex; flex: none; gap: 7px; }
.recap-kicker { color: var(--ms-primary) !important; font-size: .68rem; font-weight: 850; letter-spacing: .1em; text-transform: uppercase; }
.recap-layout { display: grid; grid-template-columns: minmax(300px, 390px) minmax(0, 1fr); align-items: start; gap: 18px; }
.recap-controls { display: flex; flex-direction: column; gap: 12px; }
.recap-controls section { display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 10px; padding: 15px; border: 1px solid var(--ms-border); border-radius: 14px; background: #fff; }
.recap-controls h2 { margin: 2px 0 13px; font-size: .9rem; }
.recap-controls label:not(.check-row) { display: block; margin: 12px 0 5px; color: var(--ms-text-muted); font-size: .68rem; font-weight: 800; text-transform: uppercase; }
.step-number { display: inline-flex; width: 30px; height: 30px; align-items: center; justify-content: center; border-radius: 50%; color: #fff; background: var(--ms-primary); font-size: .75rem; font-weight: 850; }
.choice-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }
.choice-grid button,.theme-switch button,.card-choice-grid button { min-height: 36px; padding: 7px 8px; border: 1px solid #dde1e7; border-radius: 8px; color: #565c66; background: #fafbfc; font-size: .72rem; font-weight: 750; }
.choice-grid button.active,.theme-switch button.active,.card-choice-grid button.active { border-color: #f2a17e; color: #a73c0d; background: #fff0e8; box-shadow: inset 0 0 0 1px #ffd6c4; }
.card-choice-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }
.card-choice-grid button { display: flex; align-items: center; gap: 7px; text-align: left; }
.card-choice-grid button span { display: inline-flex; width: 20px; height: 20px; flex: none; align-items: center; justify-content: center; border-radius: 50%; color: #8a4c31; background: #fbe5da; font-size: .62rem; }
.card-choice-grid button.active span { color: #fff; background: var(--ms-primary); }
.theme-switch { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.check-row { display: flex; align-items: flex-start; gap: 9px; padding: 9px 0; border-top: 1px solid #eef0f3; }
.check-row input { margin-top: 3px; accent-color: var(--ms-primary); }
.check-row span { display: flex; flex-direction: column; gap: 2px; }
.check-row strong { font-size: .75rem; }
.check-row small,.selection-note { color: var(--ms-text-muted); font-size: .64rem; }
.commute-source { display: flex; gap: 9px; margin-top: 13px; padding: 10px; border: 1px solid #b9ddce; border-radius: 9px; color: #176d50; background: #f1fbf6; }
.commute-source i { margin-top: 2px; }
.commute-source span { display: flex; flex-direction: column; gap: 2px; }
.commute-source strong { font-size: .72rem; }
.commute-source small { color: var(--ms-text-muted); font-size: .63rem; }
.impact-settings { display: grid; grid-template-columns: 1fr 1fr; gap: 7px; margin-top: 5px; padding: 10px; border-radius: 9px; background: #f5f7f8; }
.recap-controls .impact-settings label { margin: 0; font-size: .59rem; }
.impact-settings input { width: 100%; margin-top: 4px; padding: 6px 7px; border: 1px solid #d7dce3; border-radius: 7px; background: #fff; font-size: .7rem; }
.selection-note { margin: 8px 0 0; }
.selection-note--error,.recap-error { color: #9f2f23; }
.recap-error { margin: 0; padding: 9px 11px; border: 1px solid #efc1b9; border-radius: 9px; background: #fff5f3; font-size: .7rem; }
.recap-notice { margin: 0; padding: 9px 11px; border: 1px solid #a9d6c3; border-radius: 9px; color: #176d50; background: #f1fbf6; font-size: .7rem; }
.recap-actions { display: flex; flex-direction: column; gap: 7px; }
.recap-download { width: 100%; min-height: 44px; font-weight: 800; }
.recap-preview-column { position: sticky; top: 64px; min-width: 0; padding: 15px; border: 1px solid var(--ms-border); border-radius: 16px; background: #edf0f4; }
.preview-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; color: var(--ms-text-muted); font-size: .68rem; }
.preview-heading div { display: flex; gap: 8px; }
.preview-heading strong { color: var(--ms-text); }
.recap-preview { width: min(100%, 640px); overflow: hidden; margin: 0 auto; border-radius: 8px; box-shadow: 0 18px 42px rgb(31 41 55 / 18%); line-height: 0; }
.recap-preview :deep(svg) { display: block; width: 100%; height: auto; }
.recap-preview--story { width: min(100%, 430px); }
.recap-preview--square { width: min(100%, 650px); }
.carousel-stage { position: relative; display: flex; align-items: center; justify-content: center; min-width: 0; }
.carousel-arrow { position: absolute; z-index: 2; display: inline-flex; width: 38px; height: 38px; align-items: center; justify-content: center; border: 1px solid #d7dce3; border-radius: 50%; color: #4e5560; background: rgb(255 255 255 / 92%); box-shadow: 0 5px 14px rgb(31 41 55 / 15%); }
.carousel-arrow:hover,.carousel-arrow:focus-visible { color: var(--ms-primary); border-color: #f3ae90; outline: none; }
.carousel-arrow--previous { left: 4px; }
.carousel-arrow--next { right: 4px; }
.carousel-dots { display: flex; justify-content: center; gap: 7px; margin-top: 13px; }
.carousel-dots button { width: 8px; height: 8px; padding: 0; border: 0; border-radius: 999px; background: #aeb5bf; transition: width .15s ease, background .15s ease; }
.carousel-dots button.active { width: 24px; background: var(--ms-primary); }
@media (max-width: 900px) { .recap-layout { grid-template-columns: 1fr; }.recap-preview-column { position: static; }.recap-controls { order: 2; }.recap-preview-column { order: 1; } }
@media (max-width: 560px) { .recap-heading { flex-direction: column; }.recap-heading__actions { width: 100%; flex-wrap: wrap; }.recap-layout { gap: 12px; }.choice-grid { grid-template-columns: 1fr; }.recap-preview-column { padding: 8px; }.preview-heading > span { display: none; }.carousel-arrow { width: 34px; height: 34px; }.carousel-arrow--previous { left: 0; }.carousel-arrow--next { right: 0; } }
</style>

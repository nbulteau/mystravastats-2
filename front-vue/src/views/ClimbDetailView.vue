<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { useBadgesStore } from "@/stores/badges";
import { useContextStore } from "@/stores/context";
import { climbSummitId, climbVariantId } from "@/utils/climb-id";
import {
  climbAscentHistory,
  climbVariantStart,
  climbVariantTitle,
  hardestClimbSectors,
} from "@/utils/climb-detail";
import { buildDetailedClimbProfileSvg } from "@/utils/climb-poster";
import { formatTime } from "@/utils/formatters";
import ClimbAscentComparison from "@/components/ClimbAscentComparison.vue";

const route = useRoute();
const router = useRouter();
const badgesStore = useBadgesStore();
const contextStore = useContextStore();
const loading = ref(true);
const loadError = ref("");

const requestedVariantId = computed(() => String(route.params.variantId ?? ""));
const climbResult = computed(() => badgesStore.famousClimbBadgesCheckResults.find(
  (result) => climbVariantId(result) === requestedVariantId.value,
) ?? null);
const details = computed(() => climbResult.value?.climbDetails ?? null);
const summitVariants = computed(() => {
  if (!climbResult.value) return [];
  const summitId = climbSummitId(climbResult.value);
  return badgesStore.famousClimbBadgesCheckResults
    .filter((result) => result.climbDetails && climbSummitId(result) === summitId)
    .sort((left, right) => climbVariantTitle(left).localeCompare(climbVariantTitle(right), "en"));
});
const title = computed(() => climbResult.value ? climbVariantTitle(climbResult.value) : "Climb unavailable");
const startName = computed(() => climbResult.value ? climbVariantStart(climbResult.value) : "Start unavailable");
const category = computed(() => climbResult.value?.badge.category?.trim().toUpperCase() || "Unavailable");
const profileSvg = computed(() => details.value ? buildDetailedClimbProfileSvg(details.value) : "");
const hardestSectors = computed(() => details.value ? hardestClimbSectors(details.value) : []);
const ascents = computed(() => climbResult.value ? climbAscentHistory(climbResult.value) : []);
const bestAscentId = computed(() => details.value?.bestAscent?.activityId ?? null);
const yearLabel = computed(() => contextStore.currentYear === "All years" ? "all years" : contextStore.currentYear);

onMounted(async () => {
  contextStore.$patch({ currentView: "badges" });
  try {
    await badgesStore.ensureLoaded();
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : "Unable to load climb details.";
  } finally {
    loading.value = false;
  }
});

function formatInteger(value: number | null | undefined, suffix = ""): string {
  if (value == null || !Number.isFinite(value) || value <= 0) return "Unavailable";
  return `${Math.round(value).toLocaleString("en-US")}${suffix}`;
}

function formatDecimal(value: number | null | undefined, suffix = ""): string {
  if (value == null || !Number.isFinite(value) || value <= 0) return "Unavailable";
  return `${value.toLocaleString("en-US", { minimumFractionDigits: 1, maximumFractionDigits: 1 })}${suffix}`;
}

function formatDate(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value || "Unavailable";
  return parsed.toLocaleDateString("en-GB", { day: "2-digit", month: "long", year: "numeric" });
}

function formatCoordinate(value: number): string {
  return Number.isFinite(value) ? value.toLocaleString("en-US", { minimumFractionDigits: 5, maximumFractionDigits: 5 }) : "Unavailable";
}

function sectorTone(gradient: number): string {
  if (gradient >= 12) return "sector--extreme";
  if (gradient >= 9) return "sector--hard";
  if (gradient >= 6) return "sector--steady";
  if (gradient >= 3) return "sector--moderate";
  return "sector--easy";
}

function sourceLabel(sourceUrl?: string | null): string {
  if (!sourceUrl) return "Unavailable";
  try {
    return new URL(sourceUrl).hostname.replace(/^www\./, "");
  } catch {
    return sourceUrl;
  }
}
</script>

<template>
  <main class="climb-detail-page">
    <div class="detail-navigation">
      <button type="button" class="back-button" @click="router.back()">
        <i class="fa-solid fa-arrow-left" aria-hidden="true" /> Back
      </button>
      <RouterLink to="/badges" class="climb-log-link">Climb badges</RouterLink>
    </div>

    <div v-if="loading" class="detail-state">
      <span class="spinner-border spinner-border-sm" aria-hidden="true" /> Loading climb details…
    </div>
    <div v-else-if="loadError" class="detail-state detail-state--error">
      <strong>Climb details could not be loaded.</strong>
      <span>{{ loadError }}</span>
    </div>
    <div v-else-if="!climbResult || !details" class="detail-state detail-state--error">
      <strong>Climb side not found.</strong>
      <span>The current filter does not include this climb, or its identifier has changed.</span>
      <RouterLink to="/badges" class="btn btn-sm btn-outline-primary">Back to climbs</RouterLink>
    </div>

    <template v-else>
      <header class="detail-hero">
        <div>
          <p class="detail-kicker">{{ details.country }} · {{ details.massif }} · start {{ startName }}</p>
          <h1>{{ title }}</h1>
          <p class="detail-subtitle">
            Summit {{ formatInteger(details.summitAltitude, " m") }} · category {{ category }} ·
            difficulty {{ formatInteger(details.difficulty, " pts") }}
          </p>
        </div>
        <div class="hero-status" :class="{ 'hero-status--climbed': details.ascentCount > 0 }">
          <strong>{{ details.ascentCount }}</strong>
          <span>ascent{{ details.ascentCount > 1 ? "s" : "" }} · {{ yearLabel }}</span>
        </div>
      </header>

      <section class="detail-card profile-card" aria-labelledby="profile-title">
        <div class="card-heading">
          <div>
            <p class="section-kicker">Catalogue data</p>
            <h2 id="profile-title">Kilometre-by-kilometre profile</h2>
          </div>
          <div class="gradient-legend" aria-label="Gradient colour legend">
            <span class="legend-easy">&lt; 3 %</span>
            <span class="legend-moderate">3–6 %</span>
            <span class="legend-steady">6–9 %</span>
            <span class="legend-hard">9–12 %</span>
            <span class="legend-extreme">≥ 12 %</span>
          </div>
        </div>
        <div class="profile-graphic" v-html="profileSvg" />
        <p v-if="details.profile.length < 2" class="missing-note">
          Profile unavailable: no artificial shape is generated from insufficient data.
        </p>
      </section>

      <div class="detail-columns">
        <section class="detail-card" aria-labelledby="catalogue-title">
          <div class="card-heading">
            <div>
              <p class="section-kicker">Climb side reference</p>
              <h2 id="catalogue-title">Catalogue characteristics</h2>
            </div>
          </div>
          <dl class="metrics-grid">
            <div><dt>Length</dt><dd>{{ formatDecimal(details.lengthKm, " km") }}</dd></div>
            <div><dt>Elevation gain</dt><dd>{{ formatInteger(details.totalAscent, " m") }}</dd></div>
            <div><dt>Average gradient</dt><dd>{{ formatDecimal(details.averageGradient, " %") }}</dd></div>
            <div><dt>Maximum gradient</dt><dd>{{ formatDecimal(details.maximumGradient, " %") }}</dd></div>
            <div><dt>Start altitude</dt><dd>{{ formatInteger(details.minimumAltitude, " m") }}</dd></div>
            <div><dt>Maximum altitude</dt><dd>{{ formatInteger(details.summitAltitude, " m") }}</dd></div>
            <div><dt>Difficulty</dt><dd>{{ formatInteger(details.difficulty, " pts") }}</dd></div>
            <div><dt>Category</dt><dd>{{ category }}</dd></div>
          </dl>
          <div class="source-panel">
            <div>
              <small>Source</small>
              <a v-if="details.sourceUrl" :href="details.sourceUrl" target="_blank" rel="noreferrer">
                {{ sourceLabel(details.sourceUrl) }} <i class="fa-solid fa-arrow-up-right-from-square" aria-hidden="true" />
              </a>
              <strong v-else>Unavailable</strong>
            </div>
            <div><small>Last verified</small><strong>Unavailable</strong></div>
          </div>
          <dl class="coordinate-list">
            <div><dt>Start</dt><dd>{{ formatCoordinate(details.startCoordinate.latitude) }}, {{ formatCoordinate(details.startCoordinate.longitude) }}</dd></div>
            <div><dt>Summit</dt><dd>{{ formatCoordinate(details.summitCoordinate.latitude) }}, {{ formatCoordinate(details.summitCoordinate.longitude) }}</dd></div>
          </dl>
        </section>

        <section class="detail-card" aria-labelledby="sectors-title">
          <div class="card-heading">
            <div>
              <p class="section-kicker">Profile analysis</p>
              <h2 id="sectors-title">Hardest sectors</h2>
            </div>
          </div>
          <ol v-if="hardestSectors.length" class="sector-list">
            <li v-for="sector in hardestSectors" :key="`${sector.startKm}-${sector.endKm}`">
              <span class="sector-rank" :class="sectorTone(sector.averageGradient)">
                {{ sector.averageGradient.toLocaleString("en-US", { maximumFractionDigits: 1 }) }} %
              </span>
              <div>
                <strong>Km {{ sector.startKm.toLocaleString("en-US", { maximumFractionDigits: 1 }) }}–{{ sector.endKm.toLocaleString("en-US", { maximumFractionDigits: 1 }) }}</strong>
                <small>+{{ sector.elevationGain.toLocaleString("en-US") }} m over {{ (sector.endKm - sector.startKm).toLocaleString("en-US", { maximumFractionDigits: 1 }) }} km</small>
              </div>
            </li>
          </ol>
          <p v-else class="missing-note">Sectors unavailable because no usable profile was found.</p>
        </section>
      </div>

      <section class="detail-card personal-card" aria-labelledby="ascents-title">
        <div class="card-heading">
          <div>
            <p class="section-kicker section-kicker--personal">Personal data · {{ yearLabel }}</p>
            <h2 id="ascents-title">My ascents on this side</h2>
          </div>
          <span class="ascent-count">{{ ascents.length }} recorded</span>
        </div>
        <div v-if="ascents.length" class="ascent-table-wrap">
          <table class="ascent-table">
            <thead>
              <tr>
                <th>Date and activity</th>
                <th>Time</th>
                <th>VAM</th>
                <th>Speed</th>
                <th>Power</th>
                <th>Heart rate</th>
                <th><span class="visually-hidden">Open</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ascent in ascents" :key="ascent.activityId">
                <td>
                  <strong>{{ formatDate(ascent.date) }}</strong>
                  <span>{{ ascent.activityName || `Activity ${ascent.activityId}` }}</span>
                  <em v-if="ascent.activityId === bestAscentId">Best time</em>
                </td>
                <td>{{ ascent.durationSeconds > 0 ? formatTime(ascent.durationSeconds) : "Unavailable" }}</td>
                <td>{{ formatInteger(ascent.vamMetersPerHour, " m/h") }}</td>
                <td>{{ formatDecimal(ascent.averageSpeedKph, " km/h") }}</td>
                <td>{{ formatInteger(ascent.averagePowerWatts, " W") }}</td>
                <td>{{ formatInteger(ascent.averageHeartRateBpm, " bpm") }}</td>
                <td>
                  <RouterLink :to="`/activities/${ascent.activityId}`" class="activity-link" :aria-label="`Open activity from ${formatDate(ascent.date)}`">
                    <i class="fa-solid fa-arrow-right" aria-hidden="true" />
                  </RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="missing-note">No personal ascents for {{ yearLabel }}. Catalogue data remains available above.</p>
      </section>

      <ClimbAscentComparison
        :ascents="ascents"
        :best-ascent-id="bestAscentId"
        :length-km="details.lengthKm"
      />

      <section class="detail-card variants-card" aria-labelledby="variants-title">
        <div class="card-heading">
          <div>
            <p class="section-kicker">Same summit</p>
            <h2 id="variants-title">Other sides of {{ details.name }}</h2>
          </div>
          <span>{{ summitVariants.length }} variant{{ summitVariants.length > 1 ? "s" : "" }}</span>
        </div>
        <div class="variant-grid">
          <RouterLink
            v-for="variant in summitVariants"
            :key="climbVariantId(variant)"
            :to="{ name: 'climb-detail', params: { variantId: climbVariantId(variant) } }"
            class="variant-link"
            :class="{ 'variant-link--active': climbVariantId(variant) === requestedVariantId }"
          >
            <span>{{ climbVariantStart(variant) }}</span>
            <strong>{{ formatDecimal(variant.climbDetails?.lengthKm, " km") }} · +{{ formatInteger(variant.climbDetails?.totalAscent, " m") }}</strong>
            <small>{{ variant.climbDetails?.ascentCount ?? 0 }} ascent{{ (variant.climbDetails?.ascentCount ?? 0) > 1 ? "s" : "" }}</small>
          </RouterLink>
        </div>
      </section>
    </template>
  </main>
</template>

<style scoped>
.climb-detail-page {
  width: min(1480px, calc(100% - 32px));
  margin: 0 auto;
  padding: 22px 0 52px;
}

.detail-navigation,
.card-heading,
.detail-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.detail-navigation { margin-bottom: 16px; }
.back-button,
.climb-log-link {
  border: 0;
  color: var(--ms-text-muted);
  background: transparent;
  font-size: 0.82rem;
  font-weight: 750;
  text-decoration: none;
}
.back-button { display: inline-flex; align-items: center; gap: 8px; padding: 6px 0; }
.back-button:hover,
.climb-log-link:hover { color: var(--ms-primary); }

.detail-hero {
  align-items: flex-end;
  padding: 28px 30px;
  border: 1px solid #d9d2c7;
  border-radius: 20px;
  background: linear-gradient(125deg, #f7f1e7 0%, #fffdf8 54%, #edf7f1 100%);
  box-shadow: var(--ms-shadow-soft);
}
.detail-kicker,
.section-kicker { margin: 0 0 6px; color: #9f3709; font-size: 0.7rem; font-weight: 850; letter-spacing: 0.09em; text-transform: uppercase; }
.detail-hero h1 { max-width: 1050px; margin: 0; font-size: clamp(1.65rem, 3vw, 2.8rem); line-height: 1.08; }
.detail-subtitle { margin: 12px 0 0; color: var(--ms-text-muted); font-size: 0.94rem; font-weight: 650; }
.hero-status { display: flex; min-width: 150px; flex-direction: column; padding: 14px 18px; border: 1px solid #d9e0e8; border-radius: 14px; background: rgba(255,255,255,.72); text-align: center; }
.hero-status strong { font-size: 1.8rem; line-height: 1; }
.hero-status span { margin-top: 5px; color: var(--ms-text-muted); font-size: 0.72rem; }
.hero-status--climbed { border-color: #aad7c3; color: #176d50; background: #f2fbf6; }

.detail-card { margin-top: 16px; padding: 20px 22px; border: 1px solid var(--ms-border); border-radius: 17px; background: var(--ms-surface-strong); box-shadow: var(--ms-shadow-soft); }
.card-heading { align-items: flex-start; margin-bottom: 15px; }
.card-heading h2 { margin: 0; font-size: 1.12rem; }
.section-kicker--personal { color: #176d50; }
.profile-graphic { width: 100%; overflow: hidden; border-radius: 13px; background: #fbf8f1; }
.profile-graphic :deep(svg) { display: block; width: 100%; height: auto; }
.gradient-legend { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 5px; }
.gradient-legend span { padding: 3px 7px; border-radius: 5px; color: #1e2a30; font-size: 0.62rem; font-weight: 750; }
.legend-easy { background: #7bc87c; }.legend-moderate { background: #50a7c8; }.legend-steady { background: #f7d447; }.legend-hard { color: white !important; background: #e4543d; }.legend-extreme { color: white !important; background: #9f2031; }

.detail-columns { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(320px, .75fr); gap: 16px; }
.metrics-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; margin: 0; }
.metrics-grid div { min-height: 76px; padding: 11px; border: 1px solid #e5ded4; border-radius: 11px; background: #fbf8f1; }
.metrics-grid dt,
.coordinate-list dt { color: var(--ms-text-muted); font-size: 0.65rem; font-weight: 800; letter-spacing: .04em; text-transform: uppercase; }
.metrics-grid dd { margin: 7px 0 0; font-size: 1rem; font-weight: 800; }
.source-panel { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-top: 8px; }
.source-panel > div { display: flex; flex-direction: column; padding: 10px 11px; border-left: 3px solid #cfbea9; background: #faf8f4; }
.source-panel small { color: var(--ms-text-muted); font-size: .65rem; text-transform: uppercase; }
.source-panel strong,
.source-panel a { margin-top: 2px; font-size: .8rem; font-weight: 750; }
.coordinate-list { display: flex; flex-wrap: wrap; gap: 10px 24px; margin: 14px 0 0; }
.coordinate-list div { display: flex; align-items: baseline; gap: 7px; }
.coordinate-list dd { margin: 0; color: var(--ms-text-muted); font: 600 .74rem ui-monospace, monospace; }

.sector-list { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }
.sector-list li { display: flex; align-items: center; gap: 10px; padding: 8px; border-radius: 10px; background: #f7f8fa; }
.sector-rank { min-width: 66px; padding: 7px 5px; border-radius: 7px; color: #172129; font-size: .76rem; font-weight: 850; text-align: center; }
.sector--easy { background: #7bc87c; }.sector--moderate { background: #50a7c8; }.sector--steady { background: #f7d447; }.sector--hard { color: white; background: #e4543d; }.sector--extreme { color: white; background: #9f2031; }
.sector-list div { display: flex; min-width: 0; flex-direction: column; }
.sector-list strong { font-size: .78rem; }.sector-list small { color: var(--ms-text-muted); font-size: .68rem; }

.personal-card { border-color: #cce4da; background: linear-gradient(180deg, #fff 0%, #f8fcfa 100%); }
.ascent-count { padding: 5px 10px; border-radius: 999px; color: #176d50; background: #e8f7ef; font-size: .72rem; font-weight: 800; }
.ascent-table-wrap { overflow-x: auto; }
.ascent-table { width: 100%; min-width: 940px; border-collapse: collapse; }
.ascent-table th { padding: 8px 10px; color: var(--ms-text-muted); font-size: .65rem; letter-spacing: .04em; text-align: left; text-transform: uppercase; }
.ascent-table td { padding: 11px 10px; border-top: 1px solid #dfeae5; font-size: .78rem; font-weight: 650; }
.ascent-table td:first-child { display: flex; min-width: 190px; flex-direction: column; }
.ascent-table td:first-child span { color: var(--ms-text-muted); font-size: .7rem; font-weight: 500; }
.ascent-table em { width: max-content; margin-top: 3px; padding: 2px 6px; border-radius: 999px; color: #9f3709; background: #fff0e8; font-size: .58rem; font-style: normal; font-weight: 850; text-transform: uppercase; }
.activity-link { display: inline-grid; width: 30px; height: 30px; place-items: center; border-radius: 50%; color: white; background: var(--ms-primary); text-decoration: none; }

.variant-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.variant-link { display: flex; min-width: 0; flex-direction: column; padding: 12px; border: 1px solid var(--ms-border); border-radius: 11px; color: var(--ms-text); background: #fafbfc; text-decoration: none; }
.variant-link:hover { border-color: #f2ad8d; transform: translateY(-1px); }
.variant-link--active { border-color: var(--ms-primary); box-shadow: inset 3px 0 0 var(--ms-primary); background: #fff6f1; }
.variant-link span { overflow: hidden; font-size: .82rem; font-weight: 800; text-overflow: ellipsis; white-space: nowrap; }
.variant-link strong { margin-top: 5px; font-size: .72rem; }.variant-link small { color: var(--ms-text-muted); font-size: .66rem; }
.missing-note { margin: 6px 0 0; padding: 12px; border: 1px dashed #ced6df; border-radius: 9px; color: var(--ms-text-muted); background: #f8fafc; font-size: .78rem; }
.detail-state { display: flex; min-height: 260px; align-items: center; justify-content: center; gap: 10px; border: 1px solid var(--ms-border); border-radius: 16px; background: var(--ms-surface); }
.detail-state--error { flex-direction: column; color: #8f2d20; text-align: center; }

@media (max-width: 980px) {
  .detail-columns { grid-template-columns: 1fr; }
  .metrics-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .variant-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 650px) {
  .climb-detail-page { width: min(100% - 18px, 1480px); padding-top: 12px; }
  .detail-hero,
  .card-heading { align-items: flex-start; flex-direction: column; }
  .detail-hero { padding: 20px; }
  .hero-status { width: 100%; }
  .detail-card { padding: 15px 12px; }
  .gradient-legend { justify-content: flex-start; }
  .source-panel,
  .variant-grid { grid-template-columns: 1fr; }
}
</style>

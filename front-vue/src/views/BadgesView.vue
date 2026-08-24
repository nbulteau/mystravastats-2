<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useBadgesStore } from "@/stores/badges";
import { computed, nextTick, ref, watch, onMounted } from "vue";
import BadgeItem from "@/components/BadgeItem.vue";
import ClimbPosterGenerator from "@/components/ClimbPosterGenerator.vue";
import ClimbMap from "@/components/ClimbMap.vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { climbSummitId, climbVariantId } from "@/utils/climb-map";

type BadgesSectionId = "badges" | "climbs" | "posters" | "map";

const contextStore = useContextStore();
const badgesStore = useBadgesStore();
onMounted(() => contextStore.updateCurrentView("badges"));

const currentYear = computed(() => contextStore.currentYear);
const climbMapYearOptions = computed(() => {
  const options = contextStore.availableYears.length > 0
    ? contextStore.availableYears
    : [currentYear.value, "All years"];
  return [...new Set([currentYear.value, ...options])];
});
const activeSection = ref<BadgesSectionId>("badges");
const focusedMapSummitId = ref<string | null>(null);
const sections: ReadonlyArray<{
  id: BadgesSectionId;
  label: string;
  description: string;
  icon: string;
}> = [
  {
    id: "badges",
    label: "Badges",
    description: "General achievements",
    icon: "fa-solid fa-trophy",
  },
  {
    id: "climbs",
    label: "Climb log",
    description: "Cols and ascents",
    icon: "fa-solid fa-mountain-sun",
  },
  {
    id: "posters",
    label: "Posters",
    description: "Print studio",
    icon: "fa-solid fa-image",
  },
  {
    id: "map",
    label: "Climb map",
    description: "Explore your summits",
    icon: "fa-solid fa-map-location-dot",
  },
];

const sortByProgress = (badges: BadgeCheckResult[]) => {
  return [...badges].sort((a, b) => {
    const progressOrder = Number(b.nbCheckedActivities > 0) - Number(a.nbCheckedActivities > 0);
    if (progressOrder !== 0) {
      return progressOrder;
    }
    return a.badge.label.localeCompare(b.badge.label);
  });
};

const sectionSummary = (badges: BadgeCheckResult[]) => {
  const total = badges.length;
  const acquired = badges.filter((badge) => badge.nbCheckedActivities > 0).length;
  const locked = total - acquired;
  const completion = total > 0 ? Math.round((acquired / total) * 100) : 0;

  return {
    total,
    acquired,
    locked,
    completion,
  };
};

const generalBadgesCheckResults = computed(() => sortByProgress(badgesStore.generalBadgesCheckResults));
const allFamousClimbBadgesCheckResults = computed(() => sortByProgress(badgesStore.famousClimbBadgesCheckResults));
const isBadgeDataUpdating = computed(() => (
  badgesStore.isLoading || badgesStore.loadedFiltersKey !== contextStore.currentFiltersKey
));
const earnedFamousClimbs = computed(() => allFamousClimbBadgesCheckResults.value.filter((result) => (
  result.nbCheckedActivities > 0
)));
const climbAscentCount = computed(() => earnedFamousClimbs.value.reduce((total, result) => (
  total + (result.climbDetails?.ascentCount ?? result.nbCheckedActivities)
), 0));
const climbedMassifCount = computed(() => new Set(
  earnedFamousClimbs.value
    .map((result) => result.climbDetails?.massif?.trim())
    .filter((massif): massif is string => Boolean(massif)),
).size);
const selectedFamousClimbCategory = ref("ALL");

const famousClimbCategoryOptions = computed(() => {
  const categoryOrder = ["HC", "1", "2", "3", "4"];
  const availableCategories = new Set(
    allFamousClimbBadgesCheckResults.value
      .map((badgeCheckResult) => badgeCheckResult.badge.category?.toUpperCase().trim())
      .filter((category): category is string => Boolean(category)),
  );

  const orderedKnownCategories = categoryOrder.filter((category) => availableCategories.has(category));
  const otherCategories = Array.from(availableCategories)
    .filter((category) => !categoryOrder.includes(category))
    .sort((a, b) => a.localeCompare(b));

  return ["ALL", ...orderedKnownCategories, ...otherCategories];
});

watch(famousClimbCategoryOptions, (options) => {
  if (!options.includes(selectedFamousClimbCategory.value)) {
    selectedFamousClimbCategory.value = "ALL";
  }
});

const famousClimbBadgesCheckResults = computed(() => {
  if (selectedFamousClimbCategory.value === "ALL") {
    return allFamousClimbBadgesCheckResults.value;
  }

  return allFamousClimbBadgesCheckResults.value.filter(
    (badgeCheckResult) => badgeCheckResult.badge.category?.toUpperCase().trim() === selectedFamousClimbCategory.value,
  );
});

const generalSummary = computed(() => sectionSummary(generalBadgesCheckResults.value));
const famousSummary = computed(() => sectionSummary(famousClimbBadgesCheckResults.value));

async function selectAdjacentSection(offset: number) {
  const currentIndex = sections.findIndex((section) => section.id === activeSection.value);
  const nextIndex = (currentIndex + offset + sections.length) % sections.length;
  activeSection.value = sections[nextIndex].id;
  await nextTick();
  document.getElementById(`badges-section-${activeSection.value}`)?.focus();
}

async function showClimbOnMap(result: BadgeCheckResult) {
  focusedMapSummitId.value = climbSummitId(result);
  activeSection.value = "map";
  await nextTick();
}

async function openClimbInLog(variantId: string) {
  activeSection.value = "climbs";
  await nextTick();
  document.getElementById(`climb-log-${variantId}`)?.scrollIntoView({ behavior: "smooth", block: "center" });
}
</script>

<template>
  <div class="badges-page">
    <section class="badges-hub" aria-labelledby="badges-hub-title">
      <div class="badges-hub-heading">
        <div>
          <p class="badges-kicker">Achievements &amp; summits</p>
          <h1 id="badges-hub-title">Badges and climb log</h1>
          <p>Follow your progress, revisit every climbed col and turn your best ascents into a poster.</p>
        </div>
        <div class="badges-overview" aria-label="Badge and climb overview">
          <div class="overview-stat">
            <strong>{{ generalSummary.acquired }}</strong>
            <span>badges earned</span>
          </div>
          <div class="overview-stat">
            <strong>{{ earnedFamousClimbs.length }}</strong>
            <span>cols climbed</span>
          </div>
          <div class="overview-stat">
            <strong>{{ climbAscentCount }}</strong>
            <span>ascents</span>
          </div>
          <div class="overview-stat">
            <strong>{{ climbedMassifCount }}</strong>
            <span>massifs</span>
          </div>
        </div>
      </div>

      <div class="badges-sections" role="tablist" aria-label="Badges areas">
        <button
          v-for="section in sections"
          :id="`badges-section-${section.id}`"
          :key="section.id"
          type="button"
          role="tab"
          class="badges-section-tab"
          :class="{ 'badges-section-tab--active': activeSection === section.id }"
          :aria-selected="activeSection === section.id"
          :aria-controls="`badges-panel-${section.id}`"
          :tabindex="activeSection === section.id ? 0 : -1"
          @click="activeSection = section.id"
          @keydown.left.prevent="selectAdjacentSection(-1)"
          @keydown.right.prevent="selectAdjacentSection(1)"
        >
          <i :class="section.icon" aria-hidden="true" />
          <span>
            <strong>{{ section.label }}</strong>
            <small>{{ section.description }}</small>
          </span>
        </button>
      </div>
    </section>

    <section
      v-if="activeSection === 'badges'"
      id="badges-panel-badges"
      class="badges-section"
      role="tabpanel"
      aria-labelledby="badges-section-badges"
    >
      <div class="badges-header">
        <div>
          <p class="badges-section-kicker">Season achievements</p>
          <h2 class="badges-title">General badges for {{ currentYear }}</h2>
        </div>
        <div class="badges-summary">
          <span class="summary-chip summary-chip--earned">Earned {{ generalSummary.acquired }}</span>
          <span class="summary-chip summary-chip--locked">Locked {{ generalSummary.locked }}</span>
          <span class="summary-chip summary-chip--completion">{{ generalSummary.completion }}% completed</span>
        </div>
      </div>
      <div v-if="generalBadgesCheckResults.length" class="row g-3 justify-content-center">
        <div
          v-for="badge in generalBadgesCheckResults"
          :key="badge.badge.label"
          class="col-lg-2 col-md-3 col-sm-4 col-6 d-flex justify-content-center"
        >
          <BadgeItem :badge-check-result="badge" />
        </div>
      </div>
      <div v-else class="chart-empty">No general badges found for the selected filters.</div>
    </section>

    <section
      v-else-if="activeSection === 'climbs'"
      id="badges-panel-climbs"
      class="badges-section"
      role="tabpanel"
      aria-labelledby="badges-section-climbs"
    >
      <div class="badges-header badges-header--split">
        <div>
          <p class="badges-section-kicker">Your summit collection</p>
          <h2 class="badges-title">Famous climb log for {{ currentYear }}</h2>
        </div>
        <div class="badges-header-controls">
          <label for="famous-category-filter" class="category-filter-label">Category</label>
          <select
            id="famous-category-filter"
            v-model="selectedFamousClimbCategory"
            class="form-select form-select-sm category-filter-select"
          >
            <option
              v-for="categoryOption in famousClimbCategoryOptions"
              :key="categoryOption"
              :value="categoryOption"
            >
              {{ categoryOption === "ALL" ? "All categories" : `Cat. ${categoryOption}` }}
            </option>
          </select>
        </div>
        <div class="badges-summary">
          <span class="summary-chip summary-chip--earned">Climbed {{ famousSummary.acquired }}</span>
          <span class="summary-chip summary-chip--locked">To discover {{ famousSummary.locked }}</span>
          <span class="summary-chip summary-chip--completion">{{ famousSummary.completion }}% completed</span>
        </div>
      </div>
      <div v-if="famousClimbBadgesCheckResults.length" class="row g-3 justify-content-center">
        <div
          v-for="badge in famousClimbBadgesCheckResults"
          :id="`climb-log-${climbVariantId(badge)}`"
          :key="climbVariantId(badge)"
          class="climb-log-entry col-lg-2 col-md-3 col-sm-4 col-6 d-flex flex-column align-items-center justify-content-start"
        >
          <BadgeItem :badge-check-result="badge" />
          <RouterLink
            class="show-on-map-button climb-detail-link"
            :to="{ name: 'climb-detail', params: { variantId: climbVariantId(badge) } }"
          >
            <i class="fa-solid fa-file-lines" aria-hidden="true" />
            Detailed sheet
          </RouterLink>
          <button
            v-if="badge.climbDetails?.summitCoordinate"
            type="button"
            class="show-on-map-button"
            @click="showClimbOnMap(badge)"
          >
            <i class="fa-solid fa-location-dot" aria-hidden="true" />
            Show on map
          </button>
        </div>
      </div>
      <div v-else class="chart-empty">No famous climbs found for this category.</div>
    </section>

    <section
      v-else-if="activeSection === 'posters'"
      id="badges-panel-posters"
      class="badges-section poster-workspace"
      role="tabpanel"
      aria-labelledby="badges-section-posters"
    >
      <div class="poster-workspace-copy">
        <span class="workspace-icon" aria-hidden="true"><i class="fa-solid fa-image" /></span>
        <p class="badges-section-kicker">Print studio</p>
        <h2 class="badges-title">Create a poster from your climbed cols</h2>
        <p>
          Choose Altitude, Topographic log or Collection, then compose an SVG poster with up to 50 climbs.
          The hardest and longest selections are available as shortcuts.
        </p>
        <div class="poster-facts" aria-label="Poster generator capabilities">
          <span><strong>3</strong> designs</span>
          <span><strong>50</strong> cols maximum</span>
          <span :class="{ 'poster-fact--loading': isBadgeDataUpdating }">
            <strong>{{ isBadgeDataUpdating ? "…" : earnedFamousClimbs.length }}</strong>
            {{ isBadgeDataUpdating ? "updating" : "available now" }}
          </span>
        </div>
        <ClimbPosterGenerator
          :climbs="allFamousClimbBadgesCheckResults"
          :year-label="currentYear"
          :is-loading="isBadgeDataUpdating"
        />
      </div>
    </section>

    <section
      v-else
      id="badges-panel-map"
      class="badges-section map-workspace"
      role="tabpanel"
      aria-labelledby="badges-section-map"
    >
      <ClimbMap
        :climbs="allFamousClimbBadgesCheckResults"
        :year-label="currentYear"
        :year-options="climbMapYearOptions"
        :category="selectedFamousClimbCategory"
        :focus-summit-id="focusedMapSummitId"
        @update:category="selectedFamousClimbCategory = $event"
        @update:year="contextStore.updateCurrentYear($event)"
        @open-climb-log="openClimbInLog"
      />
    </section>
  </div>
</template>

<style scoped>
.badges-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.badges-hub,
.badges-section {
  border: 1px solid var(--ms-border);
  border-radius: 16px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
}

.badges-hub {
  overflow: hidden;
}

.badges-hub-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
  padding: 20px 22px 18px;
  background:
    radial-gradient(circle at 88% 0%, rgb(252 76 2 / 12%), transparent 34%),
    linear-gradient(135deg, var(--ms-surface-strong), var(--ms-surface));
}

.badges-kicker,
.badges-section-kicker {
  margin: 0 0 4px;
  color: #c4470c;
  font-size: 0.73rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.badges-hub h1 {
  margin: 0;
  color: var(--ms-text);
  font-size: clamp(1.45rem, 2.8vw, 2.05rem);
  font-weight: 800;
}

.badges-hub-heading > div > p:last-child {
  max-width: 620px;
  margin: 6px 0 0;
  color: var(--ms-text-muted);
}

.badges-overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(82px, 1fr));
  gap: 8px;
  flex: 0 1 430px;
}

.overview-stat {
  display: flex;
  min-height: 72px;
  flex-direction: column;
  justify-content: center;
  padding: 9px 10px;
  border: 1px solid rgb(252 76 2 / 18%);
  border-radius: 12px;
  background: rgb(255 255 255 / 74%);
  text-align: center;
}

.overview-stat strong {
  color: #b83d05;
  font-size: 1.22rem;
  line-height: 1.05;
}

.overview-stat span {
  margin-top: 4px;
  color: var(--ms-text-muted);
  font-size: 0.7rem;
  font-weight: 700;
}

.badges-sections {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  padding: 7px;
  border-top: 1px solid var(--ms-border);
  background: var(--ms-surface);
}

.badges-section-tab {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  padding: 11px 13px;
  border: 1px solid transparent;
  border-radius: 11px;
  color: var(--ms-text-muted);
  background: transparent;
  text-align: left;
  transition: color 120ms ease, background 120ms ease, border-color 120ms ease;
}

.badges-section-tab:hover {
  color: var(--ms-text);
  background: rgb(255 255 255 / 72%);
}

.badges-section-tab:focus-visible {
  outline: 3px solid rgb(252 76 2 / 24%);
  outline-offset: 1px;
}

.badges-section-tab--active {
  border-color: #ffc3a7;
  color: #aa3703;
  background: #fff5ef;
  box-shadow: 0 3px 10px rgb(125 52 17 / 8%);
}

.badges-section-tab > i {
  width: 24px;
  color: #d64b0b;
  font-size: 1rem;
  text-align: center;
}

.badges-section-tab span {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.badges-section-tab strong {
  color: inherit;
  font-size: 0.88rem;
}

.badges-section-tab small {
  overflow: hidden;
  font-size: 0.7rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.badges-section {
  padding: 18px 16px;
}

.badges-header {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 14px;
}

.badges-header--split {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
}

.badges-header-controls {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.category-filter-label {
  font-size: 0.76rem;
  font-weight: 800;
  color: var(--ms-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.category-filter-select {
  width: auto;
  min-width: 162px;
  border-radius: 999px;
  border-color: var(--ms-border);
  font-size: 0.85rem;
  font-weight: 600;
  padding-left: 14px;
  padding-right: 30px;
}

.badges-title {
  margin: 0;
  color: var(--ms-text);
  font-size: 1.12rem;
  font-weight: 750;
}

.badges-summary {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.badges-header--split .badges-summary {
  grid-column: 1 / -1;
}

.summary-chip {
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 0.78rem;
  font-weight: 700;
  border: 1px solid transparent;
}

.summary-chip--earned {
  color: #b73b00;
  background: #fff2ea;
  border-color: #ffccb5;
}

.summary-chip--locked {
  color: #616a78;
  background: #f3f5f8;
  border-color: #dbe1ea;
}

.summary-chip--completion {
  color: #9f3709;
  background: #fff7f2;
  border-color: #ffc9b0;
}

.poster-workspace,
.map-workspace {
  display: grid;
  min-height: 350px;
  place-items: center;
  background:
    linear-gradient(135deg, rgb(252 76 2 / 6%), transparent 45%),
    var(--ms-surface-strong);
}

.map-workspace {
  display: block;
  min-width: 0;
  min-height: 0;
  padding: 16px;
}

.poster-workspace-copy,
.map-workspace-copy {
  display: flex;
  max-width: 710px;
  flex-direction: column;
  align-items: center;
  padding: 22px;
  text-align: center;
}

.poster-workspace-copy > p:not(.badges-section-kicker),
.map-workspace-copy > p:not(.badges-section-kicker) {
  margin: 8px 0 18px;
  color: var(--ms-text-muted);
}

.workspace-icon {
  display: grid;
  width: 58px;
  height: 58px;
  margin-bottom: 14px;
  place-items: center;
  border: 1px solid #ffc5aa;
  border-radius: 18px;
  color: #c94408;
  background: #fff3ec;
  font-size: 1.45rem;
  transform: rotate(-3deg);
}

.workspace-icon i {
  transform: rotate(3deg);
}

.workspace-icon--map {
  color: #237a58;
  border-color: #bde4d3;
  background: #eefaf5;
}

.poster-facts {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.poster-facts span {
  padding: 6px 10px;
  border: 1px solid var(--ms-border);
  border-radius: 999px;
  color: var(--ms-text-muted);
  background: var(--ms-surface);
  font-size: 0.78rem;
}

.poster-facts strong {
  color: var(--ms-text);
}

.poster-fact--loading {
  color: #c85d33;
}

.planned-chip {
  margin-bottom: 10px;
  padding: 3px 9px;
  border: 1px solid #bde4d3;
  border-radius: 999px;
  color: #176c4c;
  background: #eefaf5;
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.climb-log-entry {
  gap: 8px;
  scroll-margin-top: 90px;
}

.show-on-map-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 9px;
  border: 1px solid #c8ddd5;
  border-radius: 999px;
  color: #176d50;
  background: #f1faf6;
  font-size: 0.68rem;
  font-weight: 750;
}

.show-on-map-button:hover {
  border-color: #91c9b3;
  background: #e4f6ee;
}

.climb-detail-link {
  color: #9f3709;
  text-decoration: none;
}

@media (max-width: 992px) {
  .badges-hub-heading {
    flex-direction: column;
  }

  .badges-overview {
    width: 100%;
    flex-basis: auto;
  }

  .badges-section-tab small {
    display: none;
  }
}

@media (max-width: 680px) {
  .badges-hub-heading {
    padding: 17px 15px 14px;
  }

  .badges-overview {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .badges-sections {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .badges-header--split {
    grid-template-columns: 1fr;
  }

  .badges-header-controls {
    justify-content: flex-start;
  }

  .badges-header--split .badges-summary {
    grid-column: auto;
  }

  .poster-workspace-copy,
  .map-workspace-copy {
    padding: 8px 0;
  }
}
</style>

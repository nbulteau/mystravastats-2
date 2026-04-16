<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useBadgesStore } from "@/stores/badges";
import { computed, ref, watch, onMounted } from "vue";
import BadgeItem from "@/components/BadgeItem.vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";

const contextStore = useContextStore();
const badgesStore = useBadgesStore();
onMounted(() => contextStore.updateCurrentView("badges"));

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivityType);

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
</script>

<template>
  <div class="badges-page">
    <section
      v-if="generalBadgesCheckResults.length"
      class="badges-section"
    >
      <div class="badges-header">
        <p class="badges-title">
          {{ currentActivity }} general badges for {{ currentYear }}
        </p>
        <div class="badges-summary">
          <span class="summary-chip summary-chip--earned">Earned {{ generalSummary.acquired }}</span>
          <span class="summary-chip summary-chip--locked">Locked {{ generalSummary.locked }}</span>
          <span class="summary-chip summary-chip--completion">{{ generalSummary.completion }}% completed</span>
        </div>
      </div>
      <div class="row g-3 justify-content-center">
        <div
          v-for="badge in generalBadgesCheckResults"
          :key="badge.badge.label"
          class="col-lg-2 col-md-3 col-sm-4 col-6 d-flex justify-content-center"
        >
          <BadgeItem
            :badge-check-result="badge"
          />
        </div>
      </div>
    </section>

    <section
      v-if="famousClimbBadgesCheckResults.length"
      class="badges-section"
    >
      <div class="badges-header">
        <p class="badges-title">
          Famous climb {{ currentActivity }} badges for {{ currentYear }}
        </p>
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
          <span class="summary-chip summary-chip--earned">Earned {{ famousSummary.acquired }}</span>
          <span class="summary-chip summary-chip--locked">Locked {{ famousSummary.locked }}</span>
          <span class="summary-chip summary-chip--completion">{{ famousSummary.completion }}% completed</span>
        </div>
      </div>
      <div class="row g-3 justify-content-center">
        <div
          v-for="badge in famousClimbBadgesCheckResults"
          :key="badge.badge.label"
          class="col-lg-2 col-md-3 col-sm-4 col-6 d-flex justify-content-center"
        >
          <BadgeItem
            :badge-check-result="badge"
          />
        </div>
      </div>
    </section>
    <div
      v-if="!generalBadgesCheckResults.length && !famousClimbBadgesCheckResults.length"
      class="chart-empty"
    >
      No badges found for the selected filters.
    </div>
  </div>
</template>

<style scoped>
.badges-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.badges-section {
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  background: var(--ms-surface-strong);
  box-shadow: var(--ms-shadow-soft);
  padding: 16px 14px;
}

.badges-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.badges-header-controls {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.category-filter-label {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--ms-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.02em;
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
  margin-bottom: 0;
  text-align: center;
  color: var(--ms-text);
  font-size: 1.08rem;
  font-weight: 700;
}

.badges-summary {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
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
</style>

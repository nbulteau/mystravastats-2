<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";
import BadgeItem from "@/components/BadgeItem.vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";

const contextStore = useContextStore();
contextStore.updateCurrentView("badges");

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

const generalBadgesCheckResults = computed(() => sortByProgress(contextStore.generalBadgesCheckResults));
const famousClimbBadgesCheckResults = computed(() => sortByProgress(contextStore.famousClimbBadgesCheckResults));
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
          <span class="summary-chip summary-chip--earned">Acquis {{ generalSummary.acquired }}</span>
          <span class="summary-chip summary-chip--locked">À débloquer {{ generalSummary.locked }}</span>
          <span class="summary-chip summary-chip--completion">{{ generalSummary.completion }}% complété</span>
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
          Famous climbs {{ currentActivity }} badges for {{ currentYear }}
        </p>
        <div class="badges-summary">
          <span class="summary-chip summary-chip--earned">Acquis {{ famousSummary.acquired }}</span>
          <span class="summary-chip summary-chip--locked">À débloquer {{ famousSummary.locked }}</span>
          <span class="summary-chip summary-chip--completion">{{ famousSummary.completion }}% complété</span>
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
  gap: 14px;
}

.badges-section {
  border: 1px solid #d7e2ef;
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(248, 251, 255, 0.9));
  box-shadow: 0 14px 28px rgba(24, 39, 75, 0.08);
  padding: 16px 14px;
}

.badges-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.badges-title {
  margin-bottom: 0;
  text-align: center;
  color: #2a3c52;
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
  color: #5c4300;
  background: #fff2bf;
  border-color: #f0c75a;
}

.summary-chip--locked {
  color: #596c81;
  background: #edf2f8;
  border-color: #c9d4e0;
}

.summary-chip--completion {
  color: #1f5f35;
  background: #e5f6ea;
  border-color: #9ad6ab;
}
</style>

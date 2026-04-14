<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { useBadgesStore } from "@/stores/badges";
import { computed } from "vue";
import BadgeItem from "@/components/BadgeItem.vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";

const contextStore = useContextStore();
const badgesStore = useBadgesStore();
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

const generalBadgesCheckResults = computed(() => sortByProgress(badgesStore.generalBadgesCheckResults));
const famousClimbBadgesCheckResults = computed(() => sortByProgress(badgesStore.famousClimbBadgesCheckResults));
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

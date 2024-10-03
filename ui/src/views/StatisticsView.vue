<script setup lang="ts">
import StatisticsGrid from "@/components/StatisticsGrid.vue";
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("statistics");

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivity);
const statistics = computed(() => contextStore.statistics);
</script>

<template>
  <main>
    <StatisticsGrid
      :year="currentYear"
      :activity="currentActivity"
      :statistics="statistics"
    />
  </main>
</template>

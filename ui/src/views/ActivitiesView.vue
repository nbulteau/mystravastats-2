<script setup lang="ts">
import ActivitiesGrid from "@/components/ActivitiesGrid.vue";
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("activities");

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivityType);
const activities = computed(() => contextStore.activities);


</script>

<template>
  <ActivitiesGrid
    :activities="activities"
    :current-activity="currentActivity"
    :current-year="currentYear"
  />
</template>

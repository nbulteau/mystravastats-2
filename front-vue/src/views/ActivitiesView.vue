<script setup lang="ts">
import ActivitiesGrid from "@/components/ActivitiesGrid.vue";
import { useContextStore } from "@/stores/context.js";
import { useActivitiesStore } from "@/stores/activities";
import { computed, onMounted } from "vue";

const contextStore = useContextStore();
const activitiesStore = useActivitiesStore();
onMounted(() => contextStore.updateCurrentView("activities"));

const currentYear = computed(() => contextStore.currentYear);
const currentActivity = computed(() => contextStore.currentActivityType);
const activities = computed(() => activitiesStore.activities);


</script>

<template>
  <ActivitiesGrid
    :activities="activities"
    :current-activity="currentActivity"
    :current-year="currentYear"
  />
</template>

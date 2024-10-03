<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("badges");

const year = computed(() => contextStore.currentYear);
const activity = computed(() => contextStore.currentActivity);
const statistics = computed(() => contextStore.statistics);
</script>

<template>
  <main>
  </main>
</template>



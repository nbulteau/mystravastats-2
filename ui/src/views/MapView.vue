<script setup lang="ts">
import { computed } from "vue";
import { useContextStore } from "@/stores/context.js";
import AllTracksMap from "@/components/AllTracksMap.vue";

const contextStore = useContextStore();
contextStore.updateCurrentView("map");

// Get the GPX coordinates from the context store
const gpxCoordinates = computed(() => contextStore.gpxCoordinates);

</script>

<template>
  <AllTracksMap :gpx-coordinates="gpxCoordinates" />
</template>

<style></style>

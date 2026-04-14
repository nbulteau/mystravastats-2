<script setup lang="ts">
import { computed } from "vue";
import { useContextStore } from "@/stores/context.js";
import { useMapStore } from "@/stores/map";
import AllTracksMap from "@/components/AllTracksMap.vue";

const contextStore = useContextStore();
const mapStore = useMapStore();
contextStore.updateCurrentView("map");

// Get the GPX coordinates from the context store
const gpxCoordinates = computed(() => mapStore.gpxCoordinates);

</script>

<template>
  <AllTracksMap :gpx-coordinates="gpxCoordinates" />
</template>

<style></style>

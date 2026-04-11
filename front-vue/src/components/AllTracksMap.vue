<script setup lang="ts">
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { watch, ref, onBeforeUnmount, onMounted } from "vue";

const props = defineProps<{
  gpxCoordinates: number[][][];
}>();

const map = ref<L.Map>();
const mapContainer = ref<HTMLDivElement | null>(null);

const isValidCoordinate = (coord: number[]) =>
  Number.isFinite(coord[0]) && Number.isFinite(coord[1]);

const initMap = () => {
  if (mapContainer.value) {
    if (map.value) {
      map.value.remove();
      map.value = undefined;
    }

    map.value = L.map(mapContainer.value);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
    }).addTo(map.value);
  }
};

const updateMap = () => {
  if (!map.value) {
    return;
  }

  // Clear existing track layers before drawing new ones.
  map.value.eachLayer((layer) => {
    if (layer instanceof L.Polyline && map.value) {
      map.value.removeLayer(layer);
    }
  });

  const allLatLngs: L.LatLng[] = [];
  props.gpxCoordinates.forEach((coords: number[][]) => {
    if (!map.value) {
      return;
    }

    const latLngs = coords
      .filter((coord) => isValidCoordinate(coord))
      .map((coord) => L.latLng(coord[0], coord[1]));

    if (latLngs.length === 0) {
      return;
    }

    allLatLngs.push(...latLngs);
    L.polyline(latLngs, { color: "red" }).addTo(map.value);
  });

  if (allLatLngs.length === 0) {
    return;
  }

  const bounds = L.latLngBounds(allLatLngs);
  if (bounds.isValid()) {
    map.value.fitBounds(bounds);
  }
};

// Watch for changes in gpxCoordinates and update rendered tracks.
watch(
  () => props.gpxCoordinates,
  () => {
    updateMap();
  },
  { immediate: true }
);

onMounted(() => {
  initMap();
  updateMap();
});

onBeforeUnmount(() => {
  if (map.value) {
    map.value.remove();
    map.value = undefined;
  }
});
</script>

<template>
  <main class="map-shell">
    <div
      ref="mapContainer"
      class="map-canvas"
    />
  </main>
</template>

<style scoped>
.map-shell {
  border: 1px solid #d7e2ef;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 12px 24px rgba(24, 39, 75, 0.08);
}

.map-canvas {
  width: 100%;
  height: calc(100vh - 190px);
}

@media (max-width: 992px) {
  .map-canvas {
    height: calc(100vh - 250px);
  }
}
</style>

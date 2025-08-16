<script setup lang="ts">
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { watch, ref, onMounted } from "vue";

const props = defineProps<{
  gpxCoordinates: number[][][];
}>();

const map = ref<L.Map>();

const initMap = () => {
  const mapContainer = document.getElementById("map");
  if (mapContainer) {
    map.value = L.map(mapContainer);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
    }).addTo(map.value);
  }
};

const updateMap = () => {
  if (map.value) {
    // Clear existing layers
    map.value.eachLayer((layer) => {
      if (layer instanceof L.Polyline) {
        if (map.value) {
          map.value.removeLayer(layer);
        }
      }
    });

    // Create a polyline from the GPX coordinates and add it to the map
    const polylines = props.gpxCoordinates.map((coords: number[][]) => {
      if (map.value) {
        const latLngs = coords
          .filter((coord) => typeof coord[0] === "number" && typeof coord[1] === "number")
          .map((coord) => L.latLng(coord[0] as number, coord[1] as number));
        return L.polyline(latLngs, { color: "red" }).addTo(map.value);
      }
    });

    // Fit the map to the bounds of all polylines
    // const bounds = L.latLngBounds(polylines.flatMap((polyline) => polyline.getLatLngs()));
    const bounds = L.latLngBounds(
      polylines.flatMap((polyline) => polyline!.getLatLngs().flat() as L.LatLng[])
    );
    map.value.fitBounds(bounds);
  }
};

// Watch for changes in gpxCoordinates and initialize the map when they are loaded
watch(
  () => props.gpxCoordinates,
  (newVal) => {
    if (newVal && newVal.length > 0) {
      updateMap();
    }
  },
  { immediate: true }
);

onMounted(() => {
  initMap();
});
</script>

<template>
  <main>
    <div
      id="map"
      style="width: 100%; height: calc(100vh - 150px)"
    />
  </main>
</template>

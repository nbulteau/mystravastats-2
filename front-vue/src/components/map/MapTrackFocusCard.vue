<script setup lang="ts">
import type { MapTrack } from "@/models/map.model";

defineProps<{
  track: MapTrack;
}>();

defineEmits<{
  (event: "clear"): void;
}>();
</script>

<template>
  <div
    class="map-focus-card"
    role="status"
    aria-live="polite"
  >
    <span>Focused activity</span>
    <strong>{{ track.activityName }}</strong>
    <button
      type="button"
      aria-label="Clear focused activity"
      @click="$emit('clear')"
    >
      Clear
    </button>
  </div>
</template>

<style scoped>
.map-focus-card {
  position: absolute;
  top: 14px;
  left: 58px;
  z-index: 560;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  max-width: min(560px, calc(100% - 72px));
  padding: 7px 9px;
  border: 1px solid rgba(36, 87, 197, 0.35);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.94);
  box-shadow: 0 5px 16px rgba(15, 23, 42, 0.14);
  backdrop-filter: blur(8px);
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.map-focus-card strong {
  overflow: hidden;
  color: var(--ms-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.map-focus-card button {
  padding: 2px 7px;
  border: 1px solid var(--ms-border);
  border-radius: 7px;
  color: #2457c5;
  background: #ffffff;
  font-weight: 700;
}

.map-focus-card button:focus-visible {
  outline: 2px solid #2457c5;
  outline-offset: 2px;
}

@media (max-width: 992px) {
  .map-focus-card {
    top: 56px;
    left: 12px;
    max-width: calc(100% - 24px);
  }
}
</style>

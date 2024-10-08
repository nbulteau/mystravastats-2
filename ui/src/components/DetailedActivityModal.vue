<template>
  <div
    id="detailedActivityModal"
    class="modal fade"
    tabindex="-1"
    aria-labelledby="detailedActivityModalLabel"
    aria-hidden="true"
  >
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5
            id="detailedActivityModalLabel"
            class="modal-title"
          >
            Detailed Activity
          </h5>
          <button
            type="button"
            class="btn-close"
            data-bs-dismiss="modal"
            aria-label="Close"
          />
        </div>
        <div class="modal-body">
          <!-- Modal body content goes here -->
          <p><strong>Name:</strong> {{ activity.name }}</p>
          <p><strong>Distance:</strong> {{ activity.distance / 1000 }} km</p>
          <p><strong>Elapsed Time:</strong> {{ formattedElapsedTime(activity.elapsedTime) }}</p>
          <p>
            <strong>Total Elevation Gain:</strong> {{ activity.totalElevationGain }} m
          </p>
          <p>
            <strong>Average Speed:</strong>
            {{ formatSpeedWithUnit(activity.averageSpeed, activity.type) }}
          </p>
          <p><strong>Date:</strong> {{ formatDate(activity.date) }}</p>
        </div>
        <div class="modal-footer">
          <button
            type="button"
            class="btn btn-secondary"
            data-bs-dismiss="modal"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from "vue";
import type { Activity } from "@/models/activity.model";

const props = defineProps<{
  activity: Activity;
}>();

const options: Intl.DateTimeFormatOptions = {
  weekday: "short",
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
};

function formatDate(value: string): string {
  const date = new Date(value);
  return new Intl.DateTimeFormat(navigator.language, options).format(date);
}

function formatSpeedWithUnit(speed: number, activityType: string): string {
  if (activityType === "Run") {
    return `${formatSeconds(1000 / speed)}/km`;
  } else {
    return `${(speed * 3.6).toFixed(2)} km/h`;
  }
}

function formattedElapsedTime(value: number): string {
  const hours = Math.floor((value ?? 0) / 3600);
  const minutes = Math.floor(((value ?? 0) % 3600) / 60);
  const seconds = (value ?? 0) % 60;

  if (hours === 0) {
    return `${minutes}m ${seconds}s`; // Customize the formatting as needed
  }
  return `${hours}h ${minutes}m ${seconds}s`; // Customize the formatting as needed
}

/**
 * Format seconds to minutes and seconds
 */
function formatSeconds(seconds: number): string {
  let min = Math.floor((seconds % 3600) / 60);
  let sec = Math.floor(seconds % 60);
  const hnd = Math.floor((seconds - min * 60 - sec) * 100 + 0.5);

  if (hnd === 100) {
    sec++;
    if (sec === 60) {
      sec = 0;
      min++;
    }
  }

  return `${min}'${sec < 10 ? "0" : ""}${sec}`;
}
</script>

<style scoped>

</style>

<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import {computed, ref, watch} from "vue";

const contextStore = useContextStore();
const athleteDisplayName = computed(() => contextStore.athleteDisplayName);
const selectedYear = computed({
  get: () => contextStore.currentYear,
  set: (year: string) => contextStore.updateCurrentYear(year),
});
const selectedActivity = computed(() => contextStore.currentActivityType);

const cyclingActivities = ['Ride', 'Commute', 'GravelRide', 'MountainBikeRide', 'VirtualRide'];

const runningActivities = ['Run', 'TrailRun'];

const splitActivities = (v?: string) =>
    v && v.length > 0 ? v.split('_') as string[] : ['Ride'];

const selectedActivitiesType = ref<string[]>(splitActivities(selectedActivity.value));

watch(
    () => selectedActivity.value,
    (v) => {
      selectedActivitiesType.value = splitActivities(v);
    }
);

const currentYear = new Date().getFullYear();
const years: string[] = Array.from({ length: currentYear - 2010 + 1 }, (_, i) => (currentYear - i).toString());
years.push("All years");

const onChangeCurrentYear = (event: Event) => {
  const target = event.currentTarget as HTMLInputElement;
  const year = target.value;
  contextStore.updateCurrentYear(year);
};

import { onMounted, onBeforeUnmount } from "vue";
import Tooltip from "bootstrap/js/dist/tooltip";

let tooltipInstances: Tooltip[] = [];

onMounted(() => {
  const elements = document.querySelectorAll<HTMLElement>('[data-bs-toggle="tooltip"]');
  tooltipInstances = Array.from(elements).map(el => new Tooltip(el));
});

onBeforeUnmount(() => {
  tooltipInstances.forEach(t => t.dispose());
  tooltipInstances = [];
});


// Function to handle an activity type changes:
const onChangeActivityType = (activity: 'Ride' | 'VirtualRide' | 'GravelRide' | 'MountainBikeRide' | 'Commute' | 'Run' | 'TrailRun' | 'Hike' | 'AlpineSki') => {

  if (cyclingActivities.includes(activity)) {
    if (selectedActivitiesType.value.includes(activity)) {
      // Remove the activity if it's already selected
      selectedActivitiesType.value = selectedActivitiesType.value.filter(a => a !== activity);
      // If no cycling activities left, default to 'Ride'
      if (!selectedActivitiesType.value.some(a => cyclingActivities.includes(a))) {
        selectedActivitiesType.value = ['Ride'];
      }
    } else {
      // Add the activity to the selection
      selectedActivitiesType.value = [
        ...selectedActivitiesType.value.filter(a => cyclingActivities.includes(a)), // Keep only cycling activities
        activity
      ];
    }
  } else {
    // For non-cycling activities, select that single activity
    selectedActivitiesType.value = [activity];
  }

  // Update the activity type in the store
  let activityType;
  const selectedCyclingActivities = selectedActivitiesType.value
      .filter(a => cyclingActivities.includes(a))
      .sort(); // Sort to ensure consistent ordering

  if (selectedCyclingActivities.length > 1) {
    // Create a combination name based on selected activities
    activityType = selectedCyclingActivities.join('_');
  } else {
    activityType = selectedActivitiesType.value[0];
  }

  contextStore.updateCurrentActivityType(activityType ?? "");
};

</script>

<template>
  <nav
    class="navbar"
    style="background-color: #e3f2fd"
  >
    <div class="container">
      <span class="athlete-name">{{ athleteDisplayName }}</span>

      <div class="d-flex align-items-center">
        <select
          id="year"
          v-model="selectedYear"
          name="year"
          class="form-select-lg me-3"
          @change="onChangeCurrentYear"
        >
          <option
            v-for="year in years"
            :key="year"
            :value="year"
          >
            {{ year }}
          </option>
        </select>

        <div
          class="btn-group btn-group-lg"
          role="group"
          aria-label="Activity"
        >
          <button
            id="ride"
            type="button"
            class="btn icon-btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Ride'),
              'btn-primary': selectedActivitiesType.includes('Ride'),
            }"
            data-bs-toggle="tooltip"
            data-bs-placement="bottom"
            title="Ride"
            aria-label="Ride"
            @click="onChangeActivityType('Ride')"
          >
            <img
              src="@/assets/buttons/road-bike.png"
              alt="Ride"
            >
          </button>

          <button
              id="mountain-bike-ride"
              type="button"
              class="btn icon-btn"
              :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('MountainBikeRide'),
              'btn-primary': selectedActivitiesType.includes('MountainBikeRide'),
            }"
              data-bs-toggle="tooltip"
              data-bs-placement="bottom"
              title="Mountain bike"
              aria-label="mountain bike"
              @click="onChangeActivityType('MountainBikeRide')"
          >
            <img
                src="@/assets/buttons/mountain-bike.png"
                alt="MountainBikeRide"
            >
          </button>

          <button
            id="commute"
            type="button"
            class="btn icon-btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Commute'),
              'btn-primary': selectedActivitiesType.includes('Commute'),
            }"
            data-bs-toggle="tooltip"
            data-bs-placement="bottom"
            title="Commute"
            aria-label="Commute"
            @click="onChangeActivityType('Commute')"
          >
            <img
              src="@/assets/buttons/city-bike.png"
              alt="Commute"
            >
          </button>
          <button
            id="gravel-ride"
            type="button"
            class="btn icon-btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('GravelRide'),
              'btn-primary': selectedActivitiesType.includes('GravelRide'),
            }"
            data-bs-toggle="tooltip"
            data-bs-placement="bottom"
            title="Gravel"
            aria-label="Gravel"
            @click="onChangeActivityType('GravelRide')"
          >
            <img
              src="@/assets/buttons/touring-bike.png"
              alt="Gravel Ride"
            >
          </button>
          <button
              id="virtual-ride"
              type="button"
              class="btn icon-btn"
              :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('VirtualRide'),
              'btn-primary': selectedActivitiesType.includes('VirtualRide'),
            }"
              data-bs-toggle="tooltip"
              data-bs-placement="bottom"
              title="Virtual ride"
              aria-label="Virtual ride"
              @click="onChangeActivityType('VirtualRide')"
          >
            <img
                src="@/assets/buttons/virtual-bike.png"
                alt="Virtual Ride"
            >
          </button>
        </div>

        <div
          class="btn-group btn-group-lg"
          role="group"
          aria-label="Activity"
        >
          <button
            id="run"
            type="button"
            class="btn icon-btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Run'),
              'btn-primary': selectedActivitiesType.includes('Run'),
            }"
            data-bs-toggle="tooltip"
            data-bs-placement="bottom"
            title="Run"
            aria-label="Run"
            @click="onChangeActivityType('Run')"
          >
            <img
              src="@/assets/buttons/run.png"
              alt="Run"
            >
          </button>

          <button
            id="hike"
            type="button"
            class="btn icon-btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Hike'),
              'btn-primary': selectedActivitiesType.includes('Hike'),
            }"
            data-bs-toggle="tooltip"
            data-bs-placement="bottom"
            title="Hike"
            aria-label="Hike"
            @click="onChangeActivityType('Hike')"
          >
            <img
              src="@/assets/buttons/hike.png"
              alt="Hike"
            >
          </button>

          <button
            id="alpine-ski"
            type="button"
            class="btn icon-btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('AlpineSki'),
              'btn-primary': selectedActivitiesType.includes('AlpineSki'),
            }"
            data-bs-toggle="tooltip"
            data-bs-placement="bottom"
            title="Alpine ski"
            aria-label="Alpine ski"
            @click="onChangeActivityType('AlpineSki')"
          >
            <img
              src="@/assets/buttons/alpineski.png"
              alt="Alpine Ski"
            >
          </button>
        </div>
      </div>
    </div>
  </nav>
</template>

<style scoped>
header {
  display: compact;
}

.athlete-name {
  margin-right: 20px;
}

.icon-btn {
  width: 48px;
  height: 48px;
  padding: 5px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.icon-btn img {
  width: 100%;
  height: 100%;
}

</style>

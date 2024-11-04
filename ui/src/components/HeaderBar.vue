<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
const athleteDisplayName = computed(() => contextStore.athleteDisplayName);
const selectedYear = contextStore.currentYear;
const selectedActivity = computed(() => contextStore.currentActivity);

const currentYear = new Date().getFullYear();
const years: string[] = Array.from({ length: currentYear - 2010 + 1 }, (_, i) => (2024 - i).toString());
years.push("All years");

const onChangeCurrentYear = (event: Event) => {
  const target = event.currentTarget as HTMLInputElement;
  const year = target.value;
  contextStore.updateCurrentYear(year);
};

const onChangeActivityType = (event: Event) => {
  const target = event.currentTarget as HTMLElement;
  switch (target.id) {
    case "ride":
      contextStore.updateCurrentActivityType("Ride");
      break;
    case "virtual-ride":
      contextStore.updateCurrentActivityType("VirtualRide");
      break;
    case "commute":
      contextStore.updateCurrentActivityType("Commute");
      break;
    case "run":
      contextStore.updateCurrentActivityType("Run");
      break;
    case "hike":
      contextStore.updateCurrentActivityType("Hike");
      break;
    case "alpine-ski":
      contextStore.updateCurrentActivityType("AlpineSki")
  }
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
            class="btn"
            :class="{
              'btn-outline-primary': selectedActivity !== 'Ride',
              'btn-primary': selectedActivity === 'Ride',
            }"
            @click="onChangeActivityType"
          >
            <img
              src="@/assets/buttons/ride.png"
              alt="Ride"
            >
          </button>

          <button
            id="virtual-ride"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': selectedActivity !== 'VirtualRide',
              'btn-primary': selectedActivity === 'VirtualRide',
            }"
            @click="onChangeActivityType"
          >
            <img
              src="@/assets/buttons/virtualride.png"
              alt="Virtual ride"
            >
          </button>

          <button
            id="commute"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': selectedActivity !== 'Commute',
              'btn-primary': selectedActivity === 'Commute',
            }"
            @click="onChangeActivityType"
          >
            <img
              src="@/assets/buttons/commute.png"
              alt="Commute"
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
            class="btn"
            :class="{
              'btn-outline-primary': selectedActivity !== 'Run',
              'btn-primary': selectedActivity === 'Run',
            }"
            @click="onChangeActivityType"
          >
            <img
              src="@/assets/buttons/run.png"
              alt="Run"
            >
          </button>

          <button
            id="hike"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': selectedActivity !== 'Hike',
              'btn-primary': selectedActivity === 'Hike',
            }"
            @click="onChangeActivityType"
          >
            <img
              src="@/assets/buttons/hike.png"
              alt="Hike"
            >
          </button>
          <button
            id="alpine-ski"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': selectedActivity !== 'AlpineSki',
              'btn-primary': selectedActivity === 'AlpineSki',
            }"
            @click="onChangeActivityType"
          >
            <img
              src="@/assets/buttons/alpineski.png"
              alt="AlpineSki"
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
  margin-right: 20px; /* Adjust the value as needed */
}
</style>

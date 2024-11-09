<script setup lang="ts">
import { useContextStore } from "@/stores/context.js";
import { computed } from "vue";

const contextStore = useContextStore();
const athleteDisplayName = computed(() => contextStore.athleteDisplayName);
const selectedYear = contextStore.currentYear;
const selectedActivity = computed(() => contextStore.currentActivityType);
import { ref } from "vue";

const selectedActivitiesType = ref<string[]>([selectedActivity.value as string]);

const currentYear = new Date().getFullYear();
const years: string[] = Array.from({ length: currentYear - 2010 + 1 }, (_, i) => (2024 - i).toString());
years.push("All years");

const onChangeCurrentYear = (event: Event) => {
  const target = event.currentTarget as HTMLInputElement;
  const year = target.value;
  contextStore.updateCurrentYear(year);
};

const onChangeActivityType = (activity: 'Ride' | 'VirtualRide' | 'Commute' | 'Run' | 'Hike' | 'AlpineSki') => {
  if (activity === 'Ride' || activity === 'Commute') {
    const otherActivity = activity === 'Ride' ? 'Commute' : 'Ride';
    if (selectedActivitiesType.value.includes(activity)) {
      selectedActivitiesType.value = selectedActivitiesType.value.length === 1 ? [activity] : [otherActivity];
    } else {
      selectedActivitiesType.value = selectedActivitiesType.value.includes(otherActivity) ? ['Ride', 'Commute'] : [activity];
    }
  } else {
    selectedActivitiesType.value = [activity];
  }

  const activityType = selectedActivitiesType.value.length > 1 ? 'RideWithCommute' : selectedActivitiesType.value[0];
  contextStore.updateCurrentActivityType(activityType);
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
              'btn-outline-primary': !selectedActivitiesType.includes('Ride'),
              'btn-primary': selectedActivitiesType.includes('Ride'),
            }"
            @click="onChangeActivityType('Ride')"
          >
            <img
              src="@/assets/buttons/ride.png"
              alt="Ride"
            >
          </button>

          <button
            id="commute"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Commute'),
              'btn-primary': selectedActivitiesType.includes('Commute'),
            }"
            @click="onChangeActivityType('Commute')"
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
            id="virtual-ride"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('VirtualRide'),
              'btn-primary': selectedActivitiesType.includes('VirtualRide'),
            }"
            @click="onChangeActivityType('VirtualRide')"
          >
            <img
              src="@/assets/buttons/virtualride.png"
              alt="Virtual Ride"
            >
          </button>

          <button
            id="run"
            type="button"
            class="btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Run'),
              'btn-primary': selectedActivitiesType.includes('Run'),
            }"
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
            class="btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('Hike'),
              'btn-primary': selectedActivitiesType.includes('Hike'),
            }"
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
            class="btn"
            :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('AlpineSki'),
              'btn-primary': selectedActivitiesType.includes('AlpineSki'),
            }"
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
  /* Adjust the value as needed */
}
</style>

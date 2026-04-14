<script setup lang="ts">
import {useContextStore} from "@/stores/context.js";
import { useAthleteStore } from "@/stores/athlete";
import {computed, onBeforeUnmount, onMounted, ref, watch} from "vue";
import Tooltip from "bootstrap/js/dist/tooltip";

const contextStore = useContextStore();
const athleteStore = useAthleteStore();
const athleteDisplayName = computed(() => athleteStore.athleteDisplayName);
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
const years: string[] = Array.from({length: currentYear - 2010 + 1}, (_, i) => (currentYear - i).toString());
years.push("All years");

let tooltipInstances: Tooltip[] = [];

onMounted(() => {
  const elements = document.querySelectorAll<HTMLElement>('[data-bs-toggle="tooltip"]');
  tooltipInstances = Array.from(elements).map(el => new Tooltip(el));
});

onBeforeUnmount(() => {
  tooltipInstances.forEach(t => t.dispose());
  tooltipInstances = [];
});

const toggleActivity = (activity: string, activities: string[], defaultActivity: string) => {
  if (selectedActivitiesType.value.includes(activity)) {
    selectedActivitiesType.value = selectedActivitiesType.value.filter(a => a !== activity);
    if (!selectedActivitiesType.value.some(a => activities.includes(a))) {
      selectedActivitiesType.value = [defaultActivity];
    }
  } else {
    selectedActivitiesType.value = [
      ...selectedActivitiesType.value.filter(a => activities.includes(a)),
      activity,
    ];
  }
};

// Function to handle an activity type changes:
const onChangeActivityType = (activity: 'Ride' | 'VirtualRide' | 'GravelRide' | 'MountainBikeRide' | 'Commute' | 'Run' | 'TrailRun' | 'Hike' | 'AlpineSki') => {

  if (cyclingActivities.includes(activity)) {
    toggleActivity(activity, cyclingActivities, 'Ride');
  } else if (runningActivities.includes(activity)) {
    toggleActivity(activity, runningActivities, 'Run');
  } else {
    // For non-cycling activities, select that single activity
    selectedActivitiesType.value = [activity];
  }

  // Update the activity type in the store
  let activityType;
  const selectedCyclingActivities = selectedActivitiesType.value
      .filter(a => cyclingActivities.includes(a))
      .sort(); // Sort to ensure consistent ordering

  const selectedRunningActivities = selectedActivitiesType.value
      .filter(a => runningActivities.includes(a))
      .sort(); // Sort to ensure consistent ordering

  if (selectedCyclingActivities.length > 1) {
    // Create a combination name based on selected activities
    activityType = selectedCyclingActivities.join('_');
  } else if (selectedRunningActivities.length > 1) {
    // Create a combination name based on selected activities
    activityType = selectedRunningActivities.join('_');
  } else  {
    activityType = selectedActivitiesType.value[0];
  }

  contextStore.updateCurrentActivityType(activityType ?? "");
};

</script>

<template>
  <nav
      class="navbar top-navbar"
  >
    <div class="container top-navbar__content">
      <span class="athlete-name">{{ athleteDisplayName }}</span>

      <div class="filters-wrap">
        <select
            id="year"
            v-model="selectedYear"
            name="year"
            class="form-select year-select"
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
            class="btn-group btn-group-lg activity-group"
            role="group"
            aria-label="RideActivity"
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
            class="btn-group btn-group-lg activity-group"
            role="group"
            aria-label="RunActivity"
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
              id="trail-run"
              type="button"
              class="btn icon-btn"
              :class="{
              'btn-outline-primary': !selectedActivitiesType.includes('TrailRun'),
              'btn-primary': selectedActivitiesType.includes('TrailRun'),
            }"
              data-bs-toggle="tooltip"
              data-bs-placement="bottom"
              title="Trail run"
              aria-label="Trail rin"
              @click="onChangeActivityType('TrailRun')"
          >
            <img
                src="@/assets/buttons/trail-run.png"
                alt="trail run"
            >
          </button>
        </div>

        <div
            class="btn-group btn-group-lg activity-group"
            role="group"
            aria-label="Activity"
        >
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
                src="../assets/buttons/alpine-ski.png"
                alt="Alpine Ski"
            >
          </button>
        </div>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.top-navbar {
  background: #ffffff;
  border-bottom: 1px solid var(--ms-border);
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.05);
}

.top-navbar__content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 7px;
  padding-bottom: 7px;
}

.athlete-name {
  margin-right: 14px;
  color: #2a2d33;
  font-weight: 800;
  font-size: 0.98rem;
  letter-spacing: 0.01em;
}

.filters-wrap {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex: 1;
  flex-wrap: wrap;
  gap: 8px;
}

.year-select {
  min-width: 125px;
  max-width: 140px;
  border-radius: 10px;
  border: 1px solid var(--ms-border);
  background: #ffffff;
  color: #2d3036;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.06);
  font-size: 0.95rem;
  font-weight: 700;
}

.activity-group {
  border-radius: 10px;
  padding: 2px;
  background: #f7f8fb;
  border: 1px solid #e2e4ea;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
}

.icon-btn {
  width: 39px;
  height: 39px;
  padding: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  border: 1px solid transparent;
  line-height: 1;
  transition: all 0.15s ease;
}

.icon-btn.btn-outline-primary {
  background: #ffffff;
  border-color: transparent;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.04);
}

.icon-btn.btn-outline-primary:hover {
  background: #fff2ea;
  border-color: #ffd3c1;
  transform: translateY(-1px);
}

.icon-btn.btn-primary {
  background: #fff1e8;
  border-color: #fdc1a6;
  box-shadow: 0 3px 10px rgba(252, 76, 2, 0.2);
}

.icon-btn img {
  width: 90%;
  height: 90%;
}

@media (max-width: 992px) {
  .top-navbar__content {
    align-items: flex-start;
  }

  .filters-wrap {
    justify-content: flex-start;
  }

  .year-select {
    min-width: 115px;
    max-width: 125px;
  }

  .activity-group {
    width: 100%;
    justify-content: center;
  }
}
</style>

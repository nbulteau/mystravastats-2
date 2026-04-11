<script setup lang="ts">
import {useContextStore} from "@/stores/context.js";
import {computed, onBeforeUnmount, onMounted, ref, watch} from "vue";
import Tooltip from "bootstrap/js/dist/tooltip";

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
  background: linear-gradient(120deg, rgba(237, 246, 255, 0.98), rgba(226, 241, 255, 0.95));
  border-bottom: 1px solid #d2deec;
  box-shadow: 0 8px 22px rgba(24, 39, 75, 0.09);
}

.top-navbar__content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  padding-top: 6px;
  padding-bottom: 6px;
}

.athlete-name {
  margin-right: 12px;
  color: #213247;
  font-weight: 700;
  letter-spacing: 0.2px;
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
  border-radius: 12px;
  border: 1px solid #b5c8df;
  background: #ffffff;
  color: #213247;
  box-shadow: 0 3px 10px rgba(24, 39, 75, 0.08);
  font-size: 0.95rem;
  font-weight: 600;
}

.activity-group {
  border-radius: 14px;
  padding: 3px;
  background: rgba(255, 255, 255, 0.65);
  border: 1px solid #cdd8e8;
  box-shadow: 0 6px 14px rgba(24, 39, 75, 0.08);
}

.icon-btn {
  width: 42px;
  height: 42px;
  padding: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  border: 1px solid transparent;
  line-height: 1;
  transition: all 0.2s ease;
}

.icon-btn.btn-outline-primary {
  background: #ffffff;
  border-color: #c4d4e6;
}

.icon-btn.btn-outline-primary:hover {
  background: #ebf4ff;
  border-color: #9eb8d8;
  transform: translateY(-1px);
}

.icon-btn.btn-primary {
  background: linear-gradient(135deg, #0f67c6, #0ea5e9);
  border-color: #0f67c6;
  box-shadow: 0 6px 14px rgba(15, 103, 198, 0.32);
}

.icon-btn img {
  width: 90%;
  height: 90%;
}

@media (max-width: 992px) {
  .filters-wrap {
    justify-content: flex-start;
  }

  .activity-group {
    width: 100%;
    justify-content: center;
  }
}
</style>

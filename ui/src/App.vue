<script setup lang="ts">
import "../node_modules/bootstrap/scss/bootstrap.scss";
import HeaderBar from "@/components/HeaderBar.vue";
import { onMounted } from "vue";
import { RouterLink } from "vue-router";
import { useContextStore } from "@/stores/context.js";
const contextStore = useContextStore();

const isCurrent = (name: string) => {
  return contextStore.currentView === name;
};

onMounted(async () => {
  await contextStore.updateData();
});

</script>

<template>
  <div v-if="contextStore.currentView != 'activity'">
    <HeaderBar class="fixed-top" />
    <nav class="navbar container mt-5">
      <ul
        id="myTab"
        class="nav nav-tabs"
        role="tablist"
      >
        <li
          class="nav-item"
          role="presentation"
        >
          <button
            id="statistics-tab"
            class="nav-link"
            :class="{ active: isCurrent('statistics') }"
            data-bs-toggle="tab"
            data-bs-target="#statistics-tab-pane"
            type="button"
            role="tab"
            aria-controls="home-tab-pane"
            aria-selected="true"
            href="/statistics"
          >
            <RouterLink to="/">
              Statistics
            </RouterLink>
          </button>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <button
            id="activities-tab"
            class="nav-link"
            :class="{ active: isCurrent('activities') }"
            data-bs-toggle="tab"
            data-bs-target="#activities-tab-pane"
            type="button"
            role="tab"
            aria-controls="activities-tab-pane"
            aria-selected="false"
            href="/activities"
          >
            <RouterLink to="/activities">
              Activities
            </RouterLink>
          </button>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <button
            id="map-tab"
            class="nav-link"
            :class="{ active: isCurrent('map') }"
            data-bs-toggle="tab"
            data-bs-target="#map-tab-pane"
            type="button"
            role="tab"
            aria-controls="map-tab-pane"
            aria-selected="false"
            href="/map"
          >
            <RouterLink to="/map">
              Map
            </RouterLink>
          </button>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <button
            id="charts-tab"
            class="nav-link"
            :class="{ active: isCurrent('charts') }"
            data-bs-toggle="tab"
            data-bs-target="#charts-tab-pane"
            type="button"
            role="tab"
            aria-controls="chart-tab-pane"
            aria-selected="false"
            href="/charts"
          >
            <RouterLink to="/charts">
              Charts
            </RouterLink>
          </button>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <button
            id="dashboard-tab"
            class="nav-link"
            :class="{ active: isCurrent('dashboard') }"
            data-bs-toggle="tab"
            data-bs-target="#dashboard-tab-pane"
            type="button"
            role="tab"
            aria-controls="dashboard-tab-pane"
            aria-selected="false"
            href="/dashboard"
          >
            <RouterLink to="/dashboard">
              Dashboard
            </RouterLink>
          </button>
        </li>
        <div v-if="contextStore.hasBadges">

        <li
          class="nav-item"
          role="presentation"
        >
          <button
            id="badges-tab"
            class="nav-link"
            :class="{ active: isCurrent('badges') }"
            data-bs-toggle="tab"
            data-bs-target="#badges-tab-pane"
            type="button"
            role="tab"
            aria-controls="badges-tab-pane"
            aria-selected="false"
            href="/badges"
          >
            <RouterLink to="/badges">
              Badges
            </RouterLink>
          </button>
        </li>
        </div>
      </ul>
    </nav>
  </div>
  <div class="container">
    <main>
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.fixed-top {
  position: fixed;
  top: 0;
  width: 100%;
  z-index: 1030; /* Ensure it is above other elements */
}

.mt-5 {
  margin-top: 5rem !important; /* Adjust this value if needed */
}
</style>

<script setup lang="ts">
import "../node_modules/bootstrap/scss/bootstrap.scss";
import HeaderBar from "@/components/HeaderBar.vue";
import { RouterLink } from "vue-router";
import { useContextStore } from "@/stores/context.js";

const contextStore = useContextStore();

const isCurrent = (name: string) => {
  return contextStore.currentView === name;
};
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
          <RouterLink
            id="statistics-tab"
            class="nav-link"
            :class="{ active: isCurrent('statistics') }"
            role="tab"
            aria-controls="home-tab-pane"
            aria-selected="true"
            to="/"
          >
            Statistics
          </RouterLink>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <RouterLink
            id="activities-tab"
            class="nav-link"
            :class="{ active: isCurrent('activities') }"
            role="tab"
            aria-controls="activities-tab-pane"
            aria-selected="false"
            to="/activities"
          >
            Activities
          </RouterLink>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <RouterLink
            id="map-tab"
            class="nav-link"
            :class="{ active: isCurrent('map') }"
            role="tab"
            aria-controls="map-tab-pane"
            aria-selected="false"
            to="/map"
          >
            Map
          </RouterLink>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <RouterLink
            id="charts-tab"
            class="nav-link"
            :class="{ active: isCurrent('charts') }"
            role="tab"
            aria-controls="chart-tab-pane"
            aria-selected="false"
            to="/charts"
          >
            Charts
          </RouterLink>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <RouterLink
            id="dashboard-tab"
            class="nav-link"
            :class="{ active: isCurrent('dashboard') }"
            role="tab"
            aria-controls="dashboard-tab-pane"
            aria-selected="false"
            to="/dashboard"
          >
            Dashboard
          </RouterLink>
        </li>
        <li
          class="nav-item"
          role="presentation"
        >
          <RouterLink
            id="badges-tab"
            class="nav-link"
            :class="{ active: isCurrent('badges') }"
            role="tab"
            aria-controls="badges-tab-pane"
            aria-selected="false"
            to="/badges"
          >
            Badges
          </RouterLink>
        </li>
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

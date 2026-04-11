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
  <div class="app-frame">
    <div v-if="contextStore.currentView !== 'activity'">
      <HeaderBar class="fixed-top app-header" />
      <nav class="navbar container app-tabs-shell">
      <ul
        id="myTab"
          class="nav nav-tabs app-tabs"
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
            id="heatmap-tab"
            class="nav-link"
            :class="{ active: isCurrent('heatmap') }"
            role="tab"
            aria-controls="heatmap-tab-pane"
            aria-selected="false"
            to="/heatmap"
          >
            Heatmap
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

    <div
      class="container app-main"
      :class="{ 'app-main--activity': contextStore.currentView === 'activity' }"
    >
      <main
        class="app-content"
        :class="{ 'app-content--activity': contextStore.currentView === 'activity' }"
      >
        <RouterView />
      </main>
    </div>

    <div
      class="toast-stack"
      aria-live="polite"
      aria-atomic="true"
    >
      <div
        v-for="toast in contextStore.toasts"
        :key="toast.id"
        :class="[
          'app-toast',
          `app-toast--${String(toast.type ?? 'normal').toLowerCase()}`,
        ]"
      >
        <span>{{ toast.message }}</span>
        <button
          type="button"
          class="app-toast-close"
          aria-label="Close notification"
          @click="contextStore.removeToast(toast)"
        >
          ×
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app-frame {
  min-height: 100vh;
}

.fixed-top {
  position: fixed;
  top: 0;
  width: 100%;
  z-index: 1030;
}

.app-tabs-shell {
  margin-top: 82px;
  padding-left: 0;
  padding-right: 0;
}

.app-tabs {
  width: 100%;
  gap: 6px;
  border: 1px solid #d7e2ef;
  border-radius: 16px;
  padding: 6px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 0 12px 24px rgba(24, 39, 75, 0.08);
  backdrop-filter: blur(10px);
}

.app-tabs .nav-item {
  flex: 1 1 auto;
}

.app-tabs .nav-link {
  border: 0;
  border-radius: 12px;
  color: #2f3c50;
  font-weight: 600;
  text-align: center;
  transition: all 0.2s ease;
}

.app-tabs .nav-link:hover {
  background: #edf5ff;
  color: #0b4f9f;
}

.app-tabs .nav-link.active {
  color: #ffffff;
  background: linear-gradient(135deg, #0f67c6, #0ea5e9);
  box-shadow: 0 8px 18px rgba(15, 103, 198, 0.35);
}

.app-main {
  padding-top: 14px;
  padding-bottom: 20px;
}

.app-main--activity {
  padding-top: 8px;
}

.app-content {
  border: 1px solid #d7e2ef;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 0 18px 36px rgba(24, 39, 75, 0.09);
  backdrop-filter: blur(8px);
  min-height: calc(100vh - 172px);
  padding: 14px;
}

.app-content--activity {
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
  min-height: auto;
  padding: 0;
}

.toast-stack {
  position: fixed;
  right: 16px;
  bottom: 16px;
  z-index: 1100;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 420px;
}

.app-toast {
  border-radius: 12px;
  border: 1px solid #f1c6cf;
  background: #fff5f6;
  color: #772a3a;
  box-shadow: 0 10px 26px rgba(119, 42, 58, 0.14);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.app-toast--warn {
  border-color: #f6d89b;
  background: #fff9ec;
  color: #7f5a00;
}

.app-toast--normal {
  border-color: #cce1ff;
  background: #f0f7ff;
  color: #184f86;
}

.app-toast-close {
  border: 0;
  background: transparent;
  color: currentColor;
  font-size: 1.1rem;
  line-height: 1;
  cursor: pointer;
}

@media (max-width: 992px) {
  .app-tabs-shell {
    margin-top: 130px;
  }

  .app-tabs .nav-item {
    flex: 1 1 calc(33.333% - 6px);
  }

  .app-content {
    min-height: calc(100vh - 220px);
    padding: 10px;
  }
}
</style>

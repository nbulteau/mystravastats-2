<script setup lang="ts">
import "../node_modules/bootstrap/scss/bootstrap.scss";
import HeaderBar from "@/components/HeaderBar.vue";
import { RouterLink } from "vue-router";
import { useContextStore } from "@/stores/context.js";
import { useUiStore } from "@/stores/ui";

const contextStore = useContextStore();
const uiStore = useUiStore();

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
        v-for="toast in uiStore.toasts"
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
          @click="uiStore.removeToast(toast)"
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
  margin-top: 74px;
  padding: 0;
  border-bottom: 1px solid var(--ms-border);
  background: #ffffff;
  box-shadow: 0 2px 10px rgba(15, 23, 42, 0.04);
}

.app-tabs {
  width: 100%;
  display: flex;
  flex-wrap: nowrap;
  overflow-x: auto;
  gap: 2px;
  border: 0;
  padding: 0 2px;
  background: transparent;
  scrollbar-width: thin;
}

.app-tabs .nav-item {
  flex: 0 0 auto;
}

.app-tabs .nav-link {
  border: 0;
  border-radius: 0;
  border-bottom: 3px solid transparent;
  color: #6d7079;
  font-weight: 700;
  letter-spacing: 0.01em;
  text-align: center;
  padding: 0.85rem 0.9rem 0.75rem;
  transition: color 0.2s ease, border-color 0.2s ease, background 0.2s ease;
  white-space: nowrap;
}

.app-tabs .nav-link:hover {
  background: #fff7f2;
  color: #3a3d46;
}

.app-tabs .nav-link.active {
  color: var(--ms-primary);
  background: #fff8f4;
  border-bottom-color: var(--ms-primary);
}

.app-main {
  padding-top: 16px;
  padding-bottom: 22px;
}

.app-main--activity {
  padding-top: 8px;
}

.app-content {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  min-height: calc(100vh - 152px);
  padding: 0;
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
  right: 18px;
  bottom: 18px;
  z-index: 1100;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 420px;
}

.app-toast {
  border-radius: 10px;
  border: 1px solid #ffd3c2;
  border-left: 4px solid var(--ms-primary);
  background: #ffffff;
  color: #503126;
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.12);
  padding: 10px 11px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.app-toast--warn {
  border-color: #f7df9d;
  border-left-color: #f1b428;
  color: #6e5314;
}

.app-toast--normal {
  border-color: #ffd7c9;
  border-left-color: var(--ms-primary);
  color: #5a392a;
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
    margin-top: 124px;
  }

  .app-content {
    min-height: calc(100vh - 206px);
  }

  .toast-stack {
    right: 10px;
    left: 10px;
    max-width: none;
  }
}
</style>

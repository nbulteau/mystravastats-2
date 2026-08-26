<script setup lang="ts">
import "bootstrap/scss/bootstrap.scss";
import HeaderBar from "@/components/HeaderBar.vue";
import { RouterLink, useRoute } from "vue-router";
import { useUiStore } from "@/stores/ui";
import { useContextStore } from "@/stores/context.js";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

const contextStore = useContextStore();
const uiStore = useUiStore();
const route = useRoute();

type NavItem = {
  name: string;
  to: string;
  label: string;
  icon: string;
  beta?: boolean;
};

type NavGroup = {
  id: string;
  label: string;
  icon: string;
  items: readonly NavItem[];
};

const navGroups: readonly NavGroup[] = [
  {
    id: "activity",
    label: "Activité",
    icon: "fa-solid fa-person-running",
    items: [
      { name: "dashboard", to: "/dashboard", label: "Dashboard", icon: "fa-solid fa-chart-line" },
      { name: "activities", to: "/activities", label: "Activities", icon: "fa-solid fa-list" },
      { name: "map", to: "/map", label: "Map", icon: "fa-solid fa-map-location-dot" },
    ],
  },
  {
    id: "progress",
    label: "Progression",
    icon: "fa-solid fa-arrow-trend-up",
    items: [
      { name: "statistics", to: "/statistics", label: "Statistics", icon: "fa-solid fa-ranking-star" },
      { name: "charts", to: "/charts", label: "Trends", icon: "fa-solid fa-chart-area" },
      { name: "heatmap", to: "/heatmap", label: "Heatmap", icon: "fa-solid fa-calendar-days" },
      { name: "segments", to: "/segments", label: "Segments", icon: "fa-solid fa-mountain" },
      { name: "badges", to: "/badges", label: "Badges", icon: "fa-solid fa-medal" },
    ],
  },
  {
    id: "tools",
    label: "Outils",
    icon: "fa-solid fa-screwdriver-wrench",
    items: [
      { name: "gear", to: "/gear", label: "Equipment", icon: "fa-solid fa-bicycle" },
      { name: "routes", to: "/routes", label: "GPS Art", icon: "fa-solid fa-route", beta: true },
    ],
  },
];

const secondaryItems: readonly NavItem[] = [
  { name: "settings", to: "/settings", label: "Settings", icon: "fa-solid fa-sliders" },
  { name: "diagnostics", to: "/diagnostics", label: "Status", icon: "fa-solid fa-heart-pulse" },
];

const currentRouteName = computed(() => typeof route.name === "string" ? route.name : "");
const openMenu = ref<string | null>(null);
const navigationElement = ref<HTMLElement | null>(null);

const isCurrent = (name: string) => currentRouteName.value === name;
const isGroupCurrent = (group: NavGroup) => group.items.some((item) => isCurrent(item.name))
  || (group.id === "activity" && isCurrent("annual-recap"));
const isSecondaryCurrent = computed(() => secondaryItems.some((item) => isCurrent(item.name)));

function toggleMenu(menuId: string) {
  openMenu.value = openMenu.value === menuId ? null : menuId;
}

function closeMenus() {
  openMenu.value = null;
}

function handleDocumentPointerDown(event: PointerEvent) {
  if (navigationElement.value && !navigationElement.value.contains(event.target as Node)) {
    closeMenus();
  }
}

function handleDocumentKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") closeMenus();
}

onMounted(() => {
  document.addEventListener("pointerdown", handleDocumentPointerDown);
  document.addEventListener("keydown", handleDocumentKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", handleDocumentPointerDown);
  document.removeEventListener("keydown", handleDocumentKeydown);
});

watch(() => route.fullPath, closeMenus);
</script>

<template>
  <div class="app-frame">
    <div v-if="contextStore.currentView !== 'activity'">
      <HeaderBar class="fixed-top app-header" />
      <nav
        ref="navigationElement"
        class="container app-tabs-shell"
        aria-label="Main navigation"
      >
        <div class="nav-groups">
          <div
            v-for="group in navGroups"
            :key="group.id"
            class="nav-group"
            :class="{ 'nav-group--active': isGroupCurrent(group) }"
          >
            <button
              type="button"
              class="nav-group-trigger"
              :aria-expanded="openMenu === group.id"
              :aria-controls="`${group.id}-menu`"
              @click="toggleMenu(group.id)"
            >
              <i :class="group.icon" aria-hidden="true" />
              <span>{{ group.label }}</span>
              <i class="fa-solid fa-chevron-down nav-group-chevron" aria-hidden="true" />
            </button>
            <div
              v-show="openMenu === group.id"
              :id="`${group.id}-menu`"
              class="nav-menu"
            >
              <RouterLink
                v-for="item in group.items"
                :key="item.name"
                class="nav-menu-item"
                :class="{ active: isCurrent(item.name) }"
                :aria-current="isCurrent(item.name) ? 'page' : undefined"
                :to="item.to"
              >
                <i :class="item.icon" aria-hidden="true" />
                <span>{{ item.label }}</span>
                <span v-if="item.beta" class="tab-beta">beta</span>
              </RouterLink>
            </div>
          </div>

          <div
            class="nav-group nav-group--secondary"
            :class="{ 'nav-group--active': isSecondaryCurrent }"
          >
            <button
              type="button"
              class="nav-group-trigger nav-group-trigger--secondary"
              aria-label="Settings and status"
              :aria-expanded="openMenu === 'secondary'"
              aria-controls="secondary-menu"
              @click="toggleMenu('secondary')"
            >
              <i class="fa-solid fa-gear" aria-hidden="true" />
            </button>
            <div
              v-show="openMenu === 'secondary'"
              id="secondary-menu"
              class="nav-menu nav-menu--secondary"
            >
              <RouterLink
                v-for="item in secondaryItems"
                :key="item.name"
                class="nav-menu-item"
                :class="{ active: isCurrent(item.name) }"
                :aria-current="isCurrent(item.name) ? 'page' : undefined"
                :to="item.to"
              >
                <i :class="item.icon" aria-hidden="true" />
                <span>{{ item.label }}</span>
              </RouterLink>
            </div>
          </div>
        </div>
      </nav>
    </div>

    <div
      :class="['container', 'app-main', { 'app-main--activity': contextStore.currentView === 'activity' }]"
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
  padding: 0 10px;
  border-bottom: 1px solid var(--ms-border);
  background: #ffffff;
  box-shadow: 0 2px 10px rgba(15, 23, 42, 0.04);
  position: relative;
  z-index: 1020;
}

.nav-groups {
  width: 100%;
  display: flex;
  align-items: stretch;
  gap: 4px;
}

.nav-group {
  position: relative;
}

.nav-group--secondary {
  margin-left: auto;
}

.nav-group-trigger {
  min-height: 52px;
  border: 0;
  background: transparent;
  border-bottom: 3px solid transparent;
  color: #6d7079;
  font-weight: 700;
  letter-spacing: 0.01em;
  padding: 0.75rem 1rem 0.65rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.55rem;
  transition: color 0.2s ease, border-color 0.2s ease, background 0.2s ease;
  white-space: nowrap;
}

.nav-group-trigger:hover,
.nav-group-trigger:focus-visible {
  background: #fff7f2;
  color: #3a3d46;
  outline: none;
}

.nav-group--active > .nav-group-trigger,
.nav-group-trigger[aria-expanded="true"] {
  color: var(--ms-primary);
  background: #fff8f4;
  border-bottom-color: var(--ms-primary);
}

.nav-group-trigger--secondary {
  width: 48px;
  padding-inline: 0;
  font-size: 1.05rem;
}

.nav-group-chevron {
  font-size: 0.62rem;
  transition: transform 0.15s ease;
}

.nav-group-trigger[aria-expanded="true"] .nav-group-chevron {
  transform: rotate(180deg);
}

.nav-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  min-width: 230px;
  padding: 6px;
  border: 1px solid var(--ms-border);
  border-radius: 12px;
  background: #ffffff;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.16);
  z-index: 1040;
}

.nav-menu--secondary {
  right: 0;
  left: auto;
  min-width: 190px;
}

.nav-menu-item {
  min-height: 42px;
  padding: 0.65rem 0.75rem;
  border-radius: 8px;
  color: #4b4f58;
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 0.7rem;
  font-weight: 650;
}

.nav-menu-item i {
  width: 1.15rem;
  color: #777b84;
  text-align: center;
}

.nav-menu-item:hover,
.nav-menu-item:focus-visible {
  color: #2f3238;
  background: #fff7f2;
  outline: none;
}

.nav-menu-item.active {
  color: var(--ms-primary);
  background: #fff0e8;
}

.nav-menu-item.active i {
  color: var(--ms-primary);
}

.tab-beta {
  margin-left: 0.4rem;
  border-radius: 999px;
  border: 1px solid #f6b18a;
  background: #fff0e6;
  color: #c05a2f;
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  padding: 0.06rem 0.35rem;
  vertical-align: middle;
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
  .fixed-top {
    position: static;
  }

  .app-tabs-shell {
    margin-top: 0;
    position: sticky;
    top: 0;
    padding-inline: 6px;
  }

  .nav-groups {
    justify-content: space-between;
    gap: 0;
  }

  .nav-group-trigger {
    min-height: 48px;
    padding: 0.65rem 0.55rem 0.55rem;
    gap: 0.35rem;
    font-size: 0.86rem;
  }

  .nav-group:not(.nav-group--secondary) > .nav-group-trigger > i:first-child {
    display: none;
  }

  .nav-group-trigger--secondary {
    width: 42px;
    padding-inline: 0;
  }

  .nav-group:nth-last-child(-n + 2) .nav-menu {
    right: 0;
    left: auto;
  }

  .app-content {
    min-height: calc(100vh - 64px);
  }

  .toast-stack {
    right: 10px;
    left: 10px;
    max-width: none;
  }
}
</style>

import './assets/main.css'

import {createApp} from 'vue'
import {createPinia} from 'pinia'
import { useAthleteStore } from "@/stores/athlete";
import { useBackendRefreshStore } from "@/stores/backend-refresh";
import { useContextStore } from "@/stores/context";
import { filtersFromQuery, filtersToQuery } from "@/router/filter-query";
import mitt from 'mitt';

import App from './App.vue'
import router from './router'
import '@fortawesome/fontawesome-free/css/all.css';

export const eventBus = mitt();

const bootstrap = async (): Promise<void> => {
    const pinia = createPinia();

    const app = createApp(App);
    app.provide('eventBus', eventBus);

    app.directive("focus", {
        mounted(el) {
            el.focus();
        }
    });

    app.use(pinia);
    app.use(router);

    await router.isReady();
    const contextStore = useContextStore(pinia);
    const initialFilters = filtersFromQuery(router.currentRoute.value.query, {
        year: contextStore.currentYear,
        activityType: contextStore.currentActivityType,
    });
    contextStore.$patch({
        currentYear: initialFilters.year,
        currentActivityType: initialFilters.activityType,
    });
    contextStore.$subscribe((_mutation, state) => {
        const currentQuery = router.currentRoute.value.query;
        const expected = filtersToQuery({ year: state.currentYear, activityType: state.currentActivityType }, currentQuery);
        if (currentQuery.year !== expected.year || currentQuery.activityType !== expected.activityType) {
            void router.replace({ query: expected });
        }
    });
    router.afterEach((to) => {
        const filters = filtersFromQuery(to.query, {
            year: contextStore.currentYear,
            activityType: contextStore.currentActivityType,
        });
        void contextStore.updateCurrentFilters(filters.year, filters.activityType);
    });

    app.mount('#app');

    const athleteStore = useAthleteStore(pinia);
    athleteStore.fetchAthlete().catch((err: unknown) => {
        console.error('Failed to load athlete profile:', err);
    });
    const backendRefreshStore = useBackendRefreshStore(pinia);
    backendRefreshStore.watchStartupActivityRefresh().catch((err: unknown) => {
        console.error('Failed to watch backend activity refresh:', err);
    });
};

void bootstrap();

import './assets/main.css'

import {createApp} from 'vue'
import {createPinia} from 'pinia'
import {useContextStore} from '@/stores/context.js'
import mitt from 'mitt';
import Highcharts from 'highcharts';

import App from './App.vue'
import router from './router'
import '@fortawesome/fontawesome-free/css/all.css';

const registerHeatmapModule = (moduleRef: unknown): void => {
    if (typeof moduleRef === 'function') {
        (moduleRef as (hc: typeof Highcharts) => void)(Highcharts);
        return;
    }
    if (moduleRef && typeof (moduleRef as {default?: unknown}).default === 'function') {
        ((moduleRef as {default: (hc: typeof Highcharts) => void}).default)(Highcharts);
    }
};

export const eventBus = mitt();

const registerHighchartsModules = async (): Promise<void> => {
    // With Highcharts v12, side-effect import can crash app bootstrap in some bundles.
    // Dynamic import + guarded registration keeps the app alive even if module load fails.
    try {
        const heatmapModule = await import('highcharts/modules/heatmap');
        registerHeatmapModule(heatmapModule);
    } catch (error) {
        console.warn('Failed to register Highcharts heatmap module.', error);
    }
};

const bootstrap = async (): Promise<void> => {
    await registerHighchartsModules();

    const pinia = createPinia();

    // Init main store
    const mainStore = useContextStore(pinia);
    mainStore.fetchAthlete().then(() => {});

    const app = createApp(App);
    app.provide('eventBus', eventBus);

    app.directive("focus", {
        mounted(el) {
            el.focus();
        }
    });

    app.use(pinia);
    app.use(router);

    app.mount('#app');
};

void bootstrap();

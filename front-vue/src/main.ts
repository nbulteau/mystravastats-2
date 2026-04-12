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

const configureHighchartsTheme = (): void => {
    Highcharts.setOptions({
        colors: [
            '#fc4c02',
            '#4e79a7',
            '#59a14f',
            '#e15759',
            '#af7aa1',
            '#17becf',
            '#f28e2b',
            '#edc948',
            '#76b7b2',
            '#9c755f',
            '#6c7a89',
            '#b6992d',
        ],
        chart: {
            backgroundColor: 'transparent',
            style: {
                fontFamily: '"Avenir Next", "SF Pro Text", "Segoe UI", "Helvetica Neue", sans-serif',
            },
        },
        title: {
            style: {
                color: '#242428',
                fontWeight: '700',
            },
        },
        subtitle: {
            style: {
                color: '#6a6c75',
            },
        },
        xAxis: {
            lineColor: '#e5e7ee',
            tickColor: '#e5e7ee',
            labels: {
                style: {
                    color: '#6a6c75',
                },
            },
            title: {
                style: {
                    color: '#4e5058',
                },
            },
        },
        yAxis: {
            gridLineColor: '#eceff4',
            labels: {
                style: {
                    color: '#6a6c75',
                },
            },
            title: {
                style: {
                    color: '#4e5058',
                },
            },
        },
        legend: {
            itemStyle: {
                color: '#4f525a',
                fontWeight: '600',
            },
            itemHoverStyle: {
                color: '#242428',
            },
        },
        tooltip: {
            backgroundColor: '#ffffff',
            borderColor: '#ffd3c1',
            style: {
                color: '#2d3037',
            },
        },
        plotOptions: {
            series: {
                states: {
                    inactive: {
                        opacity: 1,
                    },
                },
            },
            line: {
                lineWidth: 2,
                marker: {
                    enabled: false,
                },
            },
            spline: {
                lineWidth: 2,
                marker: {
                    enabled: false,
                },
            },
            areaspline: {
                lineWidth: 2,
                marker: {
                    enabled: false,
                },
            },
            column: {
                borderWidth: 0,
                borderRadius: 2,
            },
        },
    });
};

const bootstrap = async (): Promise<void> => {
    await registerHighchartsModules();
    configureHighchartsTheme();

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

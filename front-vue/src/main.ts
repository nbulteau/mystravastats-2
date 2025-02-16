import './assets/main.css'

import {createApp} from 'vue'
import {createPinia} from 'pinia'
import {useContextStore} from '@/stores/context.js'
import mitt from 'mitt';

import App from './App.vue'
import router from './router'
import '@fortawesome/fontawesome-free/css/all.css';

const pinia = createPinia()

export const eventBus = mitt();


// Init main store
const mainStore = useContextStore(pinia)
mainStore.fetchAthlete().then(() => {})

const app = createApp(App)
app.provide('eventBus', eventBus)

app.directive("focus", {
    mounted(el) {
        el.focus();
    }
});

app.use(pinia)
app.use(router)

app.mount('#app')

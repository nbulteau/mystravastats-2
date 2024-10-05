import './assets/main.css'

import {createApp} from 'vue'
import {createPinia} from 'pinia'
import {useContextStore} from '@/stores/context.js'

import App from './App.vue'
import router from './router'
import '@fortawesome/fontawesome-free/css/all.css';

const pinia = createPinia()

// Init main store
const mainStore = useContextStore(pinia)
mainStore.fetchAthlete().then(() => {})

const app = createApp(App)

app.directive("focus", {
    mounted(el) {
        el.focus();
    }
});

app.use(pinia)
app.use(router)

app.mount('#app')
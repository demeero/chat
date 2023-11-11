import {createApp} from 'vue'
import {createPinia} from 'pinia'
import Toast, {POSITION} from 'vue-toastification';
import 'vue-toastification/dist/index.css';
import piniaPluginPersistedState from 'pinia-plugin-persistedstate';

import App from './App.vue'
import router from './router'

const toastOpts = {
    position: POSITION.TOP_CENTER
};

const app = createApp(App)

const pinia = createPinia();
pinia.use(piniaPluginPersistedState);

app.use(pinia)
app.use(router)
app.use(Toast, toastOpts)

app.mount('#app')

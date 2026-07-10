import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import { routes } from './router'

const app = createApp(App)

const router = createRouter({
  history: createWebHistory(),
  routes,
})

app.use(createPinia())
app.use(router)
app.mount('#app')

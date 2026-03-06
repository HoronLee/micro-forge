import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '@/views/HomeView.vue'
import ElementDemoView from '@/views/ElementDemoView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/element-demo',
      name: 'element-demo',
      component: ElementDemoView,
    },
  ],
})

export default router

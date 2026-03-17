import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import DashboardView from '../views/DashboardView.vue'
import { useAuth } from '../composables/useAuth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    { path: '/', name: 'dashboard', component: DashboardView },
  ],
})

router.beforeEach((to) => {
  const { isAuthenticated } = useAuth()
  if (to.path !== '/login' && !isAuthenticated.value) {
    return '/login'
  }
  if (to.path === '/login' && isAuthenticated.value) {
    return '/'
  }
  return true
})

export default router

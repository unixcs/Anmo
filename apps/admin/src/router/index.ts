import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from '../platform/auth'
import AdminLayout from '../layout/AdminLayout.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', component: () => import('../pages/LoginPage.vue') },
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: () => import('../pages/DashboardPage.vue') },
        { path: 'appointments', component: () => import('../pages/AppointmentsPage.vue') },
        { path: 'members', component: () => import('../pages/MembersPage.vue') },
        { path: 'card-templates', component: () => import('../pages/CardTemplatesPage.vue') },
        { path: 'services', component: () => import('../pages/ServicesPage.vue') },
        { path: 'content', component: () => import('../pages/ContentPage.vue') },
        { path: 'records', component: () => import('../pages/RecordsPage.vue') },
        { path: 'insights', component: () => import('../pages/InsightsPage.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  if (to.path !== '/login' && !getToken()) return '/login'
  if (to.path === '/login' && getToken()) return '/dashboard'
  return true
})

export default router

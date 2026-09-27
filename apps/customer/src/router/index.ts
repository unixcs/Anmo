import { createRouter, createWebHashHistory } from 'vue-router'
import { currentToken } from '../platform/auth/session'

const routes = [
  { path: '/', name: 'home', component: () => import('../pages/HomePage.vue') },
  { path: '/services', name: 'services', component: () => import('../pages/ServicesPage.vue') },
  { path: '/booking', name: 'booking', component: () => import('../pages/BookingPage.vue'), meta: { auth: true } },
  { path: '/login', name: 'login', component: () => import('../pages/LoginPage.vue') },
  { path: '/me', name: 'me', component: () => import('../pages/MePage.vue'), meta: { auth: true } },
  { path: '/me/cards', name: 'cards', component: () => import('../pages/CardsPage.vue'), meta: { auth: true } },
  { path: '/me/qrcode', name: 'qrcode', component: () => import('../pages/QRCodePage.vue'), meta: { auth: true } },
  { path: '/me/appointments', name: 'my-appointments', component: () => import('../pages/MyAppointmentsPage.vue'), meta: { auth: true } },
  { path: '/me/history', name: 'history', component: () => import('../pages/HistoryPage.vue'), meta: { auth: true } },
  { path: '/me/profile', name: 'profile', component: () => import('../pages/ProfilePage.vue'), meta: { auth: true } },
  { path: '/about', name: 'about', component: () => import('../pages/AboutPage.vue') },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// 路由守卫：未登录访问受保护页跳登录
router.beforeEach((to) => {
  if (to.meta.auth && !currentToken()) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && currentToken()) {
    return { name: 'me' }
  }
  return true
})

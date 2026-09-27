<script setup lang="ts">
import { useRoute } from 'vue-router'
const route = useRoute()
const TABS = [
  { path: '/', label: '首页', icon: '🏠', match: ['home'] },
  { path: '/services', label: '服务', icon: '💆', match: ['services'] },
  { path: '/booking', label: '预约', icon: '📅', match: ['booking'] },
  { path: '/me', label: '我的', icon: '👤', match: ['me', 'cards', 'my-appointments', 'history', 'profile'] },
]
function active(tab: { match: string[] }): boolean {
  return tab.match.includes(route.name as string)
}
</script>

<template>
  <div class="app-shell">
    <main class="app-main">
      <RouterView />
    </main>
    <nav class="tabbar">
      <RouterLink
        v-for="tab in TABS"
        :key="tab.path"
        :to="tab.path"
        class="tab"
        :class="{ active: active(tab) }"
      >
        <span class="tab-icon">{{ tab.icon }}</span>
        <span>{{ tab.label }}</span>
      </RouterLink>
    </nav>
  </div>
</template>

<style scoped>
.app-shell { min-height: 100vh; display: flex; flex-direction: column; }
.app-main { flex: 1; padding-bottom: 64px; }
.tabbar {
  position: fixed; bottom: 0; left: 0; right: 0; height: 60px;
  display: flex; background: var(--card); border-top: 1px solid var(--border);
}
.tab {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center;
  font-size: 12px; color: var(--muted-foreground); text-decoration: none; gap: 2px;
}
.tab.active { color: var(--primary); }
.tab-icon { font-size: 20px; }
</style>

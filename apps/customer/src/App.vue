<script setup lang="ts">
import { useRoute } from 'vue-router'
import AppIcon, { type IconName } from './components/ui/AppIcon.vue'

const route = useRoute()
const TABS: { path: string; label: string; icon: IconName; match: string[] }[] = [
  { path: '/', label: '首页', icon: 'home', match: ['home'] },
  { path: '/services', label: '服务', icon: 'sparkles', match: ['services'] },
  { path: '/booking', label: '预约', icon: 'calendar', match: ['booking'] },
  { path: '/me', label: '我的', icon: 'user', match: ['me', 'cards', 'my-appointments', 'history', 'profile', 'qrcode'] },
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
        <AppIcon :name="tab.icon" :size="22" :stroke-width="active(tab) ? 2 : 1.8" />
        <span class="tab-label">{{ tab.label }}</span>
      </RouterLink>
    </nav>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
}

.app-main {
  flex: 1;
}

.tabbar {
  position: fixed;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  max-width: 480px;
  display: flex;
  padding: 6px 0 calc(6px + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  border-top: 1px solid var(--border);
  z-index: 40;
}

.tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  color: var(--muted-foreground);
  text-decoration: none;
  transition: color 0.15s var(--ease);
}

.tab.active {
  color: var(--primary);
}

.tab-label {
  font: 500 11px/14px var(--font-stack);
}
</style>

<template>
  <el-container class="layout">
    <el-aside v-if="!isMobile" width="220px" class="aside">
      <div class="brand">Anmo · 商家管理</div>
      <el-menu :default-active="route.path" router background-color="#001529" text-color="#a6adb4"
        active-text-color="#ffffff">
        <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">{{ item.label }}</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header" :class="{ compact: isMobile }">
        <span v-if="isMobile" class="burger" @click="menuOpen = true">☰</span>
        <span class="title">{{ title }}</span>
        <span class="spacer" />
        <span v-if="!isMobile" class="user">{{ user?.name }}（{{ roleText }}）</span>
        <el-button type="primary" :size="isMobile ? 'default' : 'small'" class="scan-btn"
          @click="scanVisible = true">扫码核销</el-button>
        <el-button v-if="!isMobile" link type="danger" @click="logout">退出登录</el-button>
        <el-button v-else link type="danger" size="small" @click="logout">退出</el-button>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>

    <!-- 手机端抽屉导航 -->
    <el-drawer v-if="isMobile" v-model="menuOpen" direction="ltr" size="230px" :with-header="false">
      <div class="drawer-inner">
        <div class="brand dark">Anmo · 商家管理</div>
        <el-menu :default-active="route.path" router @select="menuOpen = false">
          <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">{{ item.label }}</el-menu-item>
        </el-menu>
      </div>
    </el-drawer>

    <ScanRedeemDialog v-model="scanVisible" @settled="() => {}" />
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getUser, clearSession } from '../platform/auth'
import ScanRedeemDialog from '../components/ScanRedeemDialog.vue'
import { useIsMobile } from '../core/useMedia'

const route = useRoute()
const router = useRouter()
const user = getUser()
const isMobile = useIsMobile()
const scanVisible = ref(false)
const menuOpen = ref(false)

const navItems = [
  { path: '/dashboard', label: '今日工作台' },
  { path: '/appointments', label: '预约管理' },
  { path: '/members', label: '会员管理' },
  { path: '/card-templates', label: '会员卡' },
  { path: '/services', label: '服务管理' },
  { path: '/content', label: '内容管理' },
  { path: '/records', label: '收款与核销' },
  { path: '/insights', label: '运营洞察' },
]

const roleText = computed(() => (user?.role === 'OWNER' ? '老板' : '店员'))
const title = computed(() => {
  const map: Record<string, string> = Object.fromEntries(navItems.map((i) => [i.path, i.label]))
  return map[route.path] ?? ''
})

function logout() {
  clearSession()
  void router.push('/login')
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
}
.aside {
  background: #001529;
}
.brand {
  color: #fff;
  font-size: 17px;
  font-weight: 600;
  padding: 20px 20px 14px;
  letter-spacing: 1px;
}
.brand.dark {
  color: #1f2d3d;
}
.aside :deep(.el-menu) {
  border-right: none;
}
.drawer-inner {
  height: 100%;
}
.drawer-inner :deep(.el-menu) {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}
.header.compact {
  gap: 8px;
  padding: 0 12px;
  height: 52px;
  position: sticky;
  top: 0;
  z-index: 20;
}
.burger {
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  padding: 4px 6px;
  color: #303133;
}
.title {
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
}
.spacer {
  flex: 1;
}
.user {
  color: #606266;
  font-size: 13px;
}
.scan-btn {
  white-space: nowrap;
}
.main {
  padding: 16px;
}
</style>

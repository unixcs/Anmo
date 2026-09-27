<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="brand">Anmo · 商家管理</div>
      <el-menu :default-active="route.path" router background-color="#001529" text-color="#a6adb4"
        active-text-color="#ffffff">
        <el-menu-item index="/dashboard">今日工作台</el-menu-item>
        <el-menu-item index="/appointments">预约管理</el-menu-item>
        <el-menu-item index="/members">会员管理</el-menu-item>
        <el-menu-item index="/card-templates">会员卡</el-menu-item>
        <el-menu-item index="/services">服务管理</el-menu-item>
        <el-menu-item index="/content">内容管理</el-menu-item>
        <el-menu-item index="/records">收款与核销</el-menu-item>
        <el-menu-item index="/insights">运营洞察</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span class="title">{{ title }}</span>
        <span class="spacer" />
        <span class="user">{{ user?.name }}（{{ roleText }}）</span>
        <el-button type="primary" size="small" @click="scanVisible = true">扫码核销</el-button>
        <el-button link type="danger" @click="logout">退出登录</el-button>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
    <ScanRedeemDialog v-model="scanVisible" @settled="() => {}" />
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getUser, clearSession } from '../platform/auth'
import ScanRedeemDialog from '../components/ScanRedeemDialog.vue'

const route = useRoute()
const router = useRouter()
const user = getUser()
const scanVisible = ref(false)

const roleText = computed(() => (user?.role === 'OWNER' ? '老板' : '店员'))
const title = computed(() => {
  const map: Record<string, string> = {
    '/dashboard': '今日工作台',
    '/appointments': '预约管理',
    '/members': '会员管理',
    '/card-templates': '会员卡',
    '/services': '服务管理',
    '/content': '内容管理',
    '/records': '收款与核销',
    '/insights': '运营洞察',
  }
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
.aside :deep(.el-menu) {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}
.title {
  font-size: 16px;
  font-weight: 600;
}
.spacer {
  flex: 1;
}
.user {
  color: #606266;
  font-size: 13px;
}
.main {
  padding: 16px;
}
</style>

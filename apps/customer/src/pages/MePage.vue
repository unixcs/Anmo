<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import { signOut } from '../platform/auth/session'
import { useRouter } from 'vue-router'
import { notify } from '../platform/notify/toast'

const router = useRouter()
const name = ref('')
const memberNo = ref('')

onMounted(async () => {
  try {
    const res = await api.myProfile()
    name.value = res.member.name || '未设置昵称'
    memberNo.value = res.member.member_no
  } catch (e) {
    notify((e as Error).message)
  }
})

function logout(): void {
  signOut()
  router.replace('/login')
}
</script>

<template>
  <div class="page me">
    <div class="head">
      <div class="avatar">👤</div>
      <div>
        <div class="name">{{ name }}</div>
        <div class="no">会员号 {{ memberNo }}</div>
      </div>
    </div>
    <div class="menu">
      <RouterLink to="/me/cards" class="row">💳 我的会员卡 <span class="arrow">›</span></RouterLink>
      <RouterLink to="/me/qrcode" class="row">🔳 我的核销码 <span class="arrow">›</span></RouterLink>
      <RouterLink to="/me/appointments" class="row">📅 我的预约 <span class="arrow">›</span></RouterLink>
      <RouterLink to="/me/history" class="row">📖 历史记录 <span class="arrow">›</span></RouterLink>
      <RouterLink to="/me/profile" class="row">✏️ 个人资料 <span class="arrow">›</span></RouterLink>
      <RouterLink to="/about" class="row">ℹ️ 关于我们 <span class="arrow">›</span></RouterLink>
    </div>
    <button class="logout" @click="logout">退出登录</button>
  </div>
</template>

<style scoped>
.me { padding: 20px 16px; }
.head { display: flex; align-items: center; gap: 14px; background: var(--card); padding: 18px; border-radius: 12px; }
.avatar { font-size: 34px; }
.name { font-size: 17px; font-weight: 600; }
.no { color: var(--muted-foreground); font-size: 13px; }
.menu { margin-top: 14px; background: var(--card); border-radius: 12px; overflow: hidden; }
.row { display: flex; justify-content: space-between; padding: 15px 16px; color: var(--foreground); text-decoration: none; border-bottom: 1px solid var(--border); }
.arrow { color: var(--border); }
.logout { width: 100%; margin-top: 20px; height: 44px; background: var(--card); color: var(--primary); border: none; border-radius: 12px; }
</style>

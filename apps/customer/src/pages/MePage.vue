<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { api } from '../core/api/endpoints'
import type { MemberCard } from '../core/models/models'
import { todayStr } from '../core/logic/booking'
import { profileProgress, type ProfileProgress } from '../core/utils/profile'
import { signOut } from '../platform/auth/session'
import { useRouter } from 'vue-router'
import { notify } from '../platform/notify/toast'
import AppCell from '../components/ui/AppCell.vue'
import AppStat from '../components/ui/AppStat.vue'

const router = useRouter()
const name = ref('')
const memberNo = ref('')
const phone = ref('')
const cardCount = ref(0)
const remainingTotal = ref(0)
const upcoming = ref(0)
// 资料完善度（V2.2 R4）：默认 100（未加载前不渲染提示条）
const progress = ref<ProfileProgress>({ pct: 100, missing: [] })
const progressText = computed(() => {
  const missing = progress.value.missing
  if (missing.includes('name') && missing.includes('phone')) return '完善称呼与手机号'
  if (missing.includes('name')) return '完善称呼'
  return '完善手机号，方便预约联系'
})

onMounted(async () => {
  try {
    const res = await api.myProfile()
    name.value = res.member.name || '未设置昵称'
    memberNo.value = res.member.member_no
    phone.value = res.member.phone
    progress.value = profileProgress(res.member)
    // 统计块：卡与即将到店（失败不影响页面主体）
    const [cards, apts] = await Promise.all([
      api.myCards().catch((): MemberCard[] => []),
      api.myAppointments('').catch((): never[] => []),
    ])
    const list = cards ?? []
    cardCount.value = list.length
    remainingTotal.value = list
      .filter((c) => c.status === 'ACTIVE')
      .reduce((sum, c) => sum + c.remaining_count, 0)
    // 即将到店 = 今天（北京）及以后的活跃预约；日期口径与预约页一致，不用设备时区
    const today = todayStr()
    upcoming.value = (apts ?? []).filter(
      (a) =>
        (a.status === 'WAITING' || a.status === 'IN_SERVICE') &&
        a.scheduled_start.slice(0, 10) >= today,
    ).length
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
    <!-- 个人头 -->
    <div class="head">
      <div class="avatar">{{ name.slice(0, 1) }}</div>
      <div class="who">
        <div class="name">{{ name }}</div>
        <div class="no num">会员号 {{ memberNo }}</div>
      </div>
    </div>

    <!-- 资料完善度（V2.2 R4）：矮提示条，已完善隐藏，不弹窗不挡操作 -->
    <div v-if="progress.pct < 100" class="profile-tip">
      <span class="tip-text num">资料完善度 {{ progress.pct }}%｜{{ progressText }}</span>
      <button type="button" class="tip-go pressable" @click="router.push('/me/profile')">去完善</button>
    </div>

    <!-- 统计 -->
    <div class="stats">
      <AppStat label="会员卡" :value="cardCount" hint="张" />
      <AppStat label="剩余次数" :value="remainingTotal" hint="次" />
      <AppStat label="即将到店" :value="upcoming" hint="次" />
    </div>

    <!-- 菜单 -->
    <div class="cell-group menu">
      <AppCell to="/me/qrcode" icon="qr" title="我的核销码" desc="到店出示，扫码结算" />
      <AppCell to="/me/appointments" icon="calendar" title="我的预约" desc="改期、取消都在这里" />
      <AppCell to="/me/cards" icon="wallet" title="我的会员卡" desc="余额与使用明细" />
      <AppCell to="/me/history" icon="history" title="历史服务" desc="已完成的服务记录" />
      <AppCell to="/me/profile" icon="pencil" title="个人资料" desc="称呼、性别、生日" />
      <AppCell to="/about" icon="info" title="关于我们" desc="门店信息与联系方式" />
    </div>

    <button type="button" class="btn outline block lg logout pressable" @click="logout">
      退出登录
    </button>
  </div>
</template>

<style scoped>
.head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 6px 4px 20px;
}

.avatar {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-full);
  background: var(--primary);
  color: var(--primary-foreground);
  display: flex;
  align-items: center;
  justify-content: center;
  font: 600 22px/1 var(--font-stack);
}

.who {
  min-width: 0;
}

.name {
  font: var(--font-title);
}

.no {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-top: 2px;
}

.profile-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--primary-soft);
  border-radius: var(--radius-md);
  padding: 8px 12px;
  margin: -8px 4px 16px;
}

.tip-text {
  flex: 1;
  font: var(--font-caption);
  color: var(--foreground);
}

.tip-go {
  flex: 0 0 auto;
  border: none;
  background: none;
  padding: 0;
  font: 600 12px/16px var(--font-stack);
  color: var(--primary);
  cursor: pointer;
}

.stats {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.logout {
  margin-top: 24px;
  color: var(--muted-foreground);
}
</style>

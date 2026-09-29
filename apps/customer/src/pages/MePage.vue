<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { MemberCard } from '../core/models/models'
import { todayStr } from '../core/logic/booking'
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

onMounted(async () => {
  try {
    const res = await api.myProfile()
    name.value = res.member.name || '未设置昵称'
    memberNo.value = res.member.member_no
    phone.value = res.member.phone
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
        a.scheduled_start.slice(0, 10) >= todayStr,
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

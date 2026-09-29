<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import QRCode from 'qrcode'
import { api } from '../core/api/endpoints'
import type { Appointment } from '../core/models/models'
import { todayStr as bjTodayStr } from '../core/logic/booking'
import { notify } from '../platform/notify/toast'
import AppIcon from '../components/ui/AppIcon.vue'
import AppStatusBadge from '../components/ui/AppStatusBadge.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

// 核销码页（BRAND-GUIDELINES §6 / D21）：顾客永远只出示一个码。
// 会员码 ANMO-MEMBER:<member_id> 仅对持 ACTIVE 卡顾客展示；
// 今日预约以文字列表呈现在码下方，商家扫码后自选预约结算——
// 不再生成 ANMO-APT 预约单码（历史多码 bug 的根源）。

interface TodayApt extends Appointment {
  serviceNames: string
}

const name = ref('')
const memberNo = ref('')
const memberId = ref('')
const memberQr = ref('')
const hasActiveCard = ref(false)
const loading = ref(true)
const nowText = ref('')
const todayApts = ref<TodayApt[]>([])
let timer: number | undefined

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

function tick(): void {
  const d = new Date()
  nowText.value = `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// "今天"用北京时间锚定（与预约页/小程序同口径）：跨店营业日界限不由设备时区决定
const todayStr = computed(bjTodayStr)

function aptTime(a: Appointment): string {
  if (a.slot_type === 'HALF_DAY') return a.day_part === 'AM' ? '上午' : '下午'
  return a.scheduled_start.slice(11, 16)
}

onMounted(async () => {
  tick()
  timer = window.setInterval(tick, 1000)
  try {
    const [profile, cards, apts] = await Promise.all([
      api.myProfile(),
      api.myCards(),
      api.myAppointments(''),
    ])
    name.value = profile.member.name || '未设置昵称'
    memberNo.value = profile.member.member_no
    memberId.value = profile.member.id
    hasActiveCard.value = (cards ?? []).some((c) => c.status === 'ACTIVE')
    if (hasActiveCard.value) {
      memberQr.value = await QRCode.toDataURL(`ANMO-MEMBER:${memberId.value}`, {
        width: 560,
        margin: 2,
        errorCorrectionLevel: 'M',
        color: { dark: '#221E1B', light: '#FFFFFF' },
      })
    }
    // 列表项已内嵌 services 快照，无需逐单拉详情
    todayApts.value = (apts ?? [])
      .filter(
        (a) =>
          (a.status === 'WAITING' || a.status === 'IN_SERVICE') &&
          a.scheduled_start.slice(0, 10) === todayStr.value,
      )
      .sort((a, b) => a.scheduled_start.localeCompare(b.scheduled_start))
      .map((a) => {
        const item = a as Appointment & { services?: { service_name_snapshot: string }[] }
        const names = (item.services ?? []).map((s) => s.service_name_snapshot).filter(Boolean)
        return { ...a, serviceNames: names.join(' · ') }
      })
  } catch (e) {
    notify((e as Error).message)
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (timer !== undefined) window.clearInterval(timer)
})
</script>

<template>
  <div class="page qrcode">
    <div class="page-head">
      <h1>我的核销码</h1>
      <p class="sub">到店后出示给商家，扫码即完成服务登记</p>
    </div>

    <AppSkeleton v-if="loading" variant="card" />

    <template v-else>
      <!-- 会员码块：仅持 ACTIVE 卡顾客可见（D21） -->
      <div v-if="hasActiveCard" class="qr-block card">
        <div class="qr-top">
          <span class="brand-seal">安摩</span>
          <span class="live"><i class="live-dot" />有效会员</span>
        </div>
        <img v-if="memberQr" :src="memberQr" alt="我的核销码" class="qr" />
        <div class="who">
          <div class="name">{{ name }}</div>
          <div class="no num">会员号 {{ memberNo }}</div>
        </div>
        <div class="clock num">
          <AppIcon name="clock" :size="14" />
          {{ nowText }}
        </div>
        <p class="hint">每次结算由商家确认后生效，请勿将二维码截图发给他人</p>
      </div>

      <!-- 无卡：锁定引导，不出任何码 -->
      <div v-else class="card">
        <AppEmpty
          icon="lock"
          main="暂无有效会员卡"
          sub="办卡后出示会员码即可按卡结算，可到店咨询店主办理"
        >
          <RouterLink to="/services" class="btn secondary sm">先看看服务项目</RouterLink>
        </AppEmpty>
      </div>

      <!-- 今日预约：文字列表（不做预约单码） -->
      <div v-if="hasActiveCard" class="today">
        <div class="today-head">
          <h2>今日预约</h2>
          <span v-if="todayApts.length" class="count">{{ todayApts.length }} 个</span>
        </div>
        <div v-if="todayApts.length" class="card plain today-list">
          <div v-for="a in todayApts" :key="a.id" class="today-row">
            <span class="time num">{{ aptTime(a) }}</span>
            <span class="svc">{{ a.serviceNames || '到店与商家确认服务' }}</span>
            <AppStatusBadge :status="a.status" />
          </div>
        </div>
        <AppEmpty v-else icon="calendar" main="今天没有预约" sub="需要的话可以现在约一个">
          <RouterLink to="/booking" class="btn secondary sm">去预约</RouterLink>
        </AppEmpty>
      </div>
    </template>
  </div>
</template>

<style scoped>
.qrcode {
  max-width: 420px;
  margin: 0 auto;
}

/* 会员码块 */
.qr-block {
  text-align: center;
  padding: 20px 18px;
}

.qr-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.brand-seal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--primary);
  color: var(--primary-foreground);
  font: 600 15px/1 var(--font-stack);
  letter-spacing: 2px;
  text-indent: 2px;
}

.live {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font: var(--font-caption);
  color: var(--success);
  background: var(--success-soft);
  border-radius: var(--radius-full);
  padding: 4px 10px;
}

.live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--success);
  animation: anmo-breathe 2s ease-in-out infinite;
}

.qr {
  width: 100%;
  max-width: 230px;
  display: block;
  margin: 0 auto;
}

.who {
  margin-top: 10px;
}

.name {
  font: 600 17px/24px var(--font-stack);
}

.no {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-top: 2px;
}

.clock {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-top: 10px;
  font: var(--font-sub);
  color: var(--muted-foreground);
  background: var(--muted);
  border-radius: var(--radius-full);
  padding: 4px 12px;
}

.hint {
  font: var(--font-caption);
  color: var(--muted-foreground);
  margin: 14px 0 0;
  border-top: 1px dashed var(--border);
  padding-top: 12px;
}

/* 今日预约文字列表 */
.today {
  margin-top: 24px;
}

.today-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 10px;
}

.today-head h2 {
  font: var(--font-title);
  margin: 0;
}

.count {
  font: var(--font-caption);
  color: var(--muted-foreground);
}

.today-list {
  padding: 4px 16px;
}

.today-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
}

.today-row + .today-row {
  border-top: 1px solid var(--border);
}

.today-row .time {
  flex: 0 0 44px;
  font-weight: 600;
  font-size: 15px;
}

.today-row .svc {
  flex: 1;
  min-width: 0;
  font: var(--font-sub);
  color: var(--muted-foreground);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import QRCode from 'qrcode'
import { api } from '../core/api/endpoints'
import { notify } from '../platform/notify/toast'

interface Appointment {
  id: string
  status: string
  scheduled_start: string
  slot_type?: string
  day_part?: string
  service_name?: string
}

const name = ref('')
const memberNo = ref('')
const phone = ref('')
const memberId = ref('')
const memberQr = ref('')
// 核销码门槛（D21）：只有持有效会员卡的顾客才出会员码
const hasActiveCard = ref(false)
const cardsReady = ref(false)
const nowText = ref('')
// 今日待服务预约的预约单码（ANMO-APT，商家扫码直接关联该预约结算）
const todayApts = ref<Appointment[]>([])
const aptQrs = ref<Record<string, string>>({})
let timer: number | undefined

function tick(): void {
  const d = new Date()
  const p = (n: number): string => String(n).padStart(2, '0')
  nowText.value = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

const todayStr = computed(() => nowText.value.slice(0, 10))

function aptTime(a: Appointment): string {
  if (a.slot_type === 'HALF_DAY') return a.day_part === 'AM' ? '上午' : '下午'
  return a.scheduled_start.slice(11, 16)
}

async function renderAptCodes(): Promise<void> {
  for (const a of todayApts.value) {
    aptQrs.value[a.id] = await QRCode.toDataURL(`ANMO-APT:${a.id}`, {
      width: 560,
      margin: 4,
      errorCorrectionLevel: 'M',
    })
  }
}

onMounted(async () => {
  tick()
  timer = window.setInterval(tick, 1000)
  try {
    const [profile, cards, apts] = await Promise.all([
      api.myProfile(),
      api.myCards(),
      api.myAppointments(),
    ])
    name.value = profile.member.name || '未设置昵称'
    memberNo.value = profile.member.member_no
    phone.value = profile.member.phone
    memberId.value = profile.member.id
    hasActiveCard.value = (cards ?? []).some((c) => c.status === 'ACTIVE')
    cardsReady.value = true
    if (hasActiveCard.value) {
      // 会员码内容：固定前缀 + 会员ID，商家端扫码定位会员后按卡结算
      memberQr.value = await QRCode.toDataURL(`ANMO-MEMBER:${memberId.value}`, {
        width: 560,
        margin: 4,
        errorCorrectionLevel: 'M',
      })
    }
    todayApts.value = (apts ?? []).filter(
      (a) =>
        (a.status === 'WAITING' || a.status === 'IN_SERVICE') &&
        a.scheduled_start.slice(0, 10) === todayStr.value,
    )
    // 列表端点不带服务名，逐单补齐（今日待服务预约通常 0-2 个）
    for (const a of todayApts.value) {
      try {
        const d = await api.appointment(a.id)
        a.service_name = d.services?.[0]?.service_name_snapshot ?? ''
      } catch {
        a.service_name = ''
      }
    }
    await renderAptCodes()
  } catch (e) {
    notify((e as Error).message)
  }
})

onUnmounted(() => {
  if (timer !== undefined) window.clearInterval(timer)
})
</script>

<template>
  <div class="page qrcode">
    <div class="card">
      <p class="tip">到店结算时，向商家出示对应二维码</p>

      <!-- 今日预约单码：商家扫码直接关联该预约结算 -->
      <template v-if="todayApts.length > 0">
        <div v-for="a in todayApts" :key="a.id" class="apt-block">
          <p class="apt-label">今日预约 · {{ a.service_name || '服务' }} · {{ aptTime(a) }}</p>
          <img v-if="aptQrs[a.id]" :src="aptQrs[a.id]" alt="预约单码" class="qr" />
          <p class="apt-sub">出示此码，商家扫码后直接开始本次服务结算</p>
        </div>
      </template>
      <p v-else class="none-apt">今日没有待到店的预约</p>

      <div class="divider" />

      <!-- 会员卡核销码（D21：持有效卡才出码） -->
      <template v-if="cardsReady && !hasActiveCard">
        <div class="locked">
          <div class="locked-icon">🔒</div>
          <p class="locked-title">暂无有效会员卡</p>
          <p class="locked-text">办卡后可出示会员码按卡结算，更方便。<br />可到店咨询店主办理。</p>
          <RouterLink to="/services" class="locked-link">先看看服务项目 →</RouterLink>
        </div>
      </template>
      <template v-else>
        <p class="sec-label">会员卡核销码</p>
        <img v-if="memberQr" :src="memberQr" alt="我的核销码" class="qr" />
        <div v-else class="loading">生成中…</div>
      </template>

      <div class="who">
        <div class="name">{{ name }}</div>
        <div class="no">手机号 {{ phone }}</div>
        <div class="no">会员号 {{ memberNo }}</div>
        <div class="clock">🕒 {{ nowText }}</div>
      </div>
      <p class="hint">· 每次结算由商家确认后生效，扣减对应次数<br />· 请勿将二维码截图发给他人</p>
    </div>
  </div>
</template>

<style scoped>
.qrcode { padding: 24px 16px; display: flex; justify-content: center; }
.card { background: var(--card); border-radius: 16px; padding: 22px 18px; width: 100%; max-width: 380px; text-align: center; }
.tip { margin: 0 0 14px; font-size: 14px; color: var(--primary); font-weight: 600; }
.apt-block { margin-bottom: 6px; }
.apt-label { margin: 0 0 8px; font-size: 14px; font-weight: 600; color: var(--foreground); }
.apt-sub { color: var(--muted-foreground); font-size: 12px; margin: 4px 0 0; }
.none-apt { color: var(--muted-foreground); font-size: 13px; margin: 0 0 6px; }
.divider { border-top: 1px dashed var(--border); margin: 14px 0; }
.sec-label { margin: 0 0 8px; font-size: 14px; font-weight: 600; color: var(--foreground); }
.qr { width: 100%; max-width: 240px; display: block; margin: 0 auto; }
.loading { padding: 30px 0; color: var(--muted-foreground); }
.locked { padding: 10px 0 20px; }
.locked-icon { font-size: 34px; }
.locked-title { font-size: 16px; font-weight: 600; color: var(--muted-foreground); margin: 8px 0 6px; }
.locked-text { color: var(--muted-foreground); font-size: 13px; line-height: 1.8; }
.locked-link { display: inline-block; margin-top: 10px; color: var(--primary); font-size: 14px; text-decoration: none; }
.who { margin-top: 14px; }
.name { font-size: 17px; font-weight: 600; }
.no { color: var(--muted-foreground); font-size: 13px; margin-top: 2px; }
.clock { color: var(--muted-foreground); font-size: 14px; margin-top: 8px; font-variant-numeric: tabular-nums; }
.hint { margin-top: 14px; text-align: left; color: var(--muted-foreground); font-size: 12px; line-height: 1.8; }
</style>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import QRCode from 'qrcode'
import { api } from '../core/api/endpoints'
import { notify } from '../platform/notify/toast'

const name = ref('')
const memberNo = ref('')
const phone = ref('')
const qrDataUrl = ref('')
// 核销码门槛（D21）：只有持有效会员卡的顾客才出码
const hasActiveCard = ref(false)
const cardsReady = ref(false)
const nowText = ref('')
let timer: number | undefined

function tick(): void {
  const d = new Date()
  const p = (n: number): string => String(n).padStart(2, '0')
  nowText.value = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

onMounted(async () => {
  tick()
  timer = window.setInterval(tick, 1000)
  try {
    const [profile, cards] = await Promise.all([api.myProfile(), api.myCards()])
    name.value = profile.member.name || '未设置昵称'
    memberNo.value = profile.member.member_no
    phone.value = profile.member.phone
    hasActiveCard.value = (cards ?? []).some((c) => c.status === 'ACTIVE')
    cardsReady.value = true
    if (!hasActiveCard.value) return
    // 核销码内容：固定前缀 + 会员ID，商家端扫码即定位到该会员
    qrDataUrl.value = await QRCode.toDataURL(`ANMO-MEMBER:${profile.member.id}`, {
      width: 560,
      margin: 4,
      errorCorrectionLevel: 'M',
    })
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
      <p class="tip">到店结算时，向商家出示此码即可核销</p>

      <!-- 无有效会员卡：不出码（D21） -->
      <template v-if="cardsReady && !hasActiveCard">
        <div class="locked">
          <div class="locked-icon">🔒</div>
          <p class="locked-title">暂无有效会员卡</p>
          <p class="locked-text">办卡后即可出示核销码，到店结算更方便。<br />可到店咨询店主办理。</p>
          <RouterLink to="/services" class="locked-link">先看看服务项目 →</RouterLink>
        </div>
      </template>

      <template v-else>
        <img v-if="qrDataUrl" :src="qrDataUrl" alt="我的核销码" class="qr" />
        <div v-else class="loading">生成中…</div>
      </template>

      <div class="who">
        <div class="name">{{ name }}</div>
        <div class="no">手机号 {{ phone }}</div>
        <div class="no">会员号 {{ memberNo }}</div>
        <div class="clock">🕒 {{ nowText }}</div>
      </div>
      <p class="hint">· 每次核销由商家确认后生效，扣减对应次数<br />· 请勿将二维码截图发给他人</p>
    </div>
  </div>
</template>

<style scoped>
.qrcode { padding: 24px 16px; display: flex; justify-content: center; }
.card { background: #fff; border-radius: 16px; padding: 22px 18px; width: 100%; max-width: 380px; text-align: center; }
.tip { margin: 0 0 14px; font-size: 14px; color: #c85f5f; font-weight: 600; }
.qr { width: 100%; max-width: 260px; display: block; margin: 0 auto; }
.loading { padding: 60px 0; color: #999; }
.locked { padding: 34px 0; }
.locked-icon { font-size: 40px; }
.locked-title { font-size: 17px; font-weight: 600; color: #666; margin: 10px 0 6px; }
.locked-text { color: #aaa; font-size: 13px; line-height: 1.8; }
.locked-link { display: inline-block; margin-top: 12px; color: #c85f5f; font-size: 14px; text-decoration: none; }
.who { margin-top: 14px; }
.name { font-size: 17px; font-weight: 600; }
.no { color: #999; font-size: 13px; margin-top: 2px; }
.clock { color: #666; font-size: 14px; margin-top: 8px; font-variant-numeric: tabular-nums; }
.hint { margin-top: 14px; text-align: left; color: #aaa; font-size: 12px; line-height: 1.8; }
</style>

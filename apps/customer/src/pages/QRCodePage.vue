<script setup lang="ts">
import { onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import { api } from '../core/api/endpoints'
import { notify } from '../platform/notify/toast'

const name = ref('')
const memberNo = ref('')
const qrDataUrl = ref('')

onMounted(async () => {
  try {
    const res = await api.myProfile()
    name.value = res.member.name || '未设置昵称'
    memberNo.value = res.member.member_no
    // 核销码内容：固定前缀 + 会员ID，商家端扫码即定位到该会员
    qrDataUrl.value = await QRCode.toDataURL(`ANMO-MEMBER:${res.member.id}`, {
      width: 560,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
  } catch (e) {
    notify((e as Error).message)
  }
})
</script>

<template>
  <div class="page qrcode">
    <div class="card">
      <p class="tip">到店结算时，向商家出示此码即可核销</p>
      <img v-if="qrDataUrl" :src="qrDataUrl" alt="我的核销码" class="qr" />
      <div v-else class="loading">生成中…</div>
      <div class="who">
        <div class="name">{{ name }}</div>
        <div class="no">会员号 {{ memberNo }}</div>
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
.who { margin-top: 14px; }
.name { font-size: 17px; font-weight: 600; }
.no { color: #999; font-size: 13px; margin-top: 2px; }
.hint { margin-top: 14px; text-align: left; color: #aaa; font-size: 12px; line-height: 1.8; }
</style>

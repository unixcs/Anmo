<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import ShopCard from '../components/ShopCard.vue'

// 关于我们（goal §19/§67）：首页文案 + 门店信息卡（导航/拨号）
const heroTitle = ref('')
const heroBody = ref('')
const hours = ref('')
const shop = ref({ address: '', phone: '', latitude: '', longitude: '' })

onMounted(async () => {
  try {
    const s = await api.publicSettings()
    heroTitle.value = s.home_title ?? ''
    heroBody.value = s.home_body ?? ''
    if (s.open_time && s.close_time) hours.value = `${s.open_time} - ${s.close_time}`
    shop.value = {
      address: s.shop_address ?? '',
      phone: s.shop_phone ?? '',
      latitude: s.shop_latitude ?? '',
      longitude: s.shop_longitude ?? '',
    }
  } catch {
    /* 展示页静默回落 */
  }
})
</script>

<template>
  <div class="page about">
    <h1>关于我们</h1>
    <section class="intro">
      <h2>{{ heroTitle || '安摩 · 到店按摩' }}</h2>
      <p>{{ heroBody || '专业肩颈腰背放松，静候您的到来。' }}</p>
      <p v-if="hours" class="hours">🕘 营业时间 {{ hours }}</p>
    </section>
    <ShopCard v-bind="shop" />
  </div>
</template>

<style scoped>
.about { padding: 20px 16px; }
h1 { font-size: 20px; margin-bottom: 14px; }
.intro { background: var(--card, var(--card)); border-radius: 12px; padding: 16px; margin-bottom: 12px; }
.intro h2 { margin: 0 0 8px; font-size: 16px; }
.intro p { margin: 0 0 6px; color: var(--muted-foreground, var(--muted-foreground)); font-size: 14px; line-height: 1.7; }
.hours { color: var(--muted-foreground, var(--muted-foreground)); font-size: 13px; }
</style>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import ShopCard from '../components/ShopCard.vue'
import AppIcon from '../components/ui/AppIcon.vue'

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
    <div class="brand">
      <span class="seal">安摩</span>
      <h1>{{ heroTitle || '安摩 · 到店按摩' }}</h1>
      <p class="body-text">{{ heroBody || '专业肩颈腰背放松，静候您的到来。' }}</p>
    </div>

    <div v-if="hours" class="card plain hours">
      <AppIcon name="clock" :size="16" />
      <span class="num">营业时间 {{ hours }}</span>
    </div>

    <ShopCard v-bind="shop" />
  </div>
</template>

<style scoped>
.brand {
  text-align: center;
  padding: 12px 0 20px;
}

.seal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 14px;
  background: var(--primary);
  color: var(--primary-foreground);
  font: 600 19px/1 var(--font-stack);
  letter-spacing: 3px;
  text-indent: 3px;
  box-shadow: var(--shadow-card);
  margin-bottom: 14px;
}

.brand h1 {
  font: var(--font-title);
  margin: 0 0 8px;
}

.body-text {
  font: var(--font-body);
  color: var(--muted-foreground);
  margin: 0;
  line-height: 1.7;
}

.hours {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-bottom: 10px;
}
</style>

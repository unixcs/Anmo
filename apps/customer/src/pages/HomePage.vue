<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { yuan } from '../core/utils/format'

const services = ref<ServiceItem[]>([])
const phone = ref('13800000000')
const hours = ref('09:00 - 21:00')

onMounted(async () => {
  try {
    const catalog = await api.catalog()
    services.value = catalog.services.slice(0, 6)
  } catch {
    services.value = []
  }
})
</script>

<template>
  <div class="page home">
    <section class="hero">
      <h1>安摩 · 到店按摩</h1>
      <p>专业肩颈腰背放松，静候您的到来</p>
    </section>
    <section class="info">
      <div>📞 {{ phone }}</div>
      <div>🕘 营业时间 {{ hours }}</div>
    </section>
    <section class="section">
      <div class="section-head">
        <h2>服务推荐</h2>
        <RouterLink to="/services" class="more">全部服务 →</RouterLink>
      </div>
      <div class="grid">
        <div v-for="s in services" :key="s.id" class="card">
          <div class="card-name">{{ s.name }}</div>
          <div class="card-meta">{{ s.duration_minutes }} 分钟</div>
          <div class="card-price">¥{{ yuan(s.default_price) }}</div>
        </div>
      </div>
    </section>
    <RouterLink to="/booking" class="cta">立即预约</RouterLink>
  </div>
</template>

<style scoped>
.home { padding: 0 0 20px; }
.hero { background: linear-gradient(135deg, #c85f5f, #a03e3e); color: #fff; padding: 40px 20px; }
.hero h1 { margin: 0 0 6px; font-size: 24px; }
.hero p { margin: 0; opacity: .9; }
.info { display: flex; flex-direction: column; gap: 6px; background: #fff; margin: 12px; padding: 14px; border-radius: 12px; font-size: 14px; color: #555; }
.section { margin: 12px; }
.section-head { display: flex; justify-content: space-between; align-items: baseline; }
.section-head h2 { font-size: 17px; }
.more { color: #c85f5f; font-size: 13px; text-decoration: none; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 10px; }
.card { background: #fff; border-radius: 12px; padding: 14px; }
.card-name { font-weight: 600; }
.card-meta { color: #999; font-size: 13px; margin: 4px 0; }
.card-price { color: #c85f5f; font-weight: 600; }
.cta { display: block; margin: 20px 12px; text-align: center; background: #c85f5f; color: #fff; padding: 14px; border-radius: 12px; text-decoration: none; font-size: 16px; }
</style>

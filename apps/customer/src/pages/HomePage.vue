<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { HomeAnnouncement, HomeBanner, HomeBlock } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { yuan } from '../core/utils/format'

const services = ref<ServiceItem[]>([])
const banners = ref<HomeBanner[]>([])
const announcements = ref<HomeAnnouncement[]>([])
const blocks = ref<HomeBlock[]>([])
const storePhone = ref('')
const hours = ref('09:00 - 21:00')

// 默认布局：商家未配置时沿用（与后端 Home() 默认一致）
const DEFAULT_BLOCKS: HomeBlock[] = [
  { type: 'banner', data: {} },
  { type: 'announcement', data: {} },
  { type: 'service_list', data: {} },
]

function blockText(data: Record<string, unknown>, key: string): string {
  const v = data[key]
  return typeof v === 'string' ? v : ''
}

onMounted(async () => {
  try {
    const [home, catalog, settings] = await Promise.all([
      api.home(),
      api.catalog(),
      api.publicSettings().catch(() => ({})),
    ])
    banners.value = home.banners ?? []
    announcements.value = home.announcements ?? []
    blocks.value = (home.blocks && home.blocks.length > 0 ? home.blocks : DEFAULT_BLOCKS).filter(
      (b) => b.type !== 'banner' || banners.value.length > 0,
    )
    services.value = catalog.services.slice(0, 6)
    storePhone.value = settings.shop_phone ?? ''
    if (settings.open_time && settings.close_time) {
      hours.value = `${settings.open_time} - ${settings.close_time}`
    }
  } catch {
    services.value = []
    blocks.value = DEFAULT_BLOCKS
  }
})
</script>

<template>
  <div class="page home">
    <section class="hero">
      <h1>安摩 · 到店按摩</h1>
      <p>专业肩颈腰背放松，静候您的到来</p>
    </section>

    <template v-for="(b, i) in blocks" :key="i">
      <!-- 轮播图 -->
      <section v-if="b.type === 'banner' && banners.length > 0" class="banners">
        <a v-for="bn in banners" :key="bn.id" class="banner" :href="bn.link || 'javascript:;'">
          <img :src="bn.image" :alt="bn.title" loading="lazy" />
        </a>
      </section>

      <!-- 公告 -->
      <section v-else-if="b.type === 'announcement' && announcements.length > 0" class="notice">
        <span class="notice-tag">公告</span>
        <div class="notice-body">
          <p v-for="a in announcements" :key="a.id">{{ a.title }}<template v-if="a.content"> · {{ a.content }}</template></p>
        </div>
      </section>

      <!-- 服务列表 -->
      <section v-else-if="b.type === 'service_list'" class="section">
        <div class="section-head">
          <h2>{{ blockText(b.data, 'title') || '服务推荐' }}</h2>
          <RouterLink to="/services" class="more">全部服务 →</RouterLink>
        </div>
        <div class="grid">
          <RouterLink v-for="s in services" :key="s.id" :to="`/booking?service=${s.id}`" class="card">
            <div class="card-name">{{ s.name }}</div>
            <div class="card-meta">{{ s.duration_minutes }} 分钟</div>
            <div class="card-price">¥{{ yuan(s.default_price) }}</div>
          </RouterLink>
        </div>
      </section>

      <!-- 活动 -->
      <section v-else-if="b.type === 'activity'" class="activity">
        <h3>{{ blockText(b.data, 'title') || '店内活动' }}</h3>
        <p v-if="blockText(b.data, 'text')">{{ blockText(b.data, 'text') }}</p>
      </section>

      <!-- 富文本 -->
      <section v-else-if="b.type === 'richtext'" class="richtext">
        <h3 v-if="blockText(b.data, 'title')">{{ blockText(b.data, 'title') }}</h3>
        <p v-if="blockText(b.data, 'text') || blockText(b.data, 'content')">
          {{ blockText(b.data, 'text') || blockText(b.data, 'content') }}
        </p>
      </section>
    </template>

    <section class="info">
      <div v-if="storePhone">📞 {{ storePhone }}</div>
      <div>🕘 营业时间 {{ hours }}</div>
    </section>
    <RouterLink to="/booking" class="cta">立即预约</RouterLink>
  </div>
</template>

<style scoped>
.home { padding: 0 0 20px; }
.hero { background: linear-gradient(135deg, #c85f5f, #a03e3e); color: #fff; padding: 40px 20px; }
.hero h1 { margin: 0 0 6px; font-size: 24px; }
.hero p { margin: 0; opacity: .9; }
.banners { display: flex; gap: 10px; overflow-x: auto; margin: 12px 12px 0; padding-bottom: 2px; }
.banner { flex: 0 0 82%; border-radius: 12px; overflow: hidden; background: #eee; }
.banner img { width: 100%; height: 130px; object-fit: cover; display: block; }
.notice { display: flex; align-items: center; gap: 8px; background: #fff; margin: 12px; padding: 10px 14px; border-radius: 12px; }
.notice-tag { flex: 0 0 auto; color: #c85f5f; font-weight: 600; font-size: 13px; }
.notice-body { overflow: hidden; }
.notice-body p { margin: 0; font-size: 13px; color: #666; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.section { margin: 12px; }
.section-head { display: flex; justify-content: space-between; align-items: baseline; }
.section-head h2 { font-size: 17px; }
.more { color: #c85f5f; font-size: 13px; text-decoration: none; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 10px; }
.card { background: #fff; border-radius: 12px; padding: 14px; text-decoration: none; color: inherit; }
.card-name { font-weight: 600; }
.card-meta { color: #999; font-size: 13px; margin: 4px 0; }
.card-price { color: #c85f5f; font-weight: 600; }
.activity { background: #fff; margin: 12px; padding: 14px; border-radius: 12px; }
.activity h3 { margin: 0 0 6px; font-size: 15px; color: #c85f5f; }
.activity p { margin: 0; font-size: 13px; color: #666; }
.richtext { background: #fff; margin: 12px; padding: 14px; border-radius: 12px; }
.richtext h3 { margin: 0 0 6px; font-size: 15px; }
.richtext p { margin: 0; font-size: 13px; color: #666; }
.info { display: flex; flex-direction: column; gap: 6px; background: #fff; margin: 12px; padding: 14px; border-radius: 12px; font-size: 14px; color: #555; }
.cta { display: block; margin: 20px 12px; text-align: center; background: #c85f5f; color: #fff; padding: 14px; border-radius: 12px; text-decoration: none; font-size: 16px; }
</style>

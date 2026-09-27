<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { HomeAnnouncement, HomeBanner, HomeBlock, StoreStatus } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { yuan } from '../core/utils/format'
import ShopCard from '../components/ShopCard.vue'

const services = ref<ServiceItem[]>([])
const banners = ref<HomeBanner[]>([])
const announcements = ref<HomeAnnouncement[]>([])
const blocks = ref<HomeBlock[]>([])
const hours = ref('09:00 - 20:00')
// 首页文案（§31）与门店信息（§18）
const heroTitle = ref('')
const heroBody = ref('')
const shop = ref({ address: '', phone: '', latitude: '', longitude: '' })
// 门店当前状态（§17：动态计算），60s 轮询足够
const store = ref<StoreStatus | null>(null)
let statusTimer: number | undefined

const STATUS_TEXT: Record<string, string> = { FREE: '空闲中', SERVING: '服务中', BUSY: '忙碌中' }
const statusText = ref('')
const statusClass = ref('')

async function loadStatus(): Promise<void> {
  try {
    const s = await api.storeStatus()
    store.value = s
    statusText.value = s.status === 'SERVING' && s.free_at ? `服务中 · 预计 ${s.free_at} 空闲` : (STATUS_TEXT[s.status] ?? '')
    statusClass.value = s.status.toLowerCase()
  } catch {
    statusText.value = ''
  }
}

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
  void loadStatus()
  statusTimer = window.setInterval(loadStatus, 60000)
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
    if (settings.open_time && settings.close_time) {
      hours.value = `${settings.open_time} - ${settings.close_time}`
    }
    // §31 首页文案：后台可编辑，留空回落默认
    heroTitle.value = settings.home_title ?? ''
    heroBody.value = settings.home_body ?? ''
    shop.value = {
      address: settings.shop_address ?? '',
      phone: settings.shop_phone ?? '',
      latitude: settings.shop_latitude ?? '',
      longitude: settings.shop_longitude ?? '',
    }
  } catch {
    services.value = []
    blocks.value = DEFAULT_BLOCKS
  }
})

onUnmounted(() => {
  if (statusTimer !== undefined) window.clearInterval(statusTimer)
})
</script>

<template>
  <div class="page home">
    <section class="hero">
      <div class="hero-top">
        <h1>{{ heroTitle || '安摩 · 到店按摩' }}</h1>
        <span v-if="statusText" class="store-status" :class="statusClass">{{ statusText }}</span>
      </div>
      <p>{{ heroBody || '专业肩颈腰背放松，静候您的到来' }}</p>
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
      <div>🕘 营业时间 {{ hours }}</div>
      <ShopCard v-bind="shop" />
    </section>
    <RouterLink to="/booking" class="cta">立即预约</RouterLink>
  </div>
</template>

<style scoped>
.home { padding: 0 0 20px; }
.hero { background: linear-gradient(135deg, var(--primary), #a03e3e); color: var(--card); padding: 40px 20px; }
.hero-top { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; }
.hero h1 { margin: 0 0 6px; font-size: 24px; }
.hero p { margin: 0; opacity: .9; }
.store-status { background: rgba(255,255,255,.2); border-radius: 999px; padding: 4px 12px; font-size: 13px; white-space: nowrap; }
.store-status.free { background: rgba(103,194,58,.35); }
.store-status.serving { background: rgba(230,162,60,.4); }
.banners { display: flex; gap: 10px; overflow-x: auto; margin: 12px 12px 0; padding-bottom: 2px; }
.banner { flex: 0 0 82%; border-radius: 12px; overflow: hidden; background: var(--border); }
.banner img { width: 100%; height: 130px; object-fit: cover; display: block; }
.notice { display: flex; align-items: center; gap: 8px; background: var(--card); margin: 12px; padding: 10px 14px; border-radius: 12px; }
.notice-tag { flex: 0 0 auto; color: var(--primary); font-weight: 600; font-size: 13px; }
.notice-body { overflow: hidden; }
.notice-body p { margin: 0; font-size: 13px; color: var(--muted-foreground); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.section { margin: 12px; }
.section-head { display: flex; justify-content: space-between; align-items: baseline; }
.section-head h2 { font-size: 17px; }
.more { color: var(--primary); font-size: 13px; text-decoration: none; }
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 10px; }
.card { background: var(--card); border-radius: 12px; padding: 14px; text-decoration: none; color: inherit; }
.card-name { font-weight: 600; }
.card-meta { color: var(--muted-foreground); font-size: 13px; margin: 4px 0; }
.card-price { color: var(--primary); font-weight: 600; }
.activity { background: var(--card); margin: 12px; padding: 14px; border-radius: 12px; }
.activity h3 { margin: 0 0 6px; font-size: 15px; color: var(--primary); }
.activity p { margin: 0; font-size: 13px; color: var(--muted-foreground); }
.richtext { background: var(--card); margin: 12px; padding: 14px; border-radius: 12px; }
.richtext h3 { margin: 0 0 6px; font-size: 15px; }
.richtext p { margin: 0; font-size: 13px; color: var(--muted-foreground); }
.info { display: flex; flex-direction: column; gap: 6px; background: var(--card); margin: 12px; padding: 14px; border-radius: 12px; font-size: 14px; color: var(--muted-foreground); }
.cta { display: block; margin: 20px 12px; text-align: center; background: var(--primary); color: var(--card); padding: 14px; border-radius: 12px; text-decoration: none; font-size: 16px; }
</style>

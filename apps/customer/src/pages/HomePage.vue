<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { HomeAnnouncement, HomeBanner, HomeBlock, StoreStatus } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { yuan } from '../core/utils/format'
import ShopCard from '../components/ShopCard.vue'
import AppIcon from '../components/ui/AppIcon.vue'

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
const STATUS_CLS: Record<string, string> = { FREE: 'free', SERVING: 'serving', BUSY: 'busy' }
const statusText = ref('')
const statusCls = ref('')

async function loadStatus(): Promise<void> {
  try {
    const s = await api.storeStatus()
    store.value = s
    statusText.value = s.status === 'SERVING' && s.free_at ? `服务中 · ${s.free_at} 后空闲` : (STATUS_TEXT[s.status] ?? '')
    statusCls.value = STATUS_CLS[s.status] ?? ''
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
      api.publicSettings().catch(() => ({} as Record<string, string>)),
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

// 脏图/失效图直接从轮播里摘掉，不留破图占位
function onBannerErr(bn: HomeBanner): void {
  banners.value = banners.value.filter((b) => b.id !== bn.id)
}
</script>

<template>
  <div class="home">
    <!-- 顶栏：品牌印 + 状态灯（宣纸底，不做满屏大色块） -->
    <header class="hero">
      <div class="hero-row">
        <div class="hero-brand">
          <span class="seal">安摩</span>
          <div class="hero-text">
            <h1>{{ heroTitle || '安摩 · 到店按摩' }}</h1>
            <p>{{ heroBody || '专业肩颈腰背放松，静候您的到来' }}</p>
          </div>
        </div>
        <span v-if="statusText" class="store-status" :class="statusCls">
          <i class="dot" />{{ statusText }}
        </span>
      </div>
      <RouterLink to="/booking" class="hero-cta btn primary lg block pressable">
        <AppIcon name="calendar" :size="18" />
        立即预约
      </RouterLink>
    </header>

    <div class="page home-body">
      <template v-for="(b, i) in blocks" :key="i">
        <!-- 轮播图 -->
        <section v-if="b.type === 'banner' && banners.length > 0" class="banners">
          <a v-for="bn in banners" :key="bn.id" class="banner pressable" :href="bn.link || 'javascript:;'">
            <img :src="bn.image" :alt="bn.title" loading="lazy" @error="onBannerErr(bn)" />
          </a>
        </section>

        <!-- 公告 -->
        <section v-else-if="b.type === 'announcement' && announcements.length > 0" class="notice card plain">
          <span class="notice-tag">公告</span>
          <div class="notice-body">
            <p v-for="a in announcements" :key="a.id">
              {{ a.title }}<template v-if="a.content"> · {{ a.content }}</template>
            </p>
          </div>
        </section>

        <!-- 服务列表 -->
        <section v-else-if="b.type === 'service_list'" class="section">
          <div class="section-head">
            <h2>{{ blockText(b.data, 'title') || '服务推荐' }}</h2>
            <RouterLink to="/services" class="more">
              全部服务
              <AppIcon name="chevron-right" :size="14" />
            </RouterLink>
          </div>
          <div class="grid">
            <RouterLink
              v-for="s in services"
              :key="s.id"
              :to="`/booking?service=${s.id}`"
              class="card svc-card pressable"
            >
              <div class="svc-name">{{ s.name }}</div>
              <div class="svc-meta">{{ s.duration_minutes }} 分钟</div>
              <div class="svc-price money">¥{{ yuan(s.default_price) }}</div>
            </RouterLink>
          </div>
        </section>

        <!-- 活动 -->
        <section v-else-if="b.type === 'activity'" class="card block-card">
          <h3 class="accent-title">{{ blockText(b.data, 'title') || '店内活动' }}</h3>
          <p v-if="blockText(b.data, 'text')" class="block-text">{{ blockText(b.data, 'text') }}</p>
        </section>

        <!-- 富文本 -->
        <section v-else-if="b.type === 'richtext'" class="card block-card">
          <h3 v-if="blockText(b.data, 'title')" class="block-title">{{ blockText(b.data, 'title') }}</h3>
          <p v-if="blockText(b.data, 'text') || blockText(b.data, 'content')" class="block-text">
            {{ blockText(b.data, 'text') || blockText(b.data, 'content') }}
          </p>
        </section>
      </template>

      <!-- 门店信息 -->
      <section class="shop-section">
        <div class="hours-row card plain">
          <AppIcon name="clock" :size="16" />
          <span>营业时间 {{ hours }}</span>
        </div>
        <ShopCard v-bind="shop" />
      </section>
    </div>
  </div>
</template>

<style scoped>
.home {
  animation: anmo-rise 0.2s var(--ease) both;
}

/* 顶栏 */
.hero {
  padding: 28px 16px 20px;
}

.hero-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.hero-brand {
  display: flex;
  gap: 12px;
  align-items: center;
  min-width: 0;
}

.seal {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: var(--primary);
  color: var(--primary-foreground);
  font: 600 17px/1 var(--font-stack);
  letter-spacing: 2px;
  text-indent: 2px;
  box-shadow: var(--shadow-card);
}

.hero-text h1 {
  font: var(--font-title);
  margin: 0;
}

.hero-text p {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 2px 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 190px;
}

.store-status {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font: var(--font-caption);
  border-radius: var(--radius-full);
  padding: 5px 10px;
  background: var(--muted);
  color: var(--muted-foreground);
}

.store-status .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--muted-foreground);
  animation: anmo-breathe 2s ease-in-out infinite;
}

.store-status.free {
  background: var(--success-soft);
  color: var(--success);
}

.store-status.free .dot {
  background: var(--success);
}

.store-status.serving {
  background: var(--warning-soft);
  color: var(--warning);
}

.store-status.serving .dot {
  background: var(--warning);
}

.store-status.busy {
  background: var(--destructive-soft);
  color: var(--destructive);
}

.store-status.busy .dot {
  background: var(--destructive);
}

.hero-cta {
  margin-top: 16px;
}

/* 内容区 */
.home-body {
  padding-top: 0;
}

.banners {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 2px;
  scrollbar-width: none;
}

.banners::-webkit-scrollbar {
  display: none;
}

.banner {
  flex: 0 0 84%;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--muted);
  box-shadow: var(--shadow-card);
}

.banner img {
  width: 100%;
  height: 140px;
  object-fit: cover;
  display: block;
}

.notice {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 14px;
}

.notice-tag {
  flex: 0 0 auto;
  color: var(--primary);
  font: 600 var(--font-caption);
  background: var(--primary-soft);
  border-radius: var(--radius-full);
  padding: 3px 8px;
}

.notice-body {
  overflow: hidden;
}

.notice-body p {
  margin: 0;
  font: var(--font-sub);
  color: var(--muted-foreground);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.section {
  margin-top: 24px;
}

.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.section-head h2 {
  font: var(--font-title);
  margin: 0;
}

.more {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  color: var(--muted-foreground);
  font: var(--font-sub);
  text-decoration: none;
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.svc-card {
  display: block;
  text-decoration: none;
  color: inherit;
}

.svc-name {
  font-weight: 600;
}

.svc-meta {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 4px 0;
}

.svc-price {
  color: var(--primary);
}

.block-card {
  margin-top: 12px;
}

.block-title {
  font: 600 15px/22px var(--font-stack);
  margin: 0 0 6px;
}

.accent-title {
  font: 600 15px/22px var(--font-stack);
  margin: 0 0 6px;
  color: var(--accent);
}

.block-text {
  margin: 0;
  font: var(--font-sub);
  color: var(--muted-foreground);
  line-height: 1.7;
}

.shop-section {
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.hours-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  font: var(--font-sub);
  color: var(--muted-foreground);
}
</style>

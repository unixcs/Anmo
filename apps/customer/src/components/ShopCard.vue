<script setup lang="ts">
/**
 * ShopCard — 门店信息卡（goal §18-§22）。
 * 设计原则：轻、简单、老人易用——地址点一下唤起系统地图导航，
 * 电话点一下直接拨号；不做任何内嵌地图/地图 SDK/路线规划。
 */
const props = defineProps<{
  address?: string
  phone?: string
  latitude?: string
  longitude?: string
}>()

function openMap(): void {
  if (!props.address) return
  const lng = props.longitude?.trim()
  const lat = props.latitude?.trim()
  // 有经纬度：高德 marker 链接（移动浏览器唤起地图 App）；没有：只展示不可点
  const url = lng && lat
    ? `https://uri.amap.com/marker?position=${lng},${lat}&name=${encodeURIComponent(props.address)}&src=anmo&callnative=1`
    : `https://uri.amap.com/search?keyword=${encodeURIComponent(props.address)}&src=anmo&callnative=1`
  window.location.href = url
}

function callPhone(): void {
  if (props.phone) window.location.href = `tel:${props.phone}`
}
</script>

<template>
  <div class="shop-card">
    <p class="shop-title">📍 门店信息</p>
    <button v-if="address" class="shop-row" @click="openMap">
      <span class="shop-label">地址</span>
      <span class="shop-value">{{ address }}</span>
      <span class="shop-go">导航 ›</span>
    </button>
    <div v-else class="shop-row static">
      <span class="shop-label">地址</span>
      <span class="shop-value muted">到店请提前电话联系</span>
    </div>
    <button v-if="phone" class="shop-row" @click="callPhone">
      <span class="shop-label">电话</span>
      <span class="shop-value">{{ phone }}</span>
      <span class="shop-go">拨打 ›</span>
    </button>
  </div>
</template>

<style scoped>
.shop-card { background: var(--card, #fff); border-radius: 12px; padding: 14px; }
.shop-title { margin: 0 0 8px; font-size: 15px; font-weight: 600; color: var(--foreground, #303133); }
.shop-row { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left; background: none; border: 0; padding: 8px 0; border-top: 1px solid var(--border, #f0f0f0); font-size: 14px; }
.shop-row:first-of-type { border-top: 0; }
.shop-row.static { cursor: default; }
.shop-label { flex: 0 0 auto; color: var(--muted-foreground, #999); font-size: 13px; }
.shop-value { flex: 1; color: var(--foreground, #303133); line-height: 1.5; }
.shop-value.muted { color: var(--muted-foreground, #bbb); }
.shop-go { flex: 0 0 auto; color: var(--primary, #c85f5f); font-size: 13px; }
</style>

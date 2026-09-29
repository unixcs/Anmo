<script setup lang="ts">
/**
 * ShopCard — 门店信息卡（goal §18-§22）。
 * 设计原则：轻、简单、老人易用——地址点一下唤起系统地图导航，
 * 电话点一下直接拨号；不做任何内嵌地图/地图 SDK/路线规划。
 */
import AppIcon from './ui/AppIcon.vue'

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
  <div class="shop-card card">
    <p class="shop-title">门店信息</p>
    <button v-if="address" type="button" class="shop-row" @click="openMap">
      <span class="shop-label"><AppIcon name="map-pin" :size="16" /></span>
      <span class="shop-value">{{ address }}</span>
      <span class="shop-go">导航</span>
    </button>
    <div v-else class="shop-row static">
      <span class="shop-label"><AppIcon name="map-pin" :size="16" /></span>
      <span class="shop-value muted">到店请提前电话联系</span>
    </div>
    <button v-if="phone" type="button" class="shop-row" @click="callPhone">
      <span class="shop-label"><AppIcon name="phone" :size="16" /></span>
      <span class="shop-value num">{{ phone }}</span>
      <span class="shop-go">拨打</span>
    </button>
  </div>
</template>

<style scoped>
.shop-title {
  margin: 0 0 4px;
  font: 600 15px/22px var(--font-stack);
}

.shop-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  text-align: left;
  background: none;
  border: 0;
  padding: 10px 0;
  border-top: 1px solid var(--border);
  font-size: 14px;
  cursor: pointer;
}

.shop-row:active {
  opacity: 0.7;
}

.shop-label {
  flex: 0 0 auto;
  color: var(--muted-foreground);
  display: flex;
}

.shop-value {
  flex: 1;
  line-height: 1.5;
}

.shop-value.muted {
  color: var(--muted-foreground);
}

.shop-go {
  flex: 0 0 auto;
  color: var(--primary);
  font: 500 13px/18px var(--font-stack);
}
</style>

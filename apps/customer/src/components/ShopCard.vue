<script setup lang="ts">
/**
 * ShopCard — 门店信息卡（goal §18-§22）。
 * 设计原则：轻、简单、老人易用——地址点一下唤起系统地图导航，
 * 电话点一下直接拨号；不做任何内嵌地图/地图 SDK/路线规划。
 * V2.2 R3：URI 走 core/utils/amap 统一构造（坐标校验 + search 兜底），
 * 跳转后 best-effort 失败检测：约 3 秒未离开页面 → 卡片内展示兜底条
 * （重试链接 / 复制地址 / 高德搜索页），不依赖 iframe、不做 UA 分支。
 */
import { onUnmounted, ref } from 'vue'
import { amapUri, amapSearchUri } from '../core/utils/amap'
import { notify } from '../platform/notify/toast'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps<{
  address?: string
  phone?: string
  latitude?: string
  longitude?: string
}>()

const navUri = ref('')
const searchUri = ref('')
const navPending = ref(false)
const showFallback = ref(false)
const copied = ref(false)
let navTimer: number | undefined
let copiedTimer: number | undefined

function stopNavWatch(): void {
  if (navTimer !== undefined) {
    window.clearTimeout(navTimer)
    navTimer = undefined
  }
  window.removeEventListener('pagehide', onPageHide)
  window.removeEventListener('visibilitychange', onVisChange)
  navPending.value = false
}

// 跳转成功页面即离开：取消 pending，不展示兜底条
function onPageHide(): void {
  stopNavWatch()
}

function onVisChange(): void {
  if (document.visibilityState === 'hidden') stopNavWatch()
}

function openMap(): void {
  if (!props.address || navPending.value) return
  navUri.value = amapUri(props.address, props.longitude, props.latitude)
  searchUri.value = amapSearchUri(props.address)
  showFallback.value = false
  navPending.value = true
  window.addEventListener('pagehide', onPageHide)
  window.addEventListener('visibilitychange', onVisChange)
  window.location.href = navUri.value
  // 3 秒后仍未离开（个别浏览器拦截/不响应 URI）→ 展示兜底条，纯前端降级
  navTimer = window.setTimeout(() => {
    if (!navPending.value) return
    stopNavWatch()
    showFallback.value = true
  }, 3000)
}

function markCopied(): void {
  copied.value = true
  if (copiedTimer !== undefined) window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => (copied.value = false), 2000)
}

async function copyAddress(): Promise<void> {
  const text = props.address || ''
  try {
    await navigator.clipboard.writeText(text)
    markCopied()
  } catch {
    // clipboard API 不可用（http 页面/老内核）：回落隐藏 textarea + execCommand
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    let ok = false
    try {
      ok = document.execCommand('copy')
    } catch {
      ok = false
    } finally {
      document.body.removeChild(ta)
    }
    if (ok) markCopied()
    else notify('复制失败，请长按地址手动复制')
  }
}

function callPhone(): void {
  if (props.phone) window.location.href = `tel:${props.phone}`
}

onUnmounted(() => {
  stopNavWatch()
  if (copiedTimer !== undefined) window.clearTimeout(copiedTimer)
})
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

    <!-- 跳转失败兜底条（V2.2 R3）：保证用户至少拿到地址 -->
    <div v-if="showFallback" class="nav-fallback">
      <p class="nav-fb-tip">没能打开地图？试试这些方式：</p>
      <div class="nav-fb-ops">
        <a class="nav-fb-link" :href="navUri" target="_blank" rel="noopener">重试打开地图</a>
        <button type="button" class="nav-fb-btn" @click="copyAddress">{{ copied ? '已复制' : '复制地址' }}</button>
        <a class="nav-fb-link" :href="searchUri" target="_blank" rel="noopener">在高德搜索</a>
      </div>
    </div>
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

/* 跳转失败兜底条（V2.2 R3） */
.nav-fallback {
  border-top: 1px dashed var(--border);
  margin-top: 4px;
  padding-top: 10px;
}

.nav-fb-tip {
  margin: 0 0 8px;
  font: var(--font-caption);
  color: var(--muted-foreground);
}

.nav-fb-ops {
  display: flex;
  align-items: center;
  gap: 14px;
}

.nav-fb-link {
  color: var(--primary);
  font: 500 13px/18px var(--font-stack);
  text-decoration: none;
}

.nav-fb-btn {
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: var(--radius-sm);
  padding: 4px 10px;
  font: 500 13px/18px var(--font-stack);
  color: var(--foreground);
  cursor: pointer;
}
</style>

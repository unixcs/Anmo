<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { MemberCard } from '../core/models/models'
import { statusText } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

const cards = ref<MemberCard[]>([])

onMounted(async () => {
  try {
    cards.value = await api.myCards()
  } catch (e) {
    notify((e as Error).message)
  }
})
</script>

<template>
  <div class="page cards">
    <h1>我的会员卡</h1>
    <div v-if="!cards.length" class="empty">还没有会员卡，到店办卡后自动显示在这里</div>
    <div v-for="c in cards" :key="c.id" class="card" :class="{ dim: c.status !== 'ACTIVE' }">
      <div class="row">
        <span class="remaining">{{ c.remaining_count }}</span>
        <span class="unit">/ {{ c.total_count }} 次</span>
        <span class="badge">{{ statusText(c.status) }}</span>
      </div>
      <div class="meta">
        有效期 {{ c.valid_from }} ~ {{ c.valid_until ?? '永久' }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.cards { padding: 20px 16px; }
h1 { font-size: 20px; }
.empty { color: #999; text-align: center; padding: 60px 0; font-size: 14px; }
.card { background: linear-gradient(135deg, #a03e3e, #c85f5f); color: #fff; border-radius: 14px; padding: 20px; margin-bottom: 12px; }
.card.dim { opacity: .55; }
.row { display: flex; align-items: baseline; gap: 6px; }
.remaining { font-size: 34px; font-weight: 700; }
.unit { opacity: .85; }
.badge { margin-left: auto; background: rgba(255,255,255,.25); border-radius: 10px; padding: 3px 10px; font-size: 12px; }
.meta { margin-top: 8px; opacity: .85; font-size: 13px; }
</style>

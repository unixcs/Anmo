<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { MemberCard } from '../core/models/models'
import { statusText } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

interface CardTx {
  id: string
  member_card_id: string
  card_name: string
  type: string
  quantity: number
  before_count: number
  after_count: number
  created_at: string
}

const cards = ref<MemberCard[]>([])
// 使用明细按卡懒加载（§14：无预约的核销流水也必须显示）
const txs = ref<Record<string, CardTx[]>>({})
const expanded = ref<Record<string, boolean>>({})
const loadingTxs = ref('')

const TX_TEXT: Record<string, string> = {
  ISSUE: '开卡',
  REDEEM: '核销扣次',
  ADJUSTMENT: '手工调整',
  REVERSAL: '撤销恢复',
}

function fmtTime(s: string): string {
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  const p = (n: number): string => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// §16 有效期：永久显示"永久"，绝不出现 T00:00:00+08:00
function validity(c: MemberCard): string {
  const until = c.valid_until ? c.valid_until.slice(0, 10) : ''
  if (!until) return '永久'
  return `${c.valid_from.slice(0, 10)} ~ ${until}`
}

async function toggle(c: MemberCard): Promise<void> {
  expanded.value[c.id] = !expanded.value[c.id]
  if (expanded.value[c.id] && !txs.value[c.id]) {
    loadingTxs.value = c.id
    try {
      txs.value[c.id] = await api.myCardTransactions(c.id)
    } catch (e) {
      notify((e as Error).message)
      expanded.value[c.id] = false
    } finally {
      loadingTxs.value = ''
    }
  }
}

const sorted = computed(() => [...cards.value].sort((a, b) => (a.status === 'ACTIVE' ? -1 : 1) - (b.status === 'ACTIVE' ? -1 : 1)))

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
    <div v-for="c in sorted" :key="c.id" class="card" :class="{ dim: c.status !== 'ACTIVE' }">
      <div class="row">
        <span class="remaining">{{ c.remaining_count }}</span>
        <span class="unit">/ {{ c.total_count }} 次</span>
        <span class="badge">{{ statusText(c.status) }}</span>
      </div>
      <!-- §15 卡名动态来自 card_template.name，后台改名这里自动变 -->
      <div class="card-name">{{ c.card_name || '会员卡' }}</div>
      <div class="meta">
        有效期 {{ validity(c) }}
      </div>
      <button class="tx-toggle" @click="toggle(c)">
        {{ expanded[c.id] ? '收起使用明细 ▲' : '查看使用明细 ▼' }}
      </button>
      <div v-if="expanded[c.id]" class="tx-list">
        <div v-if="loadingTxs === c.id" class="tx-empty">加载中…</div>
        <template v-else-if="(txs[c.id] ?? []).length">
          <div v-for="t in txs[c.id]" :key="t.id" class="tx">
            <div class="tx-left">
              <span class="tx-type">{{ TX_TEXT[t.type] ?? t.type }}</span>
              <span class="tx-time">{{ fmtTime(t.created_at) }}</span>
            </div>
            <div class="tx-right">
              <span class="tx-qty" :class="{ neg: t.quantity < 0, pos: t.quantity > 0 }">
                {{ t.quantity > 0 ? '+' : '' }}{{ t.quantity }}
              </span>
              <span class="tx-after">余 {{ t.after_count }}</span>
            </div>
          </div>
        </template>
        <div v-else class="tx-empty">暂无使用记录</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cards { padding: 20px 16px; }
h1 { font-size: 20px; }
.empty { color: var(--muted-foreground); text-align: center; padding: 60px 0; font-size: 14px; }
.card { background: linear-gradient(135deg, #a03e3e, var(--primary)); color: var(--card); border-radius: 14px; padding: 20px; margin-bottom: 12px; }
.card.dim { opacity: .55; }
.row { display: flex; align-items: baseline; gap: 6px; }
.remaining { font-size: 34px; font-weight: 700; }
.unit { opacity: .85; }
.badge { margin-left: auto; background: rgba(255,255,255,.25); border-radius: 10px; padding: 3px 10px; font-size: 12px; }
.card-name { margin-top: 4px; font-size: 15px; font-weight: 600; }
.meta { margin-top: 4px; opacity: .85; font-size: 13px; }
.tx-toggle { margin-top: 10px; background: rgba(255,255,255,.18); color: var(--card); border: 0; border-radius: 8px; padding: 7px 12px; font-size: 13px; }
.tx-list { margin-top: 10px; background: rgba(0,0,0,.14); border-radius: 10px; padding: 6px 12px; }
.tx { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid rgba(255,255,255,.12); }
.tx:last-child { border-bottom: 0; }
.tx-left { display: flex; flex-direction: column; gap: 2px; }
.tx-type { font-size: 14px; font-weight: 600; }
.tx-time { font-size: 12px; opacity: .8; }
.tx-right { display: flex; align-items: baseline; gap: 10px; }
.tx-qty { font-size: 16px; font-weight: 700; }
.tx-qty.neg { color: #ffd9d9; }
.tx-qty.pos { color: #d6ffe3; }
.tx-after { font-size: 12px; opacity: .85; }
.tx-empty { color: rgba(255,255,255,.75); font-size: 13px; text-align: center; padding: 14px 0; }
</style>

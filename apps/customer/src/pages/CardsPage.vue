<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, type CardTx } from '../core/api/endpoints'
import type { MemberCard } from '../core/models/models'
import { notify } from '../platform/notify/toast'
import AppIcon from '../components/ui/AppIcon.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

// 余额不是唯一真相：remaining_count 是缓存，card_transaction 是历史（§§）
const cards = ref<MemberCard[]>([])
const loading = ref(true)
const pickedId = ref('')
const txs = ref<CardTx[]>([])
const txLoading = ref(false)

const picked = computed(() => cards.value.find((c) => c.id === pickedId.value))

// 卡状态徽标映射（warning=有效 / muted=已用完 / destructive=过期、作废）
const CARD_BADGE: Record<string, { variant: string; label: string }> = {
  ACTIVE: { variant: 'success', label: '有效' },
  USED_UP: { variant: 'muted', label: '已用完' },
  EXPIRED: { variant: 'destructive', label: '已过期' },
  CANCELLED: { variant: 'destructive', label: '已作废' },
}

const TX_META: Record<string, { label: string; sign: string; cls: string }> = {
  ISSUE: { label: '开卡', sign: '+', cls: 'in' },
  REDEEM: { label: '核销', sign: '−', cls: 'out' },
  REVERSAL: { label: '撤销核销', sign: '+', cls: 'in' },
  ADJUSTMENT: { label: '调整', sign: '', cls: 'adj' },
}

const cardBadge = computed(() => CARD_BADGE[picked.value?.status ?? ''] ?? { variant: 'muted', label: picked.value?.status ?? '' })

function txMeta(t: CardTx): { label: string; sign: string; cls: string } {
  const m = TX_META[t.type] ?? { label: t.type, sign: '', cls: 'adj' }
  if (t.type === 'ADJUSTMENT') {
    return { ...m, sign: t.quantity >= 0 ? '+' : '−' }
  }
  return m
}

async function pick(c: MemberCard): Promise<void> {
  pickedId.value = c.id
  txs.value = []
  txLoading.value = true
  try {
    txs.value = (await api.myCardTransactions(c.id)) ?? []
  } catch (e) {
    notify((e as Error).message)
  } finally {
    txLoading.value = false
  }
}

function fmtDay(iso: string | null): string {
  return iso ? iso.slice(0, 10) : '长期有效'
}

onMounted(async () => {
  try {
    // http 解包后可能是 null（顾客无卡），归一成数组防崩
    cards.value = (await api.myCards()) ?? []
    if (cards.value.length > 0) await pick(cards.value[0])
  } catch (e) {
    notify((e as Error).message)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page cards">
    <div class="page-head">
      <h1>我的会员卡</h1>
      <p class="sub">结算时出示核销码，由商家按卡扣次</p>
    </div>

    <AppSkeleton v-if="loading" variant="card" />

    <AppEmpty
      v-else-if="!cards.length"
      icon="wallet"
      main="还没有会员卡"
      sub="到店咨询店主即可办理，办卡后出示核销码结算"
    >
      <RouterLink to="/services" class="btn secondary sm">先看看服务项目</RouterLink>
    </AppEmpty>

    <template v-else>
      <!-- 卡片列表 -->
      <div class="my-cards">
        <button
          v-for="(c, i) in cards"
          :key="c.id"
          type="button"
          class="member-card pressable"
          :class="{ picked: c.id === pickedId, dim: c.status !== 'ACTIVE' }"
          @click="pick(c)"
        >
          <div class="mc-top">
            <span class="mc-name">{{ c.card_name || '会员卡' }}</span>
            <span class="badge" :class="CARD_BADGE[c.status]?.variant ?? 'muted'">
              {{ CARD_BADGE[c.status]?.label ?? c.status }}
            </span>
          </div>
          <div class="mc-count num">
            <b>{{ c.remaining_count }}</b>
            <span>/ {{ c.total_count }} 次</span>
          </div>
          <div class="mc-bottom num">
            <span>{{ fmtDay(c.valid_until) }}</span>
            <span v-if="i === 0 && c.status === 'ACTIVE'" class="mc-default">默认</span>
          </div>
        </button>
      </div>

      <!-- 选中卡详情：使用明细（card_transaction 是唯一真相） -->
      <section v-if="picked" class="detail">
        <div class="detail-head">
          <h2>使用明细</h2>
          <span class="badge" :class="cardBadge.variant">{{ cardBadge.label }}</span>
        </div>
        <p class="detail-sub num">
          有效期 {{ fmtDay(picked.valid_until) }}
        </p>

        <AppSkeleton v-if="txLoading" variant="text" :lines="4" />

        <AppEmpty
          v-else-if="!txs.length"
          icon="inbox"
          main="还没有使用记录"
          sub="开卡、核销、调整都会记在这里"
        />

        <div v-else class="card plain tx-list">
          <div v-for="t in txs" :key="t.id" class="tx">
            <div class="tx-icon" :class="txMeta(t).cls">
              <AppIcon
                :name="t.type === 'ISSUE' ? 'ticket' : t.type === 'REDEEM' ? 'scan' : t.type === 'REVERSAL' ? 'history' : 'settings'"
                :size="14"
              />
            </div>
            <div class="tx-body">
              <div class="tx-title">
                {{ txMeta(t).label }}
                <span v-if="t.card_name" class="tx-card">{{ t.card_name }}</span>
              </div>
              <div class="tx-sub num">
                {{ t.created_at.slice(0, 16).replace('T', ' ') }}
                <template v-if="t.remark"> · {{ t.remark }}</template>
              </div>
            </div>
            <div class="tx-qty num" :class="txMeta(t).cls">
              {{ txMeta(t).sign }}{{ Math.abs(t.quantity) }} 次
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
/* 会员卡（黄铜金 accent，BRAND-GUIDELINES §2.1） */
.my-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.member-card {
  position: relative;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--accent-soft), #fdf8ee 55%, var(--accent-soft));
  padding: 16px;
  text-align: left;
  cursor: pointer;
  color: var(--foreground);
  transition: border-color 0.15s var(--ease), box-shadow 0.15s var(--ease), opacity 0.15s;
}

.member-card.picked {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(168, 123, 47, 0.15), var(--shadow-card);
}

.member-card.dim {
  opacity: 0.62;
}

.mc-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mc-name {
  font-weight: 600;
}

.mc-count {
  margin: 10px 0 8px;
}

.mc-count b {
  font-size: 30px;
  line-height: 36px;
  font-weight: 600;
  color: var(--accent);
}

.mc-count span {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-left: 4px;
}

.mc-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font: var(--font-caption);
  color: var(--muted-foreground);
  border-top: 1px dashed rgba(168, 123, 47, 0.3);
  padding-top: 8px;
}

.mc-default {
  background: var(--accent);
  color: #fff;
  border-radius: var(--radius-full);
  padding: 1px 8px;
}

/* 明细 */
.detail {
  margin-top: 24px;
}

.detail-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-head h2 {
  font: var(--font-title);
  margin: 0;
}

.detail-sub {
  font: var(--font-caption);
  color: var(--muted-foreground);
  margin: 2px 0 12px;
}

.tx-list {
  padding: 4px 16px;
}

.tx {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
}

.tx + .tx {
  border-top: 1px solid var(--border);
}

.tx-icon {
  flex: 0 0 30px;
  height: 30px;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
}

.tx-icon.in {
  background: var(--success-soft);
  color: var(--success);
}

.tx-icon.out {
  background: var(--primary-soft);
  color: var(--primary);
}

.tx-icon.adj {
  background: var(--muted);
  color: var(--muted-foreground);
}

.tx-body {
  flex: 1;
  min-width: 0;
}

.tx-title {
  font-weight: 500;
  font-size: 14px;
}

.tx-card {
  font: var(--font-caption);
  color: var(--muted-foreground);
  margin-left: 4px;
}

.tx-sub {
  font: var(--font-caption);
  color: var(--muted-foreground);
  margin-top: 1px;
}

.tx-qty {
  font-weight: 600;
  font-size: 14px;
}

.tx-qty.in {
  color: var(--success);
}

.tx-qty.out {
  color: var(--primary);
}

.tx-qty.adj {
  color: var(--muted-foreground);
}
</style>

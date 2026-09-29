<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, type BookingOptions } from '../core/api/endpoints'
import type { Appointment } from '../core/models/models'
import { todayStr } from '../core/logic/booking'
import { notify } from '../platform/notify/toast'
import AppChip from '../components/ui/AppChip.vue'
import AppStatusBadge from '../components/ui/AppStatusBadge.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppSheet from '../components/ui/AppSheet.vue'
import AppDialog from '../components/ui/AppDialog.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'
import AppIcon from '../components/ui/AppIcon.vue'

interface AptItem extends Appointment {
  serviceNames: string
}

const all = ref<AptItem[]>([])
const loading = ref(true)
const failed = ref(false)
const busyId = ref('')
const filter = ref('ALL')

// 活跃状态优先级：待到店 > 服务中 > 其余（与小程序 appointments compare 同口径）
const ACTIVE_RANK: Record<string, number> = { WAITING: 1, IN_SERVICE: 2 }

const FILTERS = [
  { key: 'ALL', label: '全部' },
  { key: 'WAITING', label: '待到店' },
  { key: 'IN_SERVICE', label: '服务中' },
  { key: 'COMPLETED', label: '已完成' },
  { key: 'CANCELLED', label: '已取消' },
  { key: 'NO_SHOW', label: '未到店' },
]

const list = computed(() =>
  filter.value === 'ALL' ? all.value : all.value.filter((a) => a.status === filter.value),
)

// 取消确认（AppDialog 替代 window.confirm）
const cancelling = ref<AptItem | null>(null)
const cancelBusy = ref(false)

// 改期面板状态
const editing = ref<AptItem | null>(null)
const editDays = ref<{ label: string; value: string }[]>([])
const editDayIdx = ref(0)
const editOptions = ref<BookingOptions | null>(null)
const editPart = ref<'' | 'AM' | 'PM'>('')
const editSlot = ref('')

function timeLabel(a: Appointment): string {
  const d = new Date(a.scheduled_start)
  const day = `${d.getMonth() + 1}月${d.getDate()}日`
  if (a.slot_type === 'HALF_DAY') {
    return `${day} ${a.day_part === 'AM' ? '上午' : '下午'}`
  }
  return `${day} ${a.scheduled_start.slice(11, 16)}`
}

function serviceNamesOf(a: Appointment): string {
  const item = a as Appointment & { services?: { service_name_snapshot: string }[] }
  return (item.services ?? []).map((s) => s.service_name_snapshot).filter(Boolean).join(' · ')
}

async function load(): Promise<void> {
  loading.value = true
  try {
    // http 已解包包络，返回的就是预约数组（bug#5 修复）；列表项内嵌 services 快照
    const arr = await api.myAppointments('')
    all.value = [...(arr ?? [])]
      // 与小程序同口径：活跃预约（待到店/服务中）按时间就近排前，其余按日期倒序
      .sort((a, b) => {
        const ra = ACTIVE_RANK[a.status] ?? 9
        const rb = ACTIVE_RANK[b.status] ?? 9
        if (ra !== rb) return ra - rb
        return ra === 9
          ? b.scheduled_start.localeCompare(a.scheduled_start)
          : a.scheduled_start.localeCompare(b.scheduled_start)
      })
      .map((a) => ({ ...a, serviceNames: serviceNamesOf(a) }))
    failed.value = false
  } catch {
    // 列表加载失败不能渲染成"暂无预约"（顾客会以为预约丢了）：给失败态 + 重试
    failed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)

function canOperate(a: Appointment): boolean {
  return a.status === 'WAITING'
}

async function doCancel(): Promise<void> {
  const a = cancelling.value
  if (!a) return
  cancelBusy.value = true
  try {
    await api.cancelAppointment(a.id)
    notify('已取消')
    cancelling.value = null
    await load()
  } catch (e) {
    notify((e as Error).message)
  } finally {
    cancelBusy.value = false
  }
}

// ---------- 改期面板（日期 → 上午/下午 → 可选具体时间） ----------
function editLabel(d: { label: string; value: string }): string {
  return d.label.slice(0, 5) // "09-28"
}

async function openEdit(a: AptItem): Promise<void> {
  editing.value = a
  editOptions.value = null
  editPart.value = ''
  editSlot.value = ''
  if (editDays.value.length === 0) {
    editDays.value = buildDays()
  }
  editDayIdx.value = 0
  await loadEditOptions()
}

function buildDays(): { label: string; value: string }[] {
  // 北京时间锚定的候选日（与预约页/小程序同口径，设备时区不参与业务日期）
  const t = todayStr()
  const base = Date.UTC(+t.slice(0, 4), +t.slice(5, 7) - 1, +t.slice(8, 10))
  const out: { label: string; value: string }[] = []
  for (let i = 0; i < 14; i++) {
    const value = new Date(base + i * 86_400_000).toISOString().slice(0, 10)
    out.push({ label: value, value })
  }
  return out
}

async function loadEditOptions(): Promise<void> {
  editOptions.value = null
  editPart.value = ''
  editSlot.value = ''
  try {
    editOptions.value = await api.bookingOptions(editDays.value[editDayIdx.value].value)
  } catch (e) {
    notify((e as Error).message)
  }
}

async function pickEditDay(i: number): Promise<void> {
  editDayIdx.value = i
  await loadEditOptions()
}

async function submitEdit(): Promise<void> {
  const a = editing.value
  if (!a || !editPart.value) return
  busyId.value = a.id
  const dateStr = editDays.value[editDayIdx.value].value
  const target = editSlot.value
    ? { start_time: `${dateStr} ${editSlot.value}` }
    : { date: dateStr, day_part: editPart.value }
  try {
    await api.rescheduleAppointment(a.id, target)
    notify('已改期')
    editing.value = null
    await load()
  } catch (e) {
    const err = e as { code?: string; message: string }
    if (err.code === 'APT_SLOT_FULL' || err.code === 'APT_HALFDAY_FULL') {
      notify('这个时间刚被约满，换个时间试试')
      await loadEditOptions()
    } else if (err.code === 'APT_CLOSED') {
      notify('该时段店铺休息，请换一天')
      await loadEditOptions()
    } else {
      notify(err.message)
    }
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <div class="page appts">
    <div class="page-head">
      <h1>我的预约</h1>
      <p class="sub">改期或取消请在到店前操作</p>
    </div>

    <div class="chip-row filters">
      <AppChip
        v-for="f in FILTERS"
        :key="f.key"
        :picked="filter === f.key"
        @click="filter = f.key"
      >
        {{ f.label }}
      </AppChip>
    </div>

    <AppSkeleton v-if="loading" variant="card" />

    <AppEmpty
      v-else-if="failed"
      icon="alert"
      main="网络不可用"
      sub="没能取到预约记录，请稍后再试"
    >
      <AppButton variant="outline" size="sm" @click="load">重新加载</AppButton>
    </AppEmpty>

    <AppEmpty
      v-else-if="!list.length"
      icon="calendar"
      main="暂无预约"
      sub="想放松的时候随时约"
    >
      <RouterLink to="/booking" class="btn secondary sm">去预约</RouterLink>
    </AppEmpty>

    <template v-else>
      <div v-for="a in list" :key="a.id" class="card apt">
        <div class="top">
          <span class="time num">{{ timeLabel(a) }}</span>
          <AppStatusBadge :status="a.status" />
        </div>
        <div v-if="a.serviceNames" class="svc">
          <AppIcon name="sparkles" :size="14" />
          {{ a.serviceNames }}
        </div>
        <div v-if="a.slot_type === 'HALF_DAY'" class="fuzzy-tag">模糊预约 · 具体时间由店主安排</div>
        <div v-if="a.customer_note" class="note">备注：{{ a.customer_note }}</div>
        <div class="no num">{{ a.appointment_no }}</div>
        <div v-if="canOperate(a)" class="ops">
          <AppButton variant="outline" size="sm" :disabled="busyId === a.id" @click="openEdit(a)">
            改期
          </AppButton>
          <AppButton variant="destructive" size="sm" :disabled="busyId === a.id" @click="cancelling = a">
            取消预约
          </AppButton>
        </div>
      </div>
    </template>

    <!-- 取消确认 -->
    <AppDialog
      :open="!!cancelling"
      title="取消这个预约吗？"
      :body="cancelling ? `${timeLabel(cancelling)} · ${cancelling.serviceNames || '预约单'} 将被取消` : ''"
      confirm-text="确定取消"
      danger
      :loading="cancelBusy"
      @close="cancelling = null"
      @confirm="doCancel"
    />

    <!-- 改期选择层 -->
    <AppSheet :open="!!editing" title="改期" @close="editing = null">
      <template v-if="editing">
        <p class="sec">日期</p>
        <div class="day-row">
          <button
            v-for="(d, i) in editDays"
            :key="d.value"
            type="button"
            class="day pressable num"
            :class="{ picked: i === editDayIdx }"
            @click="pickEditDay(i)"
          >{{ editLabel(d) }}</button>
        </div>
        <template v-if="editOptions">
          <p class="sec">上午 / 下午</p>
          <div v-if="!editOptions.open" class="closed card plain">这一天休息</div>
          <div v-else class="part-row">
            <button
              v-if="!editOptions.am.closed"
              type="button"
              class="part pressable"
              :class="{ picked: editPart === 'AM' }"
              @click="editPart = 'AM'; editSlot = ''"
            >
              上午 <small class="num">剩 {{ editOptions.am.remaining }}</small>
            </button>
            <button
              v-if="!editOptions.pm.closed"
              type="button"
              class="part pressable"
              :class="{ picked: editPart === 'PM' }"
              @click="editPart = 'PM'; editSlot = ''"
            >
              下午 <small class="num">剩 {{ editOptions.pm.remaining }}</small>
            </button>
          </div>
          <template v-if="editPart && (editPart === 'AM' ? editOptions.am : editOptions.pm)">
            <p class="sec">具体时间（可不选）</p>
            <div class="slot-grid">
              <button
                v-for="s in (editPart === 'AM' ? editOptions.am.slots : editOptions.pm.slots) ?? []"
                :key="s.time"
                type="button"
                class="slot pressable num"
                :class="{ picked: s.time === editSlot, full: s.remaining < 1 }"
                :disabled="s.remaining < 1"
                @click="editSlot = s.time"
              >{{ s.time }}</button>
            </div>
          </template>
        </template>
        <AppButton
          block
          size="lg"
          class="sheet-submit"
          :disabled="!editPart || busyId === editing.id"
          :loading="busyId === editing.id"
          @click="submitEdit"
        >
          {{
            editPart
              ? `改到 ${editDays[editDayIdx].label}${editSlot ? ' ' + editSlot : editPart === 'AM' ? ' 上午' : ' 下午'}`
              : '请选择时段'
          }}
        </AppButton>
      </template>
    </AppSheet>
  </div>
</template>

<style scoped>
.filters {
  margin-bottom: 16px;
}

.apt {
  margin-bottom: 10px;
}

.top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.time {
  font-weight: 600;
  font-size: 16px;
}

.svc {
  display: flex;
  align-items: center;
  gap: 6px;
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-top: 6px;
}

.svc svg {
  color: var(--accent);
}

.fuzzy-tag {
  display: inline-block;
  margin-top: 6px;
  font: var(--font-caption);
  color: var(--accent);
  background: var(--accent-soft);
  border-radius: var(--radius-full);
  padding: 2px 8px;
}

.note {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin-top: 6px;
}

.no {
  font: var(--font-caption);
  color: var(--muted-foreground);
  margin-top: 6px;
}

.ops {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  justify-content: flex-end;
}

/* 改期 sheet 内部 */
.sec {
  font: 500 13px/18px var(--font-stack);
  color: var(--muted-foreground);
  margin: 14px 0 8px;
}

.day-row {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  padding-bottom: 4px;
  scrollbar-width: none;
}

.day-row::-webkit-scrollbar {
  display: none;
}

.day {
  flex: 0 0 auto;
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  font: 500 12px/16px var(--font-stack);
  color: var(--muted-foreground);
  cursor: pointer;
  transition: all 0.15s var(--ease);
}

.day.picked {
  border-color: var(--primary);
  background: var(--primary-soft);
  color: var(--primary);
  font-weight: 600;
}

.part-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.part {
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: var(--radius-md);
  padding: 10px 0;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  transition: all 0.15s var(--ease);
}

.part small {
  color: var(--muted-foreground);
  font: var(--font-caption);
}

.part.picked {
  border-color: var(--primary);
  background: var(--primary-soft);
  color: var(--primary);
}

.part.picked small {
  color: var(--primary);
}

.slot-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.slot {
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: var(--radius-sm);
  padding: 9px 0;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s var(--ease);
}

.slot.full {
  opacity: 0.4;
  cursor: not-allowed;
}

.slot.picked {
  border-color: var(--primary);
  background: var(--primary);
  color: var(--primary-foreground);
}

.closed {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  font: var(--font-sub);
  color: var(--muted-foreground);
}

.sheet-submit {
  margin-top: 16px;
}
</style>

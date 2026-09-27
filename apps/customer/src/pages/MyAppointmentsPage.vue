<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type BookingOptions } from '../core/api/endpoints'
import type { Appointment } from '../core/models/models'
import { statusText } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

const list = ref<Appointment[]>([])
const busyId = ref('')

// 改期面板状态
const editing = ref<Appointment | null>(null)
const editDays = ref<{ label: string; value: string }[]>([])
const editDayIdx = ref(0)
const editOptions = ref<BookingOptions | null>(null)
const editPart = ref<'' | 'AM' | 'PM'>('')
const editSlot = ref('')

const MONTHS = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10', '11', '12']

function timeLabel(a: Appointment): string {
  const d = new Date(a.scheduled_start)
  const day = `${MONTHS[d.getMonth()]}月${d.getDate()}日`
  if (a.slot_type === 'HALF_DAY') {
    return `${day} ${a.day_part === 'AM' ? '上午' : '下午'}`
  }
  return `${day} ${a.scheduled_start.slice(11, 16)}`
}

async function load(): Promise<void> {
  try {
    // http 已解包包络，返回的就是预约数组（bug#5 修复）
    const arr = await api.myAppointments('')
    list.value = [...(arr ?? [])].sort((a, b) => b.scheduled_start.localeCompare(a.scheduled_start))
  } catch (e) {
    notify((e as Error).message)
  }
}

onMounted(load)

function canCancel(a: Appointment): boolean {
  return a.status === 'PENDING_CONFIRM' || a.status === 'CONFIRMED'
}

async function cancel(a: Appointment): Promise<void> {
  if (!window.confirm('确定取消这个预约吗？')) return
  busyId.value = a.id
  try {
    await api.cancelAppointment(a.id)
    notify('已取消')
    await load()
  } catch (e) {
    notify((e as Error).message)
  } finally {
    busyId.value = ''
  }
}

// ---------- 改期面板（日期 → 上午/下午 → 可选具体时间） ----------
function editLabel(d: { label: string; value: string }): string {
  return d.label.slice(0, 5) // "09-28"
}

async function openEdit(a: Appointment): Promise<void> {
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
  const out: { label: string; value: string }[] = []
  const now = new Date()
  for (let i = 0; i < 14; i++) {
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() + i)
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    out.push({ label: `${mm}-${dd}`, value: `${d.getFullYear()}-${mm}-${dd}` })
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
    <h1>我的预约</h1>
    <div v-if="!list.length" class="empty">暂无预约</div>
    <div v-for="a in list" :key="a.id" class="item">
      <div class="top">
        <span class="time">{{ timeLabel(a) }}</span>
        <span class="badge" :data-status="a.status">{{ statusText(a.status) }}</span>
      </div>
      <div class="no">{{ a.appointment_no }}</div>
      <div v-if="a.slot_type === 'HALF_DAY'" class="fuzzy-tag">模糊预约 · 具体时间由店家安排</div>
      <div v-if="a.customer_note" class="note">备注：{{ a.customer_note }}</div>
      <div v-if="canCancel(a)" class="ops">
        <button :disabled="busyId === a.id" @click="openEdit(a)">改期</button>
        <button class="danger" :disabled="busyId === a.id" @click="cancel(a)">取消预约</button>
      </div>
    </div>

    <!-- 改期选择层 -->
    <div v-if="editing" class="mask" @click.self="editing = null">
      <div class="sheet">
        <div class="sheet-head">
          <b>改期</b>
          <button class="close" @click="editing = null">✕</button>
        </div>
        <p class="sec">日期</p>
        <div class="day-row">
          <button
            v-for="(d, i) in editDays"
            :key="d.value"
            class="day"
            :class="{ picked: i === editDayIdx }"
            @click="pickEditDay(i)"
          >{{ editLabel(d) }}</button>
        </div>
        <template v-if="editOptions">
          <p class="sec">上午 / 下午</p>
          <div v-if="!editOptions.open" class="empty">这一天休息</div>
          <div v-else class="part-row">
            <button v-if="!editOptions.am.closed" class="part" :class="{ picked: editPart === 'AM' }" @click="editPart = 'AM'; editSlot = ''">
              上午 <small>剩 {{ editOptions.am.remaining }}</small>
            </button>
            <button v-if="!editOptions.pm.closed" class="part" :class="{ picked: editPart === 'PM' }" @click="editPart = 'PM'; editSlot = ''">
              下午 <small>剩 {{ editOptions.pm.remaining }}</small>
            </button>
          </div>
          <template v-if="editPart && (editPart === 'AM' ? editOptions.am : editOptions.pm)">
            <p class="sec">具体时间（可不选）</p>
            <div class="slot-grid">
              <button
                v-for="s in (editPart === 'AM' ? editOptions.am.slots : editOptions.pm.slots) ?? []"
                :key="s.time"
                class="slot"
                :class="{ picked: s.time === editSlot, full: s.remaining < 1 }"
                :disabled="s.remaining < 1"
                @click="editSlot = s.time"
              >{{ s.time }}</button>
            </div>
          </template>
        </template>
        <button class="primary" :disabled="busyId === editing.id || !editPart" @click="submitEdit">
          {{ editPart ? `改到 ${editDays[editDayIdx].label}${editSlot ? ' ' + editSlot : editPart === 'AM' ? ' 上午' : ' 下午'}` : '请选择时段' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.appts { padding: 20px 16px; }
h1 { font-size: 20px; }
.empty { color: #999; text-align: center; padding: 60px 0; }
.item { background: #fff; border-radius: 12px; padding: 14px; margin-bottom: 10px; }
.top { display: flex; justify-content: space-between; align-items: center; }
.time { font-weight: 600; }
.badge { font-size: 12px; padding: 3px 10px; border-radius: 10px; background: #f2f2f2; color: #666; }
.badge[data-status='COMPLETED'] { background: #e8f6e8; color: #3a8f3a; }
.badge[data-status='CANCELLED'], .badge[data-status='NO_SHOW'] { background: #fdeaea; color: #c85f5f; }
.no { color: #bbb; font-size: 12px; margin-top: 4px; }
.fuzzy-tag { display: inline-block; margin-top: 6px; font-size: 11px; color: #b8860b; background: #fdf6e3; border-radius: 8px; padding: 2px 8px; }
.note { color: #888; font-size: 13px; margin-top: 6px; }
.ops { display: flex; gap: 8px; margin-top: 10px; justify-content: flex-end; }
.ops button { border: 1px solid #ddd; background: #fff; border-radius: 8px; padding: 7px 14px; font-size: 13px; }
.ops .danger { color: #c85f5f; border-color: #c85f5f; }
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); display: flex; align-items: flex-end; justify-content: center; z-index: 30; }
.sheet { background: #fff; border-radius: 16px 16px 0 0; padding: 16px; width: 100%; max-width: 420px; max-height: 80vh; overflow-y: auto; }
.sheet-head { display: flex; justify-content: space-between; align-items: center; }
.close { border: none; background: none; font-size: 16px; color: #999; }
.sec { font-size: 13px; color: #666; margin: 12px 0 6px; }
.day-row { display: flex; gap: 6px; overflow-x: auto; padding-bottom: 4px; }
.day { flex: 0 0 auto; border: 1px solid #eee; background: #fff; border-radius: 8px; padding: 7px 9px; font-size: 12px; }
.day.picked { border-color: #c85f5f; background: #fdf3f3; color: #c85f5f; }
.part-row { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.part { border: 1px solid #eee; background: #fff; border-radius: 10px; padding: 10px 0; font-size: 14px; }
.part small { color: #999; font-size: 11px; }
.part.picked { border-color: #c85f5f; background: #fdf3f3; color: #c85f5f; }
.part.picked small { color: #c85f5f; }
.slot-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; }
.slot { border: 1px solid #eee; background: #fff; border-radius: 8px; padding: 8px 0; font-size: 13px; }
.slot.full { opacity: .4; }
.slot.picked { border-color: #c85f5f; background: #c85f5f; color: #fff; }
.primary { width: 100%; height: 44px; background: #c85f5f; color: #fff; border: none; border-radius: 12px; font-size: 15px; margin-top: 14px; }
.primary:disabled { opacity: .5; }
</style>

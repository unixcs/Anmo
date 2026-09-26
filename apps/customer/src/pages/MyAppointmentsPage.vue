<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { Appointment } from '../core/models/models'
import { statusText } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

const list = ref<Appointment[]>([])
const busyId = ref('')

async function load(): Promise<void> {
  try {
    const page = await api.myAppointments('')
    list.value = [...(page.data ?? [])].sort((a, b) => b.scheduled_start.localeCompare(a.scheduled_start))
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

function pickNewTime(a: Appointment): void {
  const input = window.prompt('改期：输入新的开始时间（格式 YYYY-MM-DD HH:MM）', a.scheduled_start.slice(0, 16).replace('T', ' '))
  if (!input) return
  const rescheduled = reschedule(a, input)
  void rescheduled
}

async function reschedule(a: Appointment, startTime: string): Promise<void> {
  busyId.value = a.id
  try {
    await api.rescheduleAppointment(a.id, startTime)
    notify('已改期')
    await load()
  } catch (e) {
    notify((e as Error).message)
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
        <span class="time">{{ a.scheduled_start.slice(0, 16).replace('T', ' ') }}</span>
        <span class="badge" :data-status="a.status">{{ statusText(a.status) }}</span>
      </div>
      <div class="no">{{ a.appointment_no }}</div>
      <div v-if="a.customer_note" class="note">备注：{{ a.customer_note }}</div>
      <div v-if="canCancel(a)" class="ops">
        <button :disabled="busyId === a.id" @click="pickNewTime(a)">改期</button>
        <button class="danger" :disabled="busyId === a.id" @click="cancel(a)">取消预约</button>
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
.note { color: #888; font-size: 13px; margin-top: 6px; }
.ops { display: flex; gap: 8px; margin-top: 10px; justify-content: flex-end; }
.ops button { border: 1px solid #ddd; background: #fff; border-radius: 8px; padding: 7px 14px; font-size: 13px; }
.ops .danger { color: #c85f5f; border-color: #c85f5f; }
</style>

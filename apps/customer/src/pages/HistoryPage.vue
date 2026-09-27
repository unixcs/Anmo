<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../core/api/endpoints'
import type { Appointment } from '../core/models/models'
import { statusText } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

const list = ref<Appointment[]>([])

onMounted(async () => {
  try {
    const arr = await api.myAppointments('COMPLETED')
    list.value = [...(arr ?? [])].sort((a, b) => b.scheduled_start.localeCompare(a.scheduled_start))
  } catch (e) {
    notify((e as Error).message)
  }
})
</script>

<template>
  <div class="page history">
    <h1>历史服务</h1>
    <div v-if="!list.length" class="empty">还没有已完成的服务记录</div>
    <div v-for="a in list" :key="a.id" class="item">
      <div class="time">{{ a.scheduled_start.slice(0, 16).replace('T', ' ') }}</div>
      <div class="no">{{ a.appointment_no }}</div>
      <span class="badge">{{ statusText(a.status) }}</span>
    </div>
  </div>
</template>

<style scoped>
.history { padding: 20px 16px; }
h1 { font-size: 20px; }
.empty { color: var(--muted-foreground); text-align: center; padding: 60px 0; }
.item { background: var(--card); border-radius: 12px; padding: 14px; margin-bottom: 10px; }
.time { font-weight: 600; }
.no { color: var(--muted-foreground); font-size: 12px; margin: 4px 0; }
.badge { font-size: 12px; padding: 3px 10px; border-radius: 10px; background: #e8f6e8; color: #3a8f3a; }
</style>

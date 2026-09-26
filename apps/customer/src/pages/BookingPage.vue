<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { candidateDays, slotsForDay, DEFAULT_HOURS, type DayOption, type SlotOption } from '../core/logic/booking'
import { yuan } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

const route = useRoute()
const router = useRouter()

const services = ref<ServiceItem[]>([])
const serviceId = ref('')
const day = ref<DayOption[]>([])
const dayIdx = ref(0)
const slots = ref<SlotOption[]>([])
const slotIdx = ref(-1)
const note = ref('')
const busy = ref(false)

const selectedService = computed(() => services.value.find((s) => s.id === serviceId.value))
const selectedSlot = computed(() => slots.value[slotIdx.value])

onMounted(async () => {
  const catalog = await api.catalog()
  services.value = catalog.services
  const preset = route.query.service as string | undefined
  if (preset && services.value.some((s) => s.id === preset)) serviceId.value = preset
  day.value = candidateDays(DEFAULT_HOURS)
  refreshSlots()
})

function refreshSlots(): void {
  slotIdx.value = -1
  if (!selectedService.value) {
    slots.value = []
    return
  }
  slots.value = slotsForDay(day.value[dayIdx.value].value, selectedService.value.duration_minutes, DEFAULT_HOURS)
}

function pickDay(i: number): void {
  dayIdx.value = i
  refreshSlots()
}

function pickSlot(i: number): void {
  slotIdx.value = i
}

async function submit(): Promise<void> {
  if (!selectedService.value || !selectedSlot.value) {
    notify('请先选择服务和时间')
    return
  }
  busy.value = true
  try {
    await api.createAppointment(selectedService.value.id, selectedSlot.value.startTime, note.value)
    notify('预约提交成功，等待店家确认')
    router.push('/me/appointments')
  } catch (e) {
    const err = e as { code?: string; message: string }
    if (err.code === 'APPOINTMENT_CONFLICT') {
      notify('这个时间刚刚被预约，请重新选择')
      refreshSlots()
    } else {
      notify(err.message)
    }
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="page booking">
    <h1>预约</h1>

    <section class="block">
      <h2>1 · 选择服务</h2>
      <div class="svc-list">
        <button
          v-for="s in services"
          :key="s.id"
          class="svc"
          :class="{ picked: s.id === serviceId }"
          @click="serviceId = s.id; refreshSlots()"
        >
          <span>{{ s.name }}</span>
          <span class="svc-meta">{{ s.duration_minutes }}分钟 ¥{{ yuan(s.default_price) }}</span>
        </button>
      </div>
    </section>

    <section v-if="serviceId" class="block">
      <h2>2 · 选择日期</h2>
      <div class="day-row">
        <button
          v-for="(d, i) in day.slice(0, 14)"
          :key="d.value"
          class="day"
          :class="{ picked: i === dayIdx }"
          @click="pickDay(i)"
        >{{ d.label }}</button>
      </div>
      <h2>3 · 选择时间</h2>
      <div v-if="slots.length" class="slot-grid">
        <button
          v-for="(s, i) in slots"
          :key="s.startTime"
          class="slot"
          :class="{ picked: i === slotIdx }"
          @click="pickSlot(i)"
        >{{ s.label }}</button>
      </div>
      <p v-else class="empty">这一天没有可用时间了，看看别的日期吧</p>
    </section>

    <section v-if="serviceId" class="block">
      <h2>备注（可选）</h2>
      <textarea v-model="note" rows="2" placeholder="身体状况、偏好等" />
    </section>

    <button class="primary" :disabled="busy || slotIdx < 0" @click="submit">
      提交预约{{ selectedService && selectedSlot ? `：${selectedService.name} ${selectedSlot.label}` : '' }}
    </button>
  </div>
</template>

<style scoped>
.booking { padding: 20px 16px; }
h1 { font-size: 20px; }
.block { margin: 18px 0; }
.block h2 { font-size: 14px; color: #666; margin: 10px 0; }
.svc-list { display: flex; flex-wrap: wrap; gap: 8px; }
.svc { border: 1px solid #eee; background: #fff; border-radius: 10px; padding: 10px 12px; display: flex; flex-direction: column; align-items: flex-start; gap: 4px; }
.svc.picked { border-color: #c85f5f; background: #fdf3f3; }
.svc-meta { color: #c85f5f; font-size: 13px; }
.day-row { display: flex; gap: 8px; overflow-x: auto; padding-bottom: 4px; }
.day { flex: 0 0 auto; border: 1px solid #eee; background: #fff; border-radius: 8px; padding: 8px 10px; font-size: 13px; }
.day.picked { border-color: #c85f5f; background: #fdf3f3; color: #c85f5f; }
.slot-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.slot { border: 1px solid #eee; background: #fff; border-radius: 8px; padding: 10px 0; }
.slot.picked { border-color: #c85f5f; background: #c85f5f; color: #fff; }
.empty { color: #999; font-size: 14px; }
textarea { width: 100%; border: 1px solid #ddd; border-radius: 8px; padding: 10px; font-size: 14px; box-sizing: border-box; }
.primary { width: 100%; height: 48px; background: #c85f5f; color: #fff; border: none; border-radius: 12px; font-size: 16px; margin-top: 10px; }
.primary:disabled { opacity: .5; }
</style>

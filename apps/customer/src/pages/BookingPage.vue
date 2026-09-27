<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type BookingHalfDay, type BookingOptions } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { candidateDays, type DayOption } from '../core/logic/booking'
import { yuan } from '../core/utils/format'
import { notify } from '../platform/notify/toast'

const route = useRoute()
const router = useRouter()

const services = ref<ServiceItem[]>([])
const serviceId = ref('')
const day = ref<DayOption[]>([])
const dayIdx = ref(0)
const options = ref<BookingOptions | null>(null)
const part = ref<'' | 'AM' | 'PM'>('')
const slotTime = ref('')
const note = ref('')
const busy = ref(false)
const loadingOptions = ref(false)

const selectedService = computed(() => services.value.find((s) => s.id === serviceId.value))
const half = computed<BookingHalfDay | null>(() => {
  if (!options.value || !part.value) return null
  return part.value === 'AM' ? options.value.am : options.value.pm
})
const submitLabel = computed(() => {
  if (!selectedService.value || !part.value) return ''
  const t = slotTime.value ? ` ${slotTime.value}` : ` ${part.value === 'AM' ? '上午' : '下午'}`
  return `${selectedService.value.name} · ${day.value[dayIdx.value]?.label.slice(0, 5)}${t}`
})

onMounted(async () => {
  const catalog = await api.catalog()
  services.value = catalog.services
  const preset = route.query.service as string | undefined
  if (preset && services.value.some((s) => s.id === preset)) serviceId.value = preset
  day.value = candidateDays()
  if (serviceId.value) await refreshOptions()
})

async function refreshOptions(): Promise<void> {
  options.value = null
  part.value = ''
  slotTime.value = ''
  if (!serviceId.value) return
  loadingOptions.value = true
  try {
    options.value = await api.bookingOptions(day.value[dayIdx.value].value)
  } catch (e) {
    notify((e as Error).message)
  } finally {
    loadingOptions.value = false
  }
}

function pickDay(i: number): void {
  dayIdx.value = i
  void refreshOptions()
}

function pickPart(p: 'AM' | 'PM'): void {
  part.value = p
  slotTime.value = ''
}

async function submit(): Promise<void> {
  if (!selectedService.value || !part.value) {
    notify('请先选择服务、日期和上午/下午')
    return
  }
  busy.value = true
  const dateStr = day.value[dayIdx.value].value
  const target = slotTime.value
    ? { start_time: `${dateStr} ${slotTime.value}` }
    : { date: dateStr, day_part: part.value }
  try {
    await api.createAppointment(selectedService.value.id, target, note.value)
    if (!slotTime.value) {
      notify('预约成功：具体时间由店主安排，可能需要等待')
    } else {
      notify('预约提交成功，等待店家确认')
    }
    router.push('/me/appointments')
  } catch (e) {
    const err = e as { code?: string; message: string }
    if (err.code === 'APT_SLOT_FULL' || err.code === 'APT_HALFDAY_FULL') {
      notify('这个时间刚被约满，换个时间试试')
      await refreshOptions()
    } else if (err.code === 'APT_CLOSED') {
      notify('该时段店铺休息，请换一天')
      await refreshOptions()
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
          @click="serviceId = s.id; refreshOptions()"
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

      <h2>3 · 上午 / 下午（必选）</h2>
      <div v-if="loadingOptions" class="empty">加载中…</div>
      <template v-else-if="options">
        <div v-if="!options.open" class="empty">这一天休息，看看别的日期吧</div>
        <div v-else class="part-row">
          <button v-if="!options.am.closed" class="part" :class="{ picked: part === 'AM' }" @click="pickPart('AM')">
            <b>上午</b>
            <span class="part-meta">剩 {{ options.am.remaining }} 个名额</span>
          </button>
          <button v-if="!options.pm.closed" class="part" :class="{ picked: part === 'PM' }" @click="pickPart('PM')">
            <b>下午</b>
            <span class="part-meta">剩 {{ options.pm.remaining }} 个名额</span>
          </button>
        </div>

        <template v-if="half">
          <h2>4 · 具体时间（可不选，由店家安排）</h2>
          <div class="slot-grid">
            <button
              v-for="s in half.slots ?? []"
              :key="s.time"
              class="slot"
              :class="{ picked: s.time === slotTime, full: s.remaining < 1 }"
              :disabled="s.remaining < 1"
              @click="slotTime = s.time"
            >{{ s.time }}<small v-if="s.remaining < 1">满</small></button>
          </div>
        </template>
      </template>
    </section>

    <section v-if="serviceId" class="block">
      <h2>备注（可选）</h2>
      <textarea v-model="note" rows="2" placeholder="身体状况、偏好等" />
    </section>

    <button class="primary" :disabled="busy || !part" @click="submit">
      {{ part ? `提交预约：${submitLabel}` : '请先选择上午 / 下午' }}
    </button>
    <p class="fuzzy-hint">只选上午/下午提交 = 模糊预约，具体时间由店主安排，可能需要等待。</p>
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
.part-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.part { border: 1px solid #eee; background: #fff; border-radius: 12px; padding: 16px 0; display: flex; flex-direction: column; gap: 4px; align-items: center; }
.part b { font-size: 17px; }
.part.picked { border-color: #c85f5f; background: #fdf3f3; color: #c85f5f; }
.part-meta { color: #999; font-size: 12px; }
.part.picked .part-meta { color: #c85f5f; }
.slot-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.slot { border: 1px solid #eee; background: #fff; border-radius: 8px; padding: 10px 0; display: flex; flex-direction: column; align-items: center; }
.slot small { color: #c85f5f; font-size: 11px; }
.slot.full { opacity: .4; }
.slot.picked { border-color: #c85f5f; background: #c85f5f; color: #fff; }
.slot.picked small { color: #fff; }
.empty { color: #999; font-size: 14px; }
textarea { width: 100%; border: 1px solid #ddd; border-radius: 8px; padding: 10px; font-size: 14px; box-sizing: border-box; }
.primary { width: 100%; height: 48px; background: #c85f5f; color: #fff; border: none; border-radius: 12px; font-size: 16px; margin-top: 10px; }
.primary:disabled { opacity: .5; }
.fuzzy-hint { color: #aaa; font-size: 12px; text-align: center; margin-top: 8px; }
</style>

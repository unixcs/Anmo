<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api, type BookingHalfDay, type BookingOptions } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { candidateDays, type DayOption } from '../core/logic/booking'
import { yuan } from '../core/utils/format'
import { notify } from '../platform/notify/toast'
import ShopCard from '../components/ShopCard.vue'

const route = useRoute()

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

// 预约成功页（goal §19/§66）：页内成功态，展示服务/日期/时间 + 门店信息卡
const success = ref<{ name: string; date: string; time: string; fuzzy: boolean } | null>(null)
const shop = ref({ address: '', phone: '', latitude: '', longitude: '' })

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
    success.value = {
      name: selectedService.value.name,
      date: day.value[dayIdx.value].label,
      time: slotTime.value || (part.value === 'AM' ? '上午' : '下午'),
      fuzzy: !slotTime.value,
    }
    try {
      const s = await api.publicSettings()
      shop.value = {
        address: s.shop_address ?? '',
        phone: s.shop_phone ?? '',
        latitude: s.shop_latitude ?? '',
        longitude: s.shop_longitude ?? '',
      }
    } catch {
      /* 门店信息拿不到不影响成功页 */
    }
    window.scrollTo({ top: 0 })
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
    <!-- 预约成功页（§19：服务/日期/时间 + 门店信息卡，地址导航电话拨号） -->
    <section v-if="success" class="done">
      <div class="done-icon">✅</div>
      <h1>预约成功</h1>
      <div class="done-card">
        <div class="done-row"><span>服务项目</span><b>{{ success.name }}</b></div>
        <div class="done-row"><span>日期</span><b>{{ success.date }}</b></div>
        <div class="done-row"><span>时间</span><b>{{ success.time }}<template v-if="success.fuzzy">（具体时间由店主安排）</template></b></div>
      </div>
      <p class="done-tip">到店后向商家出示「我的 → 核销码 / 预约单码」即可</p>
      <ShopCard v-bind="shop" />
      <div class="done-btns">
        <RouterLink to="/me/appointments" class="primary">查看我的预约</RouterLink>
        <RouterLink to="/" class="ghost">返回首页</RouterLink>
      </div>
    </section>

    <template v-else>
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
    </template>
  </div>
</template>

<style scoped>
.booking { padding: 20px 16px; }
h1 { font-size: 20px; }
.block { margin: 18px 0; }
.block h2 { font-size: 14px; color: var(--muted-foreground); margin: 10px 0; }
.svc-list { display: flex; flex-wrap: wrap; gap: 8px; }
.svc { border: 1px solid var(--border); background: var(--card); border-radius: 10px; padding: 10px 12px; display: flex; flex-direction: column; align-items: flex-start; gap: 4px; }
.svc.picked { border-color: var(--primary); background: var(--primary-muted); }
.svc-meta { color: var(--primary); font-size: 13px; }
.day-row { display: flex; gap: 8px; overflow-x: auto; padding-bottom: 4px; }
.day { flex: 0 0 auto; border: 1px solid var(--border); background: var(--card); border-radius: 8px; padding: 8px 10px; font-size: 13px; }
.day.picked { border-color: var(--primary); background: var(--primary-muted); color: var(--primary); }
.part-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.part { border: 1px solid var(--border); background: var(--card); border-radius: 12px; padding: 16px 0; display: flex; flex-direction: column; gap: 4px; align-items: center; }
.part b { font-size: 17px; }
.part.picked { border-color: var(--primary); background: var(--primary-muted); color: var(--primary); }
.part-meta { color: var(--muted-foreground); font-size: 12px; }
.part.picked .part-meta { color: var(--primary); }
.slot-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.slot { border: 1px solid var(--border); background: var(--card); border-radius: 8px; padding: 10px 0; display: flex; flex-direction: column; align-items: center; }
.slot small { color: var(--primary); font-size: 11px; }
.slot.full { opacity: .4; }
.slot.picked { border-color: var(--primary); background: var(--primary); color: var(--card); }
.slot.picked small { color: var(--card); }
.empty { color: var(--muted-foreground); font-size: 14px; }
textarea { width: 100%; border: 1px solid var(--border); border-radius: 8px; padding: 10px; font-size: 14px; box-sizing: border-box; }
.primary { width: 100%; min-height: 48px; height: auto; padding: 12px 10px; background: var(--primary); color: var(--card); border: none; border-radius: 12px; font-size: 16px; margin-top: 10px; word-break: break-all; }
.primary:disabled { opacity: .5; }
.fuzzy-hint { color: var(--muted-foreground); font-size: 12px; text-align: center; margin-top: 8px; }
/* 预约成功页 */
.done { text-align: center; }
.done-icon { font-size: 44px; }
.done h1 { margin: 6px 0 14px; }
.done-card { background: var(--card); border-radius: 12px; padding: 14px; text-align: left; margin-bottom: 12px; }
.done-row { display: flex; justify-content: space-between; gap: 10px; padding: 7px 0; border-bottom: 1px solid var(--border); font-size: 14px; }
.done-row:last-child { border-bottom: 0; }
.done-row span { color: var(--muted-foreground); }
.done-tip { color: var(--muted-foreground); font-size: 13px; margin: 0 0 12px; }
.done-btns { display: flex; gap: 10px; margin-top: 16px; }
.done-btns .primary { flex: 1; text-align: center; text-decoration: none; line-height: 1.4; margin-top: 0; }
.done-btns .ghost { flex: 1; display: flex; align-items: center; justify-content: center; border: 1px solid var(--border); border-radius: 12px; color: var(--muted-foreground); text-decoration: none; font-size: 15px; }
</style>

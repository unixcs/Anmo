<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api, type BookingHalfDay, type BookingOptions } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { candidateDays, partOpen, partMeta, todayStr, trimPastSlots, type DayOption } from '../core/logic/booking'
import { yuan } from '../core/utils/format'
import { notify } from '../platform/notify/toast'
import ShopCard from '../components/ShopCard.vue'
import AppIcon from '../components/ui/AppIcon.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'

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
const loadingServices = ref(true)
const svcErr = ref(false)

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

// 服务目录加载失败要给"网络不可用 + 重试"，不能让空列表伪装成"暂未上架服务"
async function loadServices(): Promise<void> {
  loadingServices.value = true
  try {
    const catalog = await api.catalog()
    services.value = catalog.services
    svcErr.value = false
    const preset = route.query.service as string | undefined
    if (preset && services.value.some((s) => s.id === preset)) serviceId.value = preset
    day.value = candidateDays()
    if (serviceId.value) await refreshOptions()
  } catch {
    svcErr.value = true
  } finally {
    loadingServices.value = false
  }
}

onMounted(loadServices)

async function refreshOptions(): Promise<void> {
  const date = day.value[dayIdx.value]?.value
  options.value = null
  part.value = ''
  slotTime.value = ''
  if (!serviceId.value || !date) return
  loadingOptions.value = true
  try {
    const o = await api.bookingOptions(date)
    // 快速切换日期时丢弃过期响应
    if (day.value[dayIdx.value]?.value !== date) return
    trimPastSlots(o, date)
    options.value = o
  } catch (e) {
    if (day.value[dayIdx.value]?.value === date) notify((e as Error).message)
  } finally {
    loadingOptions.value = false
  }
}

function pickDay(i: number): void {
  dayIdx.value = i
  void refreshOptions()
}

function pickPart(p: 'AM' | 'PM'): void {
  // 闭店/时段已过/已约满的半天不可选（与小程序 slot-picker 同门槛）
  if (!partOpen(p === 'AM' ? options.value?.am : options.value?.pm)) return
  part.value = p
  slotTime.value = ''
}

async function submit(): Promise<void> {
  if (!selectedService.value || !part.value) {
    notify('请先选择服务、日期和上午/下午')
    return
  }
  // 页面跨零点挂着：日期条过期则刷新后重来（防提交昨天的日期）
  if (day.value[0]?.value !== todayStr()) {
    day.value = candidateDays()
    dayIdx.value = 0
    void refreshOptions()
    notify('日期已更新，请重新选择时间')
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
    <section v-if="success" class="done rise">
      <div class="done-icon">
        <AppIcon name="check" :size="30" :stroke-width="2.4" />
      </div>
      <h1>预约成功</h1>
      <div class="card done-card">
        <div class="done-row"><span>服务项目</span><b>{{ success.name }}</b></div>
        <div class="done-row"><span>日期</span><b class="num">{{ success.date }}</b></div>
        <div class="done-row">
          <span>时间</span>
          <b class="num">{{ success.time }}<template v-if="success.fuzzy">（具体时间由店主安排）</template></b>
        </div>
      </div>
      <p class="done-tip">到店后向商家出示「我的 → 核销码」即可</p>
      <ShopCard v-bind="shop" />
      <div class="done-btns">
        <RouterLink to="/me/appointments" class="btn primary pressable">查看我的预约</RouterLink>
        <RouterLink to="/" class="btn outline pressable">返回首页</RouterLink>
      </div>
    </section>

    <template v-else>
      <div class="page-head">
        <h1>预约</h1>
        <p class="sub">只需三步：选项目、选时间、留个备注</p>
      </div>

      <section class="block">
        <h2><i class="step-no">1</i>选择服务</h2>
        <AppSkeleton v-if="loadingServices" variant="text" :lines="3" />
        <div v-else-if="svcErr" class="load-failed">
          <p class="load-failed-main">网络不可用</p>
          <p class="load-failed-sub">服务列表没能加载，请稍后再试</p>
          <AppButton variant="outline" size="sm" @click="loadServices">重新加载</AppButton>
        </div>
        <div v-else-if="!services.length" class="load-failed">
          <p class="load-failed-sub">暂未上架服务，请稍后再来看看</p>
        </div>
        <div v-else class="svc-list">
          <button
            v-for="s in services"
            :key="s.id"
            type="button"
            class="svc pressable"
            :class="{ picked: s.id === serviceId }"
            @click="serviceId = s.id; refreshOptions()"
          >
            <span class="svc-name">{{ s.name }}</span>
            <span class="svc-meta num">{{ s.duration_minutes }}分钟 · ¥{{ yuan(s.default_price) }}</span>
          </button>
        </div>
      </section>

      <section v-if="serviceId" class="block">
        <h2><i class="step-no">2</i>选择日期</h2>
        <div class="day-row">
          <button
            v-for="(d, i) in day"
            :key="d.value"
            type="button"
            class="day pressable num"
            :class="{ picked: i === dayIdx }"
            @click="pickDay(i)"
          >{{ d.label }}</button>
        </div>
      </section>

      <section v-if="serviceId" class="block">
        <h2><i class="step-no">3</i>上午 / 下午（必选）</h2>
        <div v-if="loadingOptions" style="display: flex; flex-direction: column; gap: 8px">
          <AppSkeleton variant="text" :lines="2" />
        </div>
        <template v-else-if="options">
          <div v-if="!options.open" class="closed card plain">
            <AppIcon name="calendar-x" :size="20" />
            这一天休息，看看别的日期吧
          </div>
          <div v-else class="part-row">
            <button
              v-for="p in (['AM', 'PM'] as const)"
              :key="p"
              type="button"
              class="part pressable"
              :class="{ picked: part === p, off: !partOpen(p === 'AM' ? options.am : options.pm) }"
              @click="pickPart(p)"
            >
              <b>{{ p === 'AM' ? '上午' : '下午' }}</b>
              <span class="part-meta num">{{ partMeta(p === 'AM' ? options.am : options.pm) }}</span>
            </button>
          </div>

          <template v-if="half">
            <p class="slot-title">具体时间（可不选，由店主安排）</p>
            <div class="slot-grid">
              <button
                v-for="s in half.slots ?? []"
                :key="s.time"
                type="button"
                class="slot pressable num"
                :class="{ picked: s.time === slotTime, full: s.remaining < 1 }"
                :disabled="s.remaining < 1"
                @click="slotTime = s.time"
              >{{ s.time }}<small v-if="s.remaining < 1">满</small></button>
            </div>
          </template>
        </template>
      </section>

      <section v-if="serviceId" class="block">
        <h2><i class="step-no">4</i>备注（可选）</h2>
        <textarea v-model="note" rows="2" class="textarea" placeholder="身体状况、偏好等" />
      </section>

      <AppButton
        variant="primary"
        size="lg"
        block
        :loading="busy"
        :disabled="!part"
        class="submit"
        @click="submit"
      >
        {{ part ? `提交预约：${submitLabel}` : '请先选择上午 / 下午' }}
      </AppButton>
      <p class="fuzzy-hint">只选上午/下午提交 = 模糊预约，具体时间由店主安排，可能需要等待。</p>
    </template>
  </div>
</template>

<style scoped>
.block {
  margin: 22px 0;
}

.block h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  font: 600 15px/22px var(--font-stack);
  margin: 0 0 10px;
}

.step-no {
  font-style: normal;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: var(--radius-full);
  background: var(--primary-soft);
  color: var(--primary);
  font: 600 11px/1 var(--font-stack);
}

/* 服务选择 */
.load-failed {
  text-align: center;
  padding: 20px 12px;
}

.load-failed-main {
  font: 600 15px/1.4 var(--font-stack);
  color: var(--foreground);
}

.load-failed-sub {
  font: 400 13px/1.5 var(--font-stack);
  color: var(--muted-foreground);
  margin: 4px 0 12px;
}

.load-failed-sub:only-child {
  margin: 0;
}

.svc-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.svc {
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: var(--radius-md);
  padding: 12px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  cursor: pointer;
  transition: all 0.15s var(--ease);
}

.svc.picked {
  border-color: var(--primary);
  background: var(--primary-soft);
}

.svc-name {
  font-weight: 600;
}

.svc-meta {
  font: var(--font-sub);
  color: var(--muted-foreground);
}

.svc.picked .svc-meta {
  color: var(--primary);
}

/* 日期 */
.day-row {
  display: flex;
  gap: 8px;
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

/* 上午/下午 */
.part-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.part {
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: var(--radius-lg);
  padding: 16px 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  cursor: pointer;
  transition: all 0.15s var(--ease);
}

.part b {
  font-size: 17px;
}

.part.picked {
  border-color: var(--primary);
  background: var(--primary-soft);
  color: var(--primary);
}

.part.off {
  opacity: 0.45;
  cursor: not-allowed;
}

.part.off .part-meta {
  color: var(--muted-foreground);
}

.part-meta {
  color: var(--muted-foreground);
  font: var(--font-sub);
}

.part.picked .part-meta {
  color: var(--primary);
}

.slot-title {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 14px 0 8px;
}

.slot-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.slot {
  position: relative;
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

.slot small {
  position: absolute;
  top: -6px;
  right: -4px;
  background: var(--destructive);
  color: #fff;
  font-size: 10px;
  border-radius: var(--radius-full);
  padding: 1px 5px;
}

.closed {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  font: var(--font-sub);
  color: var(--muted-foreground);
}

.submit {
  margin-top: 24px;
}

.fuzzy-hint {
  color: var(--muted-foreground);
  font: var(--font-caption);
  text-align: center;
  margin: 10px 0 0;
}

/* 预约成功页 */
.done {
  text-align: center;
}

.done-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: var(--radius-full);
  background: var(--success-soft);
  color: var(--success);
  margin-bottom: 8px;
}

.done h1 {
  font: var(--font-display);
  margin: 0 0 16px;
}

.done-card {
  text-align: left;
  margin-bottom: 12px;
}

.done-row {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 0;
  border-bottom: 1px solid var(--border);
  font-size: 14px;
}

.done-row:last-child {
  border-bottom: 0;
}

.done-row span {
  color: var(--muted-foreground);
}

.done-row b {
  text-align: right;
}

.done-tip {
  color: var(--muted-foreground);
  font: var(--font-sub);
  margin: 0 0 16px;
}

.done-btns {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.done-btns .btn {
  flex: 1;
  text-decoration: none;
}
</style>

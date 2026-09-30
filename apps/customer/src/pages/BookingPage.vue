<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type BookingHalfDay, type BookingOptions } from '../core/api/endpoints'
import type { ServiceItem } from '../core/models/models'
import { candidateDays, partOpen, partMeta, todayStr, trimPastSlots, type DayOption } from '../core/logic/booking'
import { yuan } from '../core/utils/format'
import { profileProgress } from '../core/utils/profile'
import { notify } from '../platform/notify/toast'
import { currentToken } from '../platform/auth/session'
import ShopCard from '../components/ShopCard.vue'
import AppSheet from '../components/ui/AppSheet.vue'
import AppIcon from '../components/ui/AppIcon.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppSkeleton from '../components/ui/AppSkeleton.vue'

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
const loadingServices = ref(true)
const svcErr = ref(false)

// 预约成功页（goal §19/§66）：页内成功态，展示服务/日期/时间 + 门店信息卡
const success = ref<{ name: string; date: string; time: string; fuzzy: boolean } | null>(null)
const shop = ref({ address: '', phone: '', latitude: '', longitude: '' })

// 资料完善半屏提示（V2.2 R4）：拿原始资料判完善度，拉不到不阻塞预约
const member = ref<{ name?: string; phone?: string } | null>(null)
const profileSheet = ref(false)

const selectedService = computed(() => services.value.find((s) => s.id === serviceId.value))
const half = computed<BookingHalfDay | null>(() => {
  if (!options.value || !part.value) return null
  return part.value === 'AM' ? options.value.am : options.value.pm
})

// 底部固定提交条（V2.2 R1，与小程序 booking 同语义）：置灰三态 + 摘要行
const submitDisabled = computed(
  () => busy.value || !selectedService.value || day.value.length === 0 || !part.value,
)
const submitText = computed(() => {
  if (busy.value) return '提交中…'
  if (!selectedService.value) return '请先选择服务'
  if (!day.value.length) return '请先选择日期'
  if (!part.value) return '请先选择上午或下午'
  return '立即预约'
})
const submitSummary = computed(() => {
  if (!selectedService.value || !part.value) return ''
  const d = day.value[dayIdx.value]
  const t = slotTime.value || '店家安排时间'
  return `${selectedService.value.name} · ${d?.label.slice(0, 5)} ${part.value === 'AM' ? '上午' : '下午'} ${t}`
})

// ---- 资料完善往返的已选保留（V2.2 R4）：H5 路由离开会重建页面实例，草稿存 sessionStorage ----
interface BookingDraft {
  serviceId: string
  dayIdx: number
  part: '' | 'AM' | 'PM'
  slotTime: string
  note: string
}
let pendingDraft: BookingDraft | null = null

function takeDraft(): BookingDraft | null {
  try {
    const raw = sessionStorage.getItem('anmo.booking.draft')
    if (!raw) return null
    sessionStorage.removeItem('anmo.booking.draft')
    return JSON.parse(raw) as BookingDraft
  } catch {
    return null
  }
}

// 选项就绪后恢复上/下午与具体时段：半天当前仍可选才恢复（与 pickPart 同门槛）
function applyPendingDraft(): void {
  if (!pendingDraft) return
  const d = pendingDraft
  pendingDraft = null
  if (!d.part || !options.value) return
  const half = d.part === 'AM' ? options.value.am : options.value.pm
  if (!partOpen(half)) return
  part.value = d.part
  if (d.slotTime && (half.slots ?? []).some((s) => s.time === d.slotTime && s.remaining >= 1)) {
    slotTime.value = d.slotTime
  }
}

// 服务目录加载失败要给"网络不可用 + 重试"，不能让空列表伪装成"暂未上架服务"
async function loadServices(): Promise<void> {
  loadingServices.value = true
  try {
    const catalog = await api.catalog()
    services.value = catalog.services
    svcErr.value = false
    const preset = (route.query.service as string | undefined) || pendingDraft?.serviceId || ''
    if (preset && services.value.some((s) => s.id === preset)) serviceId.value = preset
    day.value = candidateDays()
    if (pendingDraft && pendingDraft.dayIdx > 0 && pendingDraft.dayIdx < day.value.length) {
      dayIdx.value = pendingDraft.dayIdx
    }
    if (serviceId.value) await refreshOptions()
    applyPendingDraft()
  } catch {
    svcErr.value = true
  } finally {
    loadingServices.value = false
  }
}

onMounted(() => {
  const draft = takeDraft()
  if (draft) {
    pendingDraft = draft
    note.value = draft.note || ''
  }
  void loadServices()
  // 游客浏览（V2.2 第五批）：myProfile 是登录资源，游客请求会 401 并触发全局跳登录；
  // 未登录时直接不拉（资料完善提醒本来只对已登录用户有意义）
  if (currentToken()) {
    api
      .myProfile()
      .then((r) => (member.value = r.member))
      .catch(() => {})
  }
})

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

// ---- 已选保留（V2.2 R4/第五批）：路由离开会重建页面实例，草稿存 sessionStorage ----
// 使用场景：去完善资料往返、游客点提交跳登录往返（登录后回到预约页预选不丢）
function saveDraft(): void {
  try {
    sessionStorage.setItem(
      'anmo.booking.draft',
      JSON.stringify({
        serviceId: serviceId.value,
        dayIdx: dayIdx.value,
        part: part.value,
        slotTime: slotTime.value,
        note: note.value,
      } satisfies BookingDraft),
    )
  } catch {
    /* 存储不可用仅损失"已选保留"体验 */
  }
}

// ---- 资料完善半屏提示（V2.2 R4）----
// 去完善资料：已选存 sessionStorage，返回后恢复；不置跳过标记，完善前再次提交仍会提醒
function goProfileSheet(): void {
  profileSheet.value = false
  saveDraft()
  router.push('/me/profile')
}

// 立即预约：置静默标记继续原流程，下次不再提醒
function bookNow(): void {
  localStorage.setItem('anmo.profile.bookingSkipped', '1')
  profileSheet.value = false
  void submit()
}

async function submit(): Promise<void> {
  // 游客浏览（V2.2 第五批）：/booking 路由已公开，提交动作才要求登录；
  // 已选存草稿，登录回跳后恢复预选
  if (!currentToken()) {
    notify('登录后即可提交预约')
    saveDraft()
    router.push({ name: 'login', query: { redirect: '/booking' } })
    return
  }
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
  // 资料完善半屏提示（V2.2 R4）：资料不全且未永久跳过 → 拦截首次点击
  if (
    member.value &&
    profileProgress(member.value).pct < 100 &&
    !localStorage.getItem('anmo.profile.bookingSkipped')
  ) {
    profileSheet.value = true
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
  <div class="page booking" :class="{ 'with-bar': !success }">
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

      <!-- 底部固定提交条（V2.2 R1）：不随内容滚动，bottom 让开 App.vue 底部 tab 栏 -->
      <div class="submit-bar">
        <p v-if="submitSummary" class="submit-info num">{{ submitSummary }}</p>
        <AppButton variant="primary" size="lg" block :loading="busy" :disabled="submitDisabled" @click="submit">
          {{ submitText }}
        </AppButton>
        <p class="submit-hint">只选上午/下午提交 = 模糊预约，具体时间由店主安排，可能需要等待。</p>
      </div>
      <!-- 资料完善半屏提示（V2.2 R4）：资料不全时首次点提交弹出，轻量不打断 -->
      <AppSheet :open="profileSheet" title="完善一下资料" @close="profileSheet = false">
        <p class="sheet-body">先完善资料，店主更好安排；不填也可直接预约。</p>
        <div class="sheet-ops">
          <AppButton variant="primary" block @click="bookNow">立即预约</AppButton>
          <AppButton variant="ghost" block @click="goProfileSheet">去完善资料</AppButton>
        </div>
      </AppSheet>
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

/* 底部固定提交条（V2.2 R1）：bottom 让开 tab 栏（--tabbar-h 与 App.vue .tabbar 同步），
   iPhone 底部安全区随 tab 栏一起让位；内容区 .with-bar 预留等高留白防备注被盖 */
.booking.with-bar {
  padding-bottom: calc(215px + env(safe-area-inset-bottom));
}

.submit-bar {
  position: fixed;
  bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom));
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  max-width: 480px;
  background: var(--card);
  border-top: 1px solid var(--border);
  box-shadow: 0 -4px 12px rgba(34, 30, 27, 0.06);
  padding: 10px 16px;
  z-index: 30;
}

.submit-info {
  font: var(--font-caption);
  color: var(--muted-foreground);
  text-align: center;
  margin: 0 0 6px;
}

.submit-hint {
  font: var(--font-caption);
  color: var(--muted-foreground);
  text-align: center;
  margin: 6px 0 0;
}

/* 资料完善半屏（V2.2 R4） */
.sheet-body {
  font: var(--font-sub);
  color: var(--muted-foreground);
  margin: 0 0 14px;
}

.sheet-ops {
  display: flex;
  flex-direction: column;
  gap: 10px;
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

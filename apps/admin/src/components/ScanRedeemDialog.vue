<template>
  <el-dialog :model-value="modelValue" title="扫码结算" width="min(600px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)" @open="onOpen" @close="onClose">
    <!-- 第一步：扫码 / 输入（识别 ANMO-MEMBER 会员码） -->
    <template v-if="step === 'scan'">
      <div class="cam-wrap">
        <video v-show="camReady" ref="videoEl" class="cam" muted playsinline />
        <canvas ref="canvasEl" class="snap" />
        <div v-if="!camReady" class="cam-hint">
          <p class="cam-icon">📷</p>
          <p v-if="camError">{{ camError }}</p>
          <p v-else-if="canUseCamera">正在打开摄像头…</p>
          <p v-else>
            当前页面无法直接调用摄像头（浏览器仅允许 HTTPS 或 localhost 使用）。<br />
            点击下方「拍照识别二维码」即可：手机会调起相机拍照，电脑会打开图片选择。
          </p>
        </div>
      </div>
      <input ref="fileEl" type="file" accept="image/*" capture="environment" class="qr-file" @change="onPickImage" />
      <div class="scan-actions">
        <el-button v-if="canUseCamera" :type="camReady ? 'default' : 'primary'"
          @click="camReady ? stopCamera() : startCamera()">
          {{ camReady ? '停止摄像头' : '打开摄像头' }}
        </el-button>
        <el-button type="success" :loading="qrDecoding" @click="fileEl?.click()">📷 拍照识别二维码</el-button>
      </div>
      <p class="hint">扫顾客「核销码」定位会员后选择预约按卡结算；无预约扫卡也能直接扣卡结算。也可在下方手动输入手机号。</p>

      <div class="manual">
        <el-input v-model="manual" placeholder="手动输入：手机号（11位）" clearable
          @keyup.enter="submitManual" />
        <el-button type="primary" :loading="resolving" @click="submitManual">查找会员</el-button>
      </div>

      <template v-if="phoneMatches.length > 0">
        <p class="hint">匹配到多位会员，请选择：</p>
        <el-table :data="phoneMatches" size="small" @row-click="(row: Member) => resolveMember(row.id)">
          <el-table-column prop="name" label="姓名" width="110" />
          <el-table-column prop="phone" label="手机号" width="130" />
          <el-table-column prop="member_no" label="会员号" min-width="150" />
        </el-table>
      </template>
    </template>

    <!-- 第二步：统一结算页（会员 + 今日预约 + 实际服务 + 结算方式） -->
    <template v-else-if="step === 'settle' && member">
      <div class="member-box">
        <b>{{ member.name }}</b>
        <span class="sub">（{{ member.phone }} · {{ member.member_no }}）</span>
        <el-button size="small" link @click="resetToScan">重新扫码</el-button>
      </div>

      <!-- 今日预约：0 个=散客；1 个自动选；多个让老板选（goal §10） -->
      <el-form label-width="82px" class="pick">
        <el-form-item label="今日预约">
          <el-select v-model="selectedAptId" style="width: 100%"
            :placeholder="walkIn ? '今日无待服务预约' : '选择要结算的预约'">
            <el-option v-for="a in myApts" :key="a.id" :value="a.id" :label="aptLabel(a)" />
          </el-select>
        </el-form-item>
        <el-form-item label="实际服务">
          <el-select v-model="serviceId" style="width: 100%" placeholder="选择本次实际服务">
            <el-option v-for="s in settleServices" :key="s.id" :value="s.id"
              :label="`${s.name}（¥${fen(s.default_price)}）`" />
          </el-select>
          <div v-if="snapshotName && serviceId === snapshotServiceId" class="field-hint">
            默认为预约服务「{{ snapshotName }}」，可改成实际服务
          </div>
        </el-form-item>
        <el-form-item label="结算方式">
          <el-radio-group v-model="mode">
            <el-radio-button value="card" :disabled="activeCards.length === 0">会员卡</el-radio-button>
            <el-radio-button value="CASH" :disabled="!canCash">现金</el-radio-button>
            <el-radio-button value="WECHAT_TRANSFER" :disabled="!canCash">微信</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="mode === 'card'" label="扣哪张卡">
          <el-select v-model="cardId" style="width: 100%" placeholder="选择会员卡（结算时才扣次）">
            <el-option v-for="c in activeCards" :key="c.id" :value="c.id"
              :label="`${cardName(c)} · 剩 ${c.remaining_count}/${c.total_count} 次`" />
          </el-select>
        </el-form-item>
        <el-form-item v-else label="金额">
          <el-input v-model="cashAmount" type="number" :min="0" style="width: 160px">
            <template #append>元</template>
          </el-input>
        </el-form-item>
      </el-form>

      <el-alert v-if="walkIn" type="info" :closable="false" show-icon style="margin-bottom: 10px"
        title="散客直接结算" :description="walkInHint" />
      <p v-if="mode === 'card' && !cardId && activeCards.length === 0" class="warn">
        该会员没有可用会员卡，请改用现金/微信，或先在「会员」页发卡。
      </p>

      <ServiceRecordFields v-model="record" />

      <div class="actions">
        <el-button type="primary" size="large" :loading="resolving" :disabled="!canSubmit" class="go"
          @click="doSettle">完成并结算</el-button>
        <el-button size="large" @click="refreshMember">刷新</el-button>
      </div>
    </template>

    <!-- 第三步：结果 -->
    <template v-else-if="step === 'result'">
      <el-result icon="success" title="结算成功" :sub-title="resultText" />
      <div class="actions center">
        <el-button type="primary" @click="resetToScan">继续扫码</el-button>
        <el-button @click="$emit('update:modelValue', false)">关闭</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import jsQR from 'jsqr'
import { ElMessage } from 'element-plus'
import {
  getMember,
  getToday,
  listCardTemplates,
  listMemberCards,
  listMembers,
  listServices,
  redeemCard,
  redeemWalkIn,
  settlePayment,
  type Member,
  type MemberCard,
  type RecordPayload,
  type TodayAppointment,
} from '../core/api/admin'
import { APT_STATUS_TEXT, fmtTime, todayStr } from '../core/format'
import ServiceRecordFields from './ServiceRecordFields.vue'

const MEMBER_PREFIX = 'ANMO-MEMBER:'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'settled'): void
}>()

const step = ref<'scan' | 'settle' | 'result'>('scan')
const videoEl = ref<HTMLVideoElement | null>(null)
const canvasEl = ref<HTMLCanvasElement | null>(null)
const fileEl = ref<HTMLInputElement | null>(null)
const camReady = ref(false)
const camError = ref('')
const manual = ref('')
const resolving = ref(false)
const qrDecoding = ref(false)

// getUserMedia 只在安全上下文（HTTPS / localhost）存在；HTTP 页面走「拍照识别」路径
const canUseCamera = !!navigator.mediaDevices?.getUserMedia

const member = ref<Member | null>(null)
const activeCards = ref<MemberCard[]>([])
const templateMap = ref<Record<string, string>>({})
const templateRuleMap = ref<Record<string, string[]>>({})
const myApts = ref<TodayAppointment[]>([])
const serviceCatalog = ref<{ id: string; name: string; default_price: number; status: string }[]>([])
const phoneMatches = ref<Member[]>([])
const resultText = ref('')

// 结算单状态（goal §8-§11：预约可选关联；实际服务可改；结算时才选卡，服务过程不锁卡）
const selectedAptId = ref('')
const mode = ref<'card' | 'CASH' | 'WECHAT_TRANSFER'>('card')
const cardId = ref('')
const serviceId = ref('')
const cashAmount = ref('')
const record = ref<RecordPayload>({
  body_parts: [],
  service_method: '',
  tech_note: '',
  communicated: false,
})

let stream: MediaStream | null = null
let raf = 0

const activeApts = computed(() =>
  myApts.value.filter((a) => a.status === 'WAITING' || a.status === 'IN_SERVICE'),
)
const walkIn = computed(() => activeApts.value.length === 0)
const selectedApt = computed(() => myApts.value.find((a) => a.id === selectedAptId.value) ?? null)
const snapshotServiceId = computed(() => selectedApt.value?.service?.service_id ?? '')
const snapshotName = computed(() => selectedApt.value?.service?.service_name_snapshot ?? '')

// 结算方式可用性：现金/微信走预约收款（散客现金收款无端点，明确提示不做）
const canCash = computed(() => !walkIn.value)
const walkInHint = computed(() => {
  if (activeCards.value.length > 0) return '无待服务预约：选卡与本次服务即可扣次结算。'
  return '该会员今日无待服务预约且无可用卡；现金/微信收款需关联预约，请先让顾客预约或先发卡。'
})

// 可选实际服务：按所选卡的服务规则过滤（未配规则=不限制，后端兜底）
const settleServices = computed(() => {
  let list = serviceCatalog.value.filter((s) => s.status === 'ACTIVE')
  const card = activeCards.value.find((c) => c.id === cardId.value)
  if (card) {
    const allowed = templateRuleMap.value[card.card_template_id]
    if (allowed && allowed.length > 0) list = list.filter((s) => allowed.includes(s.id))
  }
  // 预约模式把预约快照服务排最前（即使 catalog 已下架也保留快照可选）
  if (selectedApt.value?.service) {
    const svc = selectedApt.value.service
    if (!list.some((s) => s.id === svc.service_id)) {
      list = [{ id: svc.service_id, name: `${svc.service_name_snapshot}（预约快照）`, default_price: svc.price_snapshot, status: 'ACTIVE' }, ...list]
    }
  }
  return list
})

const canSubmit = computed(() => {
  if (!record.value.communicated) return false // 沟通确认必勾（D28）
  if (mode.value === 'card') {
    // 卡结算：选卡 + 选服务；预约模式还需定位到预约（散客不需要）
    return !!cardId.value && !!serviceId.value && (walkIn.value || !!selectedAptId.value)
  }
  return selectedAptId.value !== '' && Number(cashAmount.value) > 0
})

function aptLabel(a: TodayAppointment): string {
  const time = a.slot_type === 'HALF_DAY' ? (a.day_part === 'AM' ? '上午' : '下午') : fmtTime(a.scheduled_start).slice(11)
  return `${time} ${a.service?.service_name_snapshot ?? ''}（${APT_STATUS_TEXT[a.status] ?? a.status}）`
}

function cardName(c: MemberCard): string {
  return templateMap.value[c.card_template_id] ?? c.card_template_id.slice(0, 8)
}

function fen(v: number): string {
  return (v / 100).toFixed(v % 100 === 0 ? 0 : 2)
}

watch([cardId, selectedAptId], () => {
  // 换卡/换单后重算可选服务；默认落在预约快照服务，否则第一项
  const keep = settleServices.value.some((s) => s.id === serviceId.value)
  if (!keep) serviceId.value = snapshotServiceId.value || settleServices.value[0]?.id || ''
})

watch(walkIn, (w) => {
  if (w && mode.value !== 'card') mode.value = activeCards.value.length > 0 ? 'card' : 'CASH'
})

// ---------- 摄像头 ----------
async function startCamera() {
  camError.value = ''
  camReady.value = false
  try {
    stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
    })
    camReady.value = true
    // 等 DOM 渲染出 video 再绑定
    setTimeout(() => {
      if (!videoEl.value || !stream) return
      videoEl.value.srcObject = stream
      void videoEl.value.play()
      tick()
    }, 60)
  } catch {
    camError.value =
      '摄像头打开失败（可能未授权或被占用）。可用「拍照识别二维码」或下方手动输入手机号。'
  }
}

function stopCamera() {
  cancelAnimationFrame(raf)
  stream?.getTracks().forEach((t) => t.stop())
  stream = null
  camReady.value = false
}

function tick() {
  const video = videoEl.value
  const canvas = canvasEl.value
  if (!video || !canvas || !stream) return
  if (video.readyState === video.HAVE_ENOUGH_DATA && video.videoWidth > 0) {
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    const ctx = canvas.getContext('2d')
    if (ctx) {
      ctx.drawImage(video, 0, 0)
      const img = ctx.getImageData(0, 0, canvas.width, canvas.height)
      const code = jsQR(img.data, img.width, img.height, { inversionAttempts: 'dontInvert' })
      if (code?.data) {
        stopCamera()
        void handleCode(code.data)
        return
      }
    }
  }
  raf = requestAnimationFrame(tick)
}

// ---------- 拍照识别（HTTP 页面也可用：调起手机原生相机 / 电脑选图） ----------
async function onPickImage(e: Event): Promise<void> {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  stopCamera()
  qrDecoding.value = true
  try {
    const code = await decodeImageFile(file)
    if (!code?.data) {
      ElMessage.warning('未能从这张照片识别出二维码，请正对二维码、避免反光后重拍')
      return
    }
    await handleCode(code.data)
  } catch {
    ElMessage.error('图片读取失败，请重试')
  } finally {
    qrDecoding.value = false
  }
}

async function decodeImageFile(file: File): Promise<{ data: string } | null> {
  const bitmap = await createImageBitmap(file)
  const canvas = document.createElement('canvas')
  // 过大的照片等比缩到 1600px 内，兼顾识别率与解码耗时
  const maxDim = 1600
  const scale = Math.min(1, maxDim / Math.max(bitmap.width, bitmap.height))
  canvas.width = Math.max(1, Math.round(bitmap.width * scale))
  canvas.height = Math.max(1, Math.round(bitmap.height * scale))
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) throw new Error('canvas 不可用')
  ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
  bitmap.close()
  const img = ctx.getImageData(0, 0, canvas.width, canvas.height)
  return jsQR(img.data, img.width, img.height, { inversionAttempts: 'attemptBoth' })
}

// ---------- 识别与解析：会员码 → 定位会员 ----------
async function handleCode(raw: string): Promise<void> {
  const content = raw.trim()
  if (content.startsWith(MEMBER_PREFIX)) {
    await resolveMember(content.slice(MEMBER_PREFIX.length))
    return
  }
  ElMessage.warning('不是本店核销码（应为 ANMO-MEMBER 开头）')
  if (canUseCamera) await startCamera()
}

async function submitManual(): Promise<void> {
  const v = manual.value.trim()
  if (!v) return
  if (/^\d{11}$/.test(v)) {
    resolving.value = true
    try {
      const res = await listMembers(v, 1, 20)
      if (res.data.length === 1) await resolveMember(res.data[0].id)
      else if (res.data.length === 0) ElMessage.warning('未找到该手机号的会员')
      else phoneMatches.value = res.data
    } catch (e) {
      ElMessage.error(e instanceof Error ? e.message : '查询失败')
    } finally {
      resolving.value = false
    }
    return
  }
  ElMessage.warning('请输入 11 位手机号，或扫描顾客二维码')
}

// ---------- 结算单装配 ----------
async function resolveMember(memberId: string): Promise<void> {
  resolving.value = true
  try {
    await assemble(memberId)
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '识别失败')
    if (canUseCamera) await startCamera()
  } finally {
    resolving.value = false
  }
}

/** 拉齐会员/卡/今日预约/服务目录并初始化结算单 */
async function assemble(memberId: string): Promise<void> {
  const [detail, cards, tpl, today, svcs] = await Promise.all([
    getMember(memberId),
    listMemberCards(memberId),
    listCardTemplates(),
    getToday(todayStr()),
    listServices(),
  ])
  serviceCatalog.value = (svcs ?? []).map((s) => ({ id: s.id, name: s.name, default_price: s.default_price, status: s.status }))
  member.value = detail.member
  activeCards.value = (cards ?? []).filter((c) => c.status === 'ACTIVE')
  templateMap.value = Object.fromEntries((tpl ?? []).map((t) => [t.id, t.name]))
  templateRuleMap.value = Object.fromEntries((tpl ?? []).map((t) => [t.id, t.service_ids ?? []]))
  myApts.value = (today.appointments ?? []).filter((a) => a.member_id === memberId)
  phoneMatches.value = []
  manual.value = ''
  step.value = 'settle'

  const preferred = activeApts.value.find((a) => a.status === 'IN_SERVICE')?.id
    ?? activeApts.value[0]?.id
    ?? ''
  selectedAptId.value = preferred
  cardId.value = activeCards.value[0]?.id ?? ''
  mode.value = activeCards.value.length > 0
    ? 'card'
    : (activeApts.value.length > 0 ? 'CASH' : 'card')
  serviceId.value = ''
  cashAmount.value = ''
  record.value = { body_parts: [], service_method: '', tech_note: '', communicated: false }
  // 默认实际服务 = 预约快照服务；散客默认第一项
  applyServiceDefault()
  if (walkIn.value && activeCards.value.length === 0) {
    ElMessage.info('该会员今日无待服务预约且无可用卡')
  }
}

function applyServiceDefault(): void {
  serviceId.value = snapshotServiceId.value || settleServices.value[0]?.id || ''
  syncCashDefault()
}

function syncCashDefault(): void {
  if (mode.value === 'card') return
  const svc = serviceCatalog.value.find((s) => s.id === serviceId.value)
  if (svc && !cashAmount.value) cashAmount.value = String(fen(svc.default_price))
}

watch(serviceId, () => syncCashDefault())
watch(mode, () => syncCashDefault())

async function refreshMember(): Promise<void> {
  if (member.value) {
    resolving.value = true
    try {
      await assemble(member.value.id)
    } finally {
      resolving.value = false
    }
  }
}

// ---------- 提交：一条链路分发到三个后端事务 ----------
async function doSettle(): Promise<void> {
  resolving.value = true
  try {
    if (mode.value === 'card') {
      if (!cardId.value) {
        ElMessage.warning('请选择会员卡')
        return
      }
      if (!serviceId.value) {
        ElMessage.warning('请选择本次实际服务')
        return
      }
      if (walkIn.value) {
        // 散客核销（D19）：无预约按卡扣次
        const res = await redeemWalkIn(cardId.value, serviceId.value, record.value)
        resultText.value = `卡余额 ${res.redemption.before_count} → ${res.redemption.after_count} 次；已记散客核销收款。`
      } else {
        if (!selectedAptId.value) {
          ElMessage.warning('请选择要结算的预约')
          return
        }
        const res = await redeemCard(selectedAptId.value, cardId.value, serviceId.value, record.value)
        resultText.value = `卡余额 ${res.redemption.before_count} → ${res.redemption.after_count} 次；预约已完成并记 CARD 收款。`
      }
    } else {
      // 现金 / 微信：关联预约收款，同事务完成预约（§8）
      if (!selectedAptId.value) {
        ElMessage.warning('请选择要结算的预约')
        return
      }
      await settlePayment(selectedAptId.value, {
        method: mode.value,
        amount: Math.round(Number(cashAmount.value) * 100),
      }, record.value)
      resultText.value = `已记 ${mode.value === 'CASH' ? '现金' : '微信'}收款，预约已完成。`
    }
    step.value = 'result'
    emit('settled')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '结算失败')
  } finally {
    resolving.value = false
  }
}

function resetToScan(): void {
  step.value = 'scan'
  member.value = null
  myApts.value = []
  activeCards.value = []
  selectedAptId.value = ''
  cardId.value = ''
  serviceId.value = ''
  cashAmount.value = ''
  record.value = { body_parts: [], service_method: '', tech_note: '', communicated: false }
  phoneMatches.value = []
  if (canUseCamera) void startCamera()
}

function onOpen(): void {
  step.value = 'scan'
  if (canUseCamera) void startCamera()
}

function onClose(): void {
  stopCamera()
}

watch(
  () => props.modelValue,
  (v) => {
    if (!v) onClose()
  },
)
</script>

<style scoped>
.cam-wrap {
  position: relative;
  height: 240px;
  background: #1f2d3d;
  border-radius: 10px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cam {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.snap {
  display: none;
}
.cam-hint {
  color: #cbd3da;
  font-size: 13px;
  text-align: center;
  padding: 0 20px;
}
.cam-icon {
  font-size: 30px;
  margin-bottom: 6px;
}
.qr-file {
  display: none;
}
.scan-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}
.scan-actions .el-button {
  flex: 1;
}
.hint {
  color: #909399;
  font-size: 12px;
  margin: 10px 0;
}
.manual {
  display: flex;
  gap: 10px;
}
.member-box {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}
.sub {
  color: #909399;
  font-size: 13px;
}
.pick {
  margin-top: 4px;
}
.field-hint {
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}
.warn {
  color: #e6a23c;
  font-size: 12px;
  margin: 0 0 10px;
}
.actions {
  margin-top: 12px;
  display: flex;
  gap: 10px;
}
.actions .go {
  flex: 1;
}
.actions.center {
  justify-content: center;
}
</style>

<template>
  <el-dialog :model-value="modelValue" title="扫码核销" width="min(600px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)" @open="onOpen" @close="onClose">
    <!-- 第一步：扫码 / 输入 -->
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
      <p class="hint">现场扫码：打开摄像头对准顾客"我的核销码"；HTTP 页面请用「拍照识别二维码」（对顾客手机上的核销码拍照）。也可在下方手动输入手机号。</p>

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

    <!-- 第二步：会员 + 预约 + 选卡 -->
    <template v-else-if="step === 'resolved' && member">
      <div class="member-box">
        <b>{{ member.name }}</b>
        <span class="sub">（{{ member.phone }} · {{ member.member_no }}）</span>
        <el-button size="small" link @click="resetToScan">重新扫码</el-button>
      </div>

      <el-form label-width="90px" class="pick">
        <el-form-item label="今日预约">
          <el-select v-model="selectedAptId" style="width: 100%" placeholder="该会员今日暂无预约">
            <el-option v-for="a in myApts" :key="a.id" :value="a.id"
              :label="`${fmtTime(a.scheduled_start).slice(11)} ${a.service?.service_name_snapshot ?? ''}（${APT_STATUS_TEXT[a.status]}）`" />
          </el-select>
        </el-form-item>
      </el-form>
      <p v-if="myApts.length > 0 && !inServiceApts.includes(selectedAptId ?? '')" class="warn">
        核销需要预约处于"服务中"状态；可先在下方开始服务，或到"预约管理"操作。
      </p>

      <el-table :data="activeCards" size="small" v-loading="resolving">
        <el-table-column label="卡" min-width="130">
          <template #default="{ row }">{{ templateName(row.card_template_id) }}</template>
        </el-table-column>
        <el-table-column label="剩余/总" width="90">
          <template #default="{ row }">{{ row.remaining_count }}/{{ row.total_count }}</template>
        </el-table-column>
        <el-table-column label="有效期至" width="110">
          <template #default="{ row }">{{ row.valid_until ?? '永久' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button size="small" type="primary" :disabled="row.remaining_count < 1 || !selectedAptId"
              @click="doRedeem(row)">核销 1 次</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="activeCards.length === 0" description="该会员没有可用会员卡" :image-size="60" />

      <div class="actions">
        <el-button v-if="confirmableApt" size="default" type="success" @click="startService">开始服务</el-button>
        <el-button size="default" @click="refreshMember">刷新</el-button>
      </div>
    </template>

    <!-- 第三步：结果 -->
    <template v-else-if="step === 'result'">
      <el-result icon="success" title="核销成功"
        :sub-title="resultText" />
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
  redeemCard,
  startAppointment,
  type Member,
  type MemberCard,
  type TodayAppointment,
} from '../core/api/admin'
import { APT_STATUS_TEXT, fmtTime, todayStr } from '../core/format'

const QR_PREFIX = 'ANMO-MEMBER:'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'settled'): void
}>()

const step = ref<'scan' | 'resolved' | 'result'>('scan')
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
const myApts = ref<TodayAppointment[]>([])
const selectedAptId = ref('')
const phoneMatches = ref<Member[]>([])
const resultText = ref('')

let stream: MediaStream | null = null
let raf = 0

const inServiceApts = computed(() =>
  myApts.value.filter((a) => a.status === 'IN_SERVICE').map((a) => a.id),
)
const confirmableApt = computed(() => {
  const a = myApts.value.find((x) => x.id === selectedAptId.value)
  return a && a.status === 'CONFIRMED' ? a : null
})

function templateName(id: string): string {
  return templateMap.value[id] ?? id.slice(0, 8)
}

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
      ElMessage.warning('未能从这张照片识别出二维码，请正对核销码、避免反光后重拍')
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

// ---------- 识别与解析 ----------
async function handleCode(raw: string): Promise<void> {
  const content = raw.trim()
  if (!content.startsWith(QR_PREFIX)) {
    ElMessage.warning('不是本店核销码（应为 ANMO-MEMBER 开头）')
    if (canUseCamera) await startCamera()
    return
  }
  await resolveMember(content.slice(QR_PREFIX.length))
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
  ElMessage.warning('请输入 11 位手机号，或扫描顾客核销码')
}

// ---------- 会员定位 ----------
async function resolveMember(memberId: string): Promise<void> {
  resolving.value = true
  try {
    const [detail, cards, tpl, today] = await Promise.all([
      getMember(memberId),
      listMemberCards(memberId),
      listCardTemplates(),
      getToday(todayStr()),
    ])
    member.value = detail.member
    activeCards.value = (cards ?? []).filter((c) => c.status === 'ACTIVE')
    templateMap.value = Object.fromEntries((tpl ?? []).map((t) => [t.id, t.name]))
    myApts.value = (today.appointments ?? []).filter((a) => a.member_id === memberId)
    phoneMatches.value = []
    manual.value = ''
    const preferred = myApts.value.find((a) => a.status === 'IN_SERVICE')
    selectedAptId.value = preferred?.id ?? myApts.value[0]?.id ?? ''
    step.value = 'resolved'
    if (myApts.value.length === 0) {
      ElMessage.info('该会员今日暂无预约')
    }
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '识别失败')
    if (canUseCamera) await startCamera()
  } finally {
    resolving.value = false
  }
}

async function refreshMember(): Promise<void> {
  if (member.value) await resolveMember(member.value.id)
}

async function startService(): Promise<void> {
  const apt = confirmableApt.value
  if (!apt) return
  try {
    await startAppointment(apt.id)
    ElMessage.success('已开始服务')
    await refreshMember()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '开始服务失败')
  }
}

async function doRedeem(card: MemberCard): Promise<void> {
  if (!selectedAptId.value) {
    ElMessage.warning('请先选择要结算的预约')
    return
  }
  resolving.value = true
  try {
    const res = await redeemCard(selectedAptId.value, card.id)
    const rd = res.redemption
    resultText.value = `卡余额 ${rd.before_count} → ${rd.after_count} 次；预约已自动完成并记 CARD 收款。`
    step.value = 'result'
    emit('settled')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '核销失败')
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
}
.sub {
  color: #909399;
  font-size: 13px;
}
.pick {
  margin-top: 4px;
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
.actions.center {
  justify-content: center;
}
</style>

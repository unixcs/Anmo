<template>
  <el-dialog :model-value="modelValue" title="散客快速结算" width="min(520px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)" @open="onOpen">
    <el-form label-width="90px">
      <el-form-item label="手机号">
        <el-input v-model="phone" maxlength="11" placeholder="选填；登记后生成服务记录" style="width: 200px"
          clearable @input="onPhoneInput" @blur="checkPhone" />
        <span v-if="checking" class="sub-hint">查询中…</span>
        <span v-else-if="phoneKnown === false" class="sub-hint new">将创建会员档案</span>
        <span v-else-if="phoneKnown === true" class="sub-hint">已匹配老会员</span>
      </el-form-item>
      <el-form-item v-if="phone" label="姓名">
        <el-input v-model="name" placeholder="选填" style="width: 200px" />
      </el-form-item>
      <el-form-item label="服务项目" required>
        <el-select v-model="serviceId" style="width: 100%" placeholder="选择本次服务项目" @change="onServiceChange">
          <el-option v-for="s in services" :key="s.id" :value="s.id"
            :label="`${s.name}（¥${fen(s.default_price)} · ${s.duration_minutes}分钟）`" />
        </el-select>
      </el-form-item>
      <el-form-item label="收款方式">
        <el-radio-group v-model="payMethod">
          <el-radio value="CASH">现金</el-radio>
          <el-radio value="WECHAT_TRANSFER">微信</el-radio>
          <el-radio value="OTHER">其他</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="金额（元）">
        <el-input-number v-model="amountYuan" :min="0" :precision="2" :step="10" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="remark" placeholder="选填" />
      </el-form-item>
    </el-form>

    <el-alert v-if="!phone" type="info" :closable="false" show-icon style="margin-bottom: 8px"
      title="不录手机号仅记账，不生成服务记录" />
    <ServiceRecordFields v-model="record" :disabled="!phone" />

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="submitting" :disabled="!canSubmit" @click="doSettle">确认结算</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
// WalkInSettleDialog — 散客快速结算（D29 §3.3）：无预约无卡，手机号选填；
// 不录手机号仅记账（无记录）；沟通确认必勾；CARD 不在收款方式中（后端拒绝）。
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listMembers,
  listServices,
  walkinSettle,
  type RecordPayload,
  type ServiceItem,
} from '../core/api/admin'
import { toFen, yuan } from '../core/format'
import ServiceRecordFields from './ServiceRecordFields.vue'

defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'settled'): void
}>()

const phone = ref('')
const name = ref('')
const serviceId = ref('')
const payMethod = ref<'CASH' | 'WECHAT_TRANSFER' | 'OTHER'>('CASH')
const amountYuan = ref<number>(0)
const remark = ref('')
const services = ref<ServiceItem[]>([])
const submitting = ref(false)
const record = ref<RecordPayload>({
  body_parts: [],
  service_method: '',
  tech_note: '',
  communicated: false,
})

// 手机号联动：填写即显示姓名输入；满 11 位后查询是否老会员（提示将建档）
const checking = ref(false)
const phoneKnown = ref<boolean | null>(null)

const canSubmit = computed(
  () => !!serviceId.value && record.value.communicated && (phone.value.trim() === '' || /^1\d{10}$/.test(phone.value.trim())),
)

function fen(v: number): string {
  return (v / 100).toFixed(v % 100 === 0 ? 0 : 2)
}

function onServiceChange(id: string): void {
  const svc = services.value.find((s) => s.id === id)
  if (svc) amountYuan.value = svc.default_price / 100
}

function onPhoneInput(): void {
  phoneKnown.value = null
  if (/^1\d{10}$/.test(phone.value.trim())) void checkPhone()
}

async function checkPhone(): Promise<void> {
  const p = phone.value.trim()
  phoneKnown.value = null
  if (!/^1\d{10}$/.test(p)) return
  checking.value = true
  try {
    const res = await listMembers(p, 1, 1)
    phoneKnown.value = res.total > 0
  } catch {
    phoneKnown.value = null
  } finally {
    checking.value = false
  }
}

function onOpen(): void {
  phone.value = ''
  name.value = ''
  phoneKnown.value = null
  checking.value = false
  serviceId.value = ''
  payMethod.value = 'CASH'
  amountYuan.value = 0
  remark.value = ''
  record.value = { body_parts: [], service_method: '', tech_note: '', communicated: false }
  void loadServices()
}

async function loadServices(): Promise<void> {
  try {
    services.value = ((await listServices()) ?? []).filter((s) => s.status === 'ACTIVE')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载服务项目失败')
  }
}

async function doSettle(): Promise<void> {
  if (!serviceId.value) {
    ElMessage.warning('请选择服务项目')
    return
  }
  if (!record.value.communicated) {
    ElMessage.warning('请先勾选「服务前已完成沟通」')
    return
  }
  const p = phone.value.trim()
  if (p && !/^1\d{10}$/.test(p)) {
    ElMessage.warning('手机号需为 1 开头的 11 位数字，或留空')
    return
  }
  submitting.value = true
  try {
    const res = await walkinSettle({
      phone: p || undefined,
      name: name.value.trim() || undefined,
      service_id: serviceId.value,
      pay_method: payMethod.value,
      amount: toFen(String(amountYuan.value)),
      remark: remark.value || undefined,
      body_parts: record.value.body_parts,
      service_method: record.value.service_method,
      tech_note: record.value.tech_note,
      communicated: record.value.communicated === true,
    })
    const parts: string[] = [`收款成功 ${yuan(res.payment.amount)}`]
    if (res.member_created) parts.push('已创建会员档案')
    if (res.record_skipped) parts.push('已记账（未关联会员）')
    ElMessage.success(parts.join('；'))
    emit('update:modelValue', false)
    emit('settled')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '结算失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.sub-hint {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}
.sub-hint.new {
  color: #e6a23c;
}
</style>

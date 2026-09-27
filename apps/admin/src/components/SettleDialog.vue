<template>
  <el-dialog :model-value="modelValue" :title="`结算 — ${appointment?.appointment_no ?? ''}`"
    width="min(520px, 94vw)" @update:model-value="$emit('update:modelValue', $event)" @open="onOpen">
    <el-tabs v-model="tab">
      <el-tab-pane label="次卡核销" name="redeem">
        <p class="hint" v-if="service">
          项目：{{ service.service_name_snapshot }} · 扣 1 次 ·
          顾客：{{ memberName || appointment?.member_id }}
        </p>
        <el-empty v-if="cards.length === 0 && !loading" description="该顾客暂无可用会员卡" :image-size="70" />
        <el-table v-else :data="cards" size="small" v-loading="loading" @row-dblclick="doRedeem">
          <el-table-column prop="card_template_id" label="模板" width="120">
            <template #default="{ row }">{{ templateName(row.card_template_id) }}</template>
          </el-table-column>
          <el-table-column label="剩余 / 总次数" width="120">
            <template #default="{ row }">{{ row.remaining_count }} / {{ row.total_count }}</template>
          </el-table-column>
          <el-table-column label="有效期至" width="120">
            <template #default="{ row }">{{ row.valid_until ?? '永久' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90">
            <template #default="{ row }">
              <el-button size="small" type="primary" :disabled="row.remaining_count < 1"
                @click="doRedeem(row)">核销</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="现金 / 转账收款" name="pay">
        <el-form label-width="90px">
          <el-form-item label="收款方式">
            <el-radio-group v-model="payMethod">
              <el-radio value="WECHAT_TRANSFER">微信转账</el-radio>
              <el-radio value="CASH">现金</el-radio>
              <el-radio value="OTHER">其他</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="金额（元）">
            <el-input-number v-model="payAmountYuan" :min="0" :precision="2" :step="10" />
          </el-form-item>
          <el-form-item label="转账单号">
            <el-input v-model="payRef" placeholder="微信转账单号（选填）" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="payRemark" placeholder="选填" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="paying" @click="doPay">确认收款</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listMemberCards,
  listCardTemplates,
  redeemCard,
  settlePayment,
  type Appointment,
  type AppointmentService,
  type MemberCard,
} from '../core/api/admin'
import { toFen, yuan } from '../core/format'

const props = defineProps<{
  modelValue: boolean
  appointment: Appointment | null
  service: AppointmentService | null
  memberName?: string
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'settled'): void
}>()

const tab = ref('redeem')
const cards = ref<MemberCard[]>([])
const templates = ref<Record<string, string>>({})
const loading = ref(false)
const paying = ref(false)
const payMethod = ref<'WECHAT_TRANSFER' | 'CASH' | 'OTHER'>('WECHAT_TRANSFER')
const payAmountYuan = ref<number>(0)
const payRef = ref('')
const payRemark = ref('')

function templateName(id: string): string {
  return templates.value[id] ?? id.slice(0, 8)
}

async function onOpen() {
  tab.value = 'redeem'
  const apt = props.appointment
  if (!apt) return
  payAmountYuan.value = props.service ? props.service.price_snapshot / 100 : 0
  payRef.value = ''
  payRemark.value = ''
  loading.value = true
  try {
    const [c, t] = await Promise.all([listMemberCards(apt.member_id), listCardTemplates()])
    templates.value = Object.fromEntries((t ?? []).map((x) => [x.id, x.name]))
    cards.value = (c ?? []).filter((x) => x.status === 'ACTIVE')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载会员卡失败')
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) void onOpen()
  },
)

async function doRedeem(row: MemberCard) {
  const apt = props.appointment
  if (!apt) return
  try {
    await redeemCard(apt.id, row.id)
    ElMessage.success('核销成功')
    emit('update:modelValue', false)
    emit('settled')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '核销失败')
  }
}

async function doPay() {
  const apt = props.appointment
  if (!apt) return
  paying.value = true
  try {
    await settlePayment(apt.id, {
      method: payMethod.value,
      amount: toFen(String(payAmountYuan.value)),
      reference_no: payRef.value,
      remark: payRemark.value,
    })
    ElMessage.success(`收款成功 ${yuan(toFen(String(payAmountYuan.value)))}`)
    emit('update:modelValue', false)
    emit('settled')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '收款失败')
  } finally {
    paying.value = false
  }
}
</script>

<style scoped>
.hint {
  color: #606266;
  font-size: 13px;
  margin: 0 0 10px;
}
</style>

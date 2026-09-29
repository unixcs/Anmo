<template>
  <el-card shadow="never">
    <el-tabs v-model="tab" @tab-change="load">
      <el-tab-pane label="收款记录" name="payments">
        <div class="tab-head">
          <el-radio-group v-model="payStatus" size="small" @change="load">
            <el-radio-button value="">全部</el-radio-button>
            <el-radio-button value="VALID">有效</el-radio-button>
            <el-radio-button value="VOIDED">已作废</el-radio-button>
          </el-radio-group>
          <span class="hint">最近 200 条</span>
        </div>
        <template v-if="isMobile">
          <el-empty v-if="payments.length === 0" description="暂无收款记录" :image-size="70" />
          <div v-for="row in payments" :key="row.id" class="r-card">
            <div class="r-top">
              <b>{{ memberName(row.member_id) }}</b>
              <el-tag size="small" :type="row.status === 'VALID' ? 'success' : 'danger'">
                {{ row.status === 'VALID' ? '有效' : '已作废' }}
              </el-tag>
            </div>
            <div class="r-line">
              <span>{{ PAY_METHOD_TEXT[row.method] ?? row.method }} · <b class="r-amount">{{ yuan(row.amount) }}</b></span>
              <span class="r-sub">{{ row.appointment_id ?? '散客' }}</span>
            </div>
            <div v-if="row.remark || row.reference_no" class="r-sub">备注：{{ row.remark || row.reference_no }}</div>
          </div>
        </template>
        <el-table v-else :data="payments" size="default" v-loading="loading">
          <el-table-column label="会员" min-width="130">
            <template #default="{ row }">{{ memberName(row.member_id) }}</template>
          </el-table-column>
          <el-table-column label="方式" width="110">
            <template #default="{ row }">{{ PAY_METHOD_TEXT[row.method] ?? row.method }}</template>
          </el-table-column>
          <el-table-column label="金额" width="100">
            <template #default="{ row }">{{ yuan(row.amount) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="row.status === 'VALID' ? 'success' : 'danger'">
                {{ row.status === 'VALID' ? '有效' : '已作废' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reference_no" label="单号" min-width="120" show-overflow-tooltip />
          <el-table-column prop="remark" label="备注" min-width="110" show-overflow-tooltip />
          <el-table-column label="预约" width="120" show-overflow-tooltip>
            <template #default="{ row }">{{ row.appointment_id ?? '散客' }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="核销记录" name="redemptions">
        <div class="tab-head">
          <el-radio-group v-model="rdmStatus" size="small" @change="load">
            <el-radio-button value="">全部</el-radio-button>
            <el-radio-button value="SUCCESS">成功</el-radio-button>
            <el-radio-button value="REVERSED">已撤销</el-radio-button>
          </el-radio-group>
          <span class="hint">最近 200 条</span>
        </div>
        <template v-if="isMobile">
          <el-empty v-if="redemptions.length === 0" description="暂无核销记录" :image-size="70" />
          <div v-for="row in redemptions" :key="row.id" class="r-card">
            <div class="r-top">
              <b>{{ memberName(row.member_id) }}</b>
              <el-tag size="small" :type="row.status === 'SUCCESS' ? 'success' : 'info'">
                {{ row.status === 'SUCCESS' ? '成功' : '已撤销' }}
              </el-tag>
            </div>
            <div class="r-line">
              <span>扣 {{ row.quantity }} 次 · {{ row.before_count }} → {{ row.after_count }}</span>
              <el-button v-if="row.status === 'SUCCESS'" size="small" type="danger"
                @click="doReverse(row)">撤销</el-button>
            </div>
            <div class="r-sub">{{ row.appointment_id ?? '散客核销' }}</div>
          </div>
        </template>
        <el-table v-else :data="redemptions" size="default" v-loading="loading">
          <el-table-column label="会员" min-width="130">
            <template #default="{ row }">{{ memberName(row.member_id) }}</template>
          </el-table-column>
          <el-table-column label="扣次" width="70">
            <template #default="{ row }">-{{ row.quantity }}</template>
          </el-table-column>
          <el-table-column label="余量变化" width="100">
            <template #default="{ row }">{{ row.before_count }} → {{ row.after_count }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="row.status === 'SUCCESS' ? 'success' : 'info'">
                {{ row.status === 'SUCCESS' ? '成功' : '已撤销' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="预约" min-width="120" show-overflow-tooltip>
            <template #default="{ row }">{{ row.appointment_id ?? '散客' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status === 'SUCCESS'" size="small" type="danger"
                @click="doReverse(row)">撤销</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listMembers,
  listPayments,
  listRedemptions,
  reverseRedemption,
  type Payment,
  type Redemption,
} from '../core/api/admin'
import { PAY_METHOD_TEXT, yuan } from '../core/format'
import { useIsMobile } from '../core/useMedia'

const isMobile = useIsMobile()
const tab = ref('payments')
const payStatus = ref('')
const rdmStatus = ref('')
const payments = ref<Payment[]>([])
const redemptions = ref<Redemption[]>([])
const loading = ref(false)
const memberMap = ref<Record<string, string>>({})

function memberName(id: string | null): string {
  if (!id) return '未登记' // D29：散客快速结算未录手机号，payment.member_id 为 NULL
  return memberMap.value[id] ?? id.slice(0, 8)
}

async function load() {
  loading.value = true
  try {
    const jobs: Promise<unknown>[] = [
      tab.value === 'payments' ? listPayments(payStatus.value || undefined) : listRedemptions(rdmStatus.value || undefined),
    ]
    if (Object.keys(memberMap.value).length === 0) jobs.push(listMembers('', 1, 100))
    const [first, m] = (await Promise.all(jobs)) as [unknown, { data: { id: string; name: string; phone: string }[] } | undefined]
    if (tab.value === 'payments') payments.value = (first as Payment[]) ?? []
    else redemptions.value = (first as Redemption[]) ?? []
    if (m) memberMap.value = Object.fromEntries(m.data.map((x) => [x.id, `${x.name}（${x.phone}）`]))
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function doReverse(row: Redemption) {
  let reason = ''
  try {
    const res = await ElMessageBox.prompt('请输入撤销原因（如：误核销）。撤销后次数退回、CARD 收款作废。', '撤销核销', {
      inputPlaceholder: '撤销原因（选填，≤500字）',
    })
    reason = res.value.trim()
  } catch {
    return
  }
  try {
    await reverseRedemption(row.id, reason)
    ElMessage.success('已撤销，次数已退回')
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '撤销失败')
  }
}

onMounted(load)
</script>

<style scoped>
.tab-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.hint {
  color: #909399;
  font-size: 12px;
}
/* 手机卡片流（§25：操作按钮进入可视区，无横向滚动） */
.r-card {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 12px;
  margin-bottom: 10px;
  background: #fff;
}
.r-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.r-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  color: #606266;
}
.r-amount {
  color: #67c23a;
}
.r-sub {
  color: #b0b3b8;
  font-size: 12px;
  margin-top: 4px;
}
</style>

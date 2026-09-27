<template>
  <div>
    <el-card shadow="never" class="block">
      <div class="stat-head">
        <span class="stat-date">{{ summary?.date ?? todayStr() }} 今日概览</span>
        <el-date-picker v-model="date" type="date" value-format="YYYY-MM-DD" :clearable="false"
          style="width: 150px" @change="refresh" />
      </div>
      <div class="stats">
        <div class="stat"><b>{{ summary?.total ?? 0 }}</b><span>预约总数</span></div>
        <div class="stat warn"><b>{{ summary?.pending_confirm ?? 0 }}</b><span>待确认</span></div>
        <div class="stat info"><b>{{ summary?.confirmed ?? 0 }}</b><span>已确认</span></div>
        <div class="stat ok"><b>{{ summary?.in_service ?? 0 }}</b><span>服务中</span></div>
        <div class="stat"><b>{{ summary?.completed ?? 0 }}</b><span>已完成</span></div>
        <div class="stat bad"><b>{{ summary?.cancelled ?? 0 }}</b><span>已取消</span></div>
        <div class="stat bad"><b>{{ summary?.no_show ?? 0 }}</b><span>未到店</span></div>
      </div>
    </el-card>

    <el-card shadow="never" class="block" header="今日预约">
      <el-table :data="today" v-loading="loading" size="default">
        <el-table-column label="时间" width="110">
          <template #default="{ row }">{{ fmtTime(row.scheduled_start).slice(11) }}</template>
        </el-table-column>
        <el-table-column label="顾客" width="110">
          <template #default="{ row }">{{ memberName(row.member_id) }}</template>
        </el-table-column>
        <el-table-column label="项目" min-width="140">
          <template #default="{ row }">
            {{ row.service ? row.service.service_name_snapshot : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="APT_STATUS_TAG[row.status]" size="small">{{ APT_STATUS_TEXT[row.status] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="280">
          <template #default="{ row }">
            <el-button v-if="row.status === 'PENDING_CONFIRM'" size="small" type="primary"
              @click="act.confirmApt(row.id)">确认</el-button>
            <el-button v-if="row.status === 'CONFIRMED'" size="small" type="primary"
              @click="act.startApt(row.id)">开始服务</el-button>
            <el-button v-if="row.status === 'CONFIRMED'" size="small" @click="act.noShowApt(row.id)">未到店</el-button>
            <el-button v-if="row.status === 'IN_SERVICE'" size="small" type="success"
              @click="act.completeApt(row.id)">完成</el-button>
            <el-button v-if="['IN_SERVICE', 'COMPLETED'].includes(row.status)" size="small" type="warning"
              @click="openSettle(row)">结算</el-button>
            <el-button v-if="['PENDING_CONFIRM', 'CONFIRMED'].includes(row.status)" size="small"
              @click="act.rescheduleApt(row.id)">改期</el-button>
            <el-button v-if="['PENDING_CONFIRM', 'CONFIRMED'].includes(row.status)" size="small" type="danger"
              @click="act.cancelApt(row.id)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="block" header="结算工作台">
      <el-empty v-if="workbench.length === 0" description="今日暂无待结算预约" :image-size="70" />
      <el-table v-else :data="workbench" size="default">
        <el-table-column label="预约号" width="170">
          <template #default="{ row }">{{ row.appointment.appointment_no }}</template>
        </el-table-column>
        <el-table-column prop="member_name" label="会员" width="110" />
        <el-table-column label="项目" min-width="130">
          <template #default="{ row }">
            {{ row.service ? row.service.service_name_snapshot : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="APT_STATUS_TAG[row.appointment.status]" size="small">
              {{ APT_STATUS_TEXT[row.appointment.status] }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="收款" min-width="160">
          <template #default="{ row }">
            <span v-if="row.payment">
              {{ PAY_METHOD_TEXT[row.payment.method] }} · {{ yuan(row.payment.amount) }}
              <el-tag size="small" type="success">已收</el-tag>
            </span>
            <el-tag v-else size="small" type="info">未收款</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button size="small" type="warning" @click="openSettle(row.appointment, row.service)">
              {{ row.payment ? '查看/补收' : '去结算' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <SettleDialog v-model="settleVisible" :appointment="settleApt" :service="settleSvc"
      :member-name="settleApt ? memberName(settleApt.member_id) : ''" @settled="refresh" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  getToday,
  getWorkbench,
  listMembers,
  type Appointment,
  type AppointmentService,
  type TodayAppointment,
  type TodaySummary,
  type WorkbenchCard,
} from '../core/api/admin'
import { APT_STATUS_TAG, APT_STATUS_TEXT, PAY_METHOD_TEXT, fmtTime, todayStr, yuan } from '../core/format'
import { useAptActions } from '../components/aptActions'
import SettleDialog from '../components/SettleDialog.vue'

const date = ref(todayStr())
const summary = ref<TodaySummary | null>(null)
const today = ref<TodayAppointment[]>([])
const workbench = ref<WorkbenchCard[]>([])
const loading = ref(false)
const memberMap = ref<Record<string, string>>({})

const settleVisible = ref(false)
const settleApt = ref<Appointment | null>(null)
const settleSvc = ref<AppointmentService | null>(null)

const act = useAptActions(refresh)

function memberName(id: string): string {
  return memberMap.value[id] ?? id.slice(0, 8)
}

function openSettle(apt: Appointment, svc: AppointmentService | null = null) {
  settleApt.value = apt
  settleSvc.value = svc
  settleVisible.value = true
}

async function refresh() {
  loading.value = true
  try {
    const [t, w, m] = await Promise.all([
      getToday(date.value),
      getWorkbench(date.value),
      listMembers('', 1, 100),
    ])
    summary.value = t.summary
    today.value = t.appointments ?? []
    workbench.value = w.cards ?? []
    if (w.summary) summary.value = t.summary
    memberMap.value = Object.fromEntries(m.data.map((x) => [x.id, `${x.name}（${x.phone}）`]))
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

onMounted(refresh)
</script>

<style scoped>
.block {
  margin-bottom: 14px;
}
.stat-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.stat-date {
  font-weight: 600;
}
.stats {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.stat {
  flex: 1;
  min-width: 90px;
  background: #f5f7fa;
  border-radius: 8px;
  padding: 12px 0;
  text-align: center;
}
.stat b {
  display: block;
  font-size: 22px;
}
.stat span {
  color: #909399;
  font-size: 12px;
}
.stat.warn b {
  color: #e6a23c;
}
.stat.ok b {
  color: #67c23a;
}
.stat.info b {
  color: #409eff;
}
.stat.bad b {
  color: #f56c6c;
}
</style>

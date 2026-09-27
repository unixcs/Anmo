<template>
  <div>
    <el-card shadow="never" class="block">
      <div class="stat-head">
        <span class="stat-date">{{ summary?.date ?? todayStr() }} 今日概览</span>
        <div class="stat-ops">
          <el-button type="primary" @click="scanVisible = true">扫码核销</el-button>
          <el-date-picker v-model="date" type="date" value-format="YYYY-MM-DD" :clearable="false"
            style="width: 150px" @change="refresh" />
        </div>
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
      <!-- 手机：卡片流 -->
      <template v-if="isMobile">
        <el-empty v-if="today.length === 0" description="今日暂无预约" :image-size="70" />
        <div v-for="row in today" :key="row.id" class="apt-card">
          <div class="apt-top">
            <span class="apt-time">{{ fmtTime(row.scheduled_start).slice(11) }}</span>
            <el-tag :type="APT_STATUS_TAG[row.status]" size="small">{{ APT_STATUS_TEXT[row.status] }}</el-tag>
          </div>
          <div class="apt-main">
            <span class="apt-member">{{ memberName(row.member_id) }}</span>
            <span class="apt-svc">{{ row.service ? row.service.service_name_snapshot : '-' }}</span>
          </div>
          <div class="apt-btns">
            <AptActionButtons :row="row" :refresh="refresh" @settle="openSettle(row)" />
          </div>
        </div>
      </template>
      <!-- 桌面：表格 -->
      <el-table v-else :data="today" v-loading="loading" size="default">
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
            <AptActionButtons :row="row" :refresh="refresh" @settle="openSettle(row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="block" header="结算工作台">
      <el-empty v-if="workbench.length === 0" description="今日暂无待结算预约" :image-size="70" />
      <template v-else-if="isMobile">
        <div v-for="(row, i) in workbench" :key="i" class="apt-card">
          <div class="apt-top">
            <span class="apt-member">{{ row.member_name }}</span>
            <el-tag :type="APT_STATUS_TAG[row.appointment.status]" size="small">
              {{ APT_STATUS_TEXT[row.appointment.status] }}
            </el-tag>
          </div>
          <div class="apt-main">
            <span class="apt-svc">{{ row.service ? row.service.service_name_snapshot : '-' }}</span>
            <span v-if="row.payment" class="apt-pay">{{ PAY_METHOD_TEXT[row.payment.method] }} {{ yuan(row.payment.amount) }} 已收</span>
            <el-tag v-else size="small" type="info">未收款</el-tag>
          </div>
          <div class="apt-btns">
            <el-button size="small" type="warning" @click="openSettle(row.appointment, row.service)">
              {{ row.payment ? '查看/补收' : '去结算' }}
            </el-button>
          </div>
        </div>
      </template>
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
    <ScanRedeemDialog v-model="scanVisible" @settled="refresh" />
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
import { useIsMobile } from '../core/useMedia'
import AptActionButtons from '../components/AptActionButtons.vue'
import SettleDialog from '../components/SettleDialog.vue'
import ScanRedeemDialog from '../components/ScanRedeemDialog.vue'

const isMobile = useIsMobile()
const date = ref(todayStr())
const scanVisible = ref(false)
const summary = ref<TodaySummary | null>(null)
const today = ref<TodayAppointment[]>([])
const workbench = ref<WorkbenchCard[]>([])
const loading = ref(false)
const memberMap = ref<Record<string, string>>({})

const settleVisible = ref(false)
const settleApt = ref<Appointment | null>(null)
const settleSvc = ref<AppointmentService | null>(null)

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
  flex-wrap: wrap;
  gap: 8px;
}
.stat-date {
  font-weight: 600;
}
.stat-ops {
  display: flex;
  gap: 8px;
  align-items: center;
}
.stats {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.stat {
  flex: 1;
  min-width: 76px;
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
/* 手机卡片流 */
.apt-card {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 12px;
  margin-bottom: 10px;
  background: #fff;
}
.apt-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.apt-time {
  font-weight: 600;
  color: #303133;
}
.apt-main {
  margin: 8px 0 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.apt-member {
  font-size: 14px;
  color: #303133;
}
.apt-svc {
  color: #606266;
  font-size: 13px;
}
.apt-pay {
  color: #67c23a;
  font-size: 13px;
}
.apt-btns {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.apt-btns :deep(.el-button) {
  margin-left: 0;
}
</style>

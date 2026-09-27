<template>
  <el-card shadow="never">
    <div class="toolbar">
      <el-date-picker v-model="date" type="date" value-format="YYYY-MM-DD" placeholder="全部日期"
        style="width: 150px" clearable @change="load" />
      <el-select v-model="status" placeholder="全部状态" clearable style="width: 140px" @change="load">
        <el-option v-for="(text, key) in APT_STATUS_TEXT" :key="key" :label="text" :value="key" />
      </el-select>
      <el-button type="primary" @click="load">查询</el-button>
      <el-button type="warning" plain @click="closureVisible = true">闭店设置</el-button>
      <span class="spacer" />
      <span class="total">共 {{ total }} 条</span>
    </div>

    <!-- 手机：卡片流 -->
    <template v-if="isMobile">
      <el-empty v-if="rows.length === 0" description="暂无预约" :image-size="70" />
      <div v-for="row in rows" :key="row.id" class="apt-card">
        <div class="apt-top">
          <span class="apt-time">{{ timeLabel(row) }}</span>
          <el-tag v-if="row.slot_type === 'HALF_DAY'" size="small" type="warning">{{ row.day_part === 'AM' ? '上午到店' : '下午到店' }}</el-tag>
          <el-tag :type="APT_STATUS_TAG[row.status]" size="small">{{ APT_STATUS_TEXT[row.status] }}</el-tag>
        </div>
        <div class="apt-main">
          <span class="apt-member">{{ memberName(row.member_id) }}</span>
          <span class="apt-no">{{ row.appointment_no }}</span>
        </div>
        <div v-if="row.customer_note" class="apt-note">备注：{{ row.customer_note }}</div>
        <div class="apt-btns">
          <AptActionButtons :row="row" :refresh="load" @settle="openSettle(row)" @reschedule="openReschedule(row)" />
        </div>
      </div>
    </template>
    <!-- 桌面：表格 -->
    <el-table v-else :data="rows" v-loading="loading" size="default">
      <el-table-column prop="appointment_no" label="预约号" width="170" />
      <el-table-column label="顾客" width="150">
        <template #default="{ row }">{{ memberName(row.member_id) }}</template>
      </el-table-column>
      <el-table-column label="时间" width="180">
        <template #default="{ row }">
          {{ timeLabel(row) }}
          <el-tag v-if="row.slot_type === 'HALF_DAY'" size="small" type="warning">
            {{ row.day_part === 'AM' ? '上午到店' : '下午到店' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="结束时间" width="150">
        <template #default="{ row }">{{ fmtTime(row.scheduled_end) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="APT_STATUS_TAG[row.status]" size="small">{{ APT_STATUS_TEXT[row.status] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="customer_note" label="顾客备注" min-width="120" show-overflow-tooltip />
      <el-table-column label="操作" min-width="300" fixed="right">
        <template #default="{ row }">
          <AptActionButtons :row="row" :refresh="load" @settle="openSettle(row)" @reschedule="openReschedule(row)" />
        </template>
      </el-table-column>
    </el-table>

    <el-pagination class="pager" background layout="prev, pager, next, total" :total="total"
      :page-size="perPage" :current-page="page" @current-change="onPage" />
  </el-card>

  <SettleDialog v-model="settleVisible" :appointment="settleApt" :service="null"
    :member-name="settleApt ? memberName(settleApt.member_id) : ''" @settled="load" />
  <RescheduleDialog v-model="rescheduleVisible" :apt-id="rescheduleId" :on-done="load" />
  <ClosureDialog v-model="closureVisible" />
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  listAppointments,
  listMembers,
  type Appointment,
} from '../core/api/admin'
import { APT_STATUS_TAG, APT_STATUS_TEXT, fmtTime } from '../core/format'
import { useIsMobile } from '../core/useMedia'
import AptActionButtons from '../components/AptActionButtons.vue'
import SettleDialog from '../components/SettleDialog.vue'
import RescheduleDialog from '../components/RescheduleDialog.vue'
import ClosureDialog from '../components/ClosureDialog.vue'

const isMobile = useIsMobile()
const date = ref('')
const status = ref('')
const page = ref(1)
const perPage = 20
const total = ref(0)
const rows = ref<Appointment[]>([])
const loading = ref(false)
const memberMap = ref<Record<string, string>>({})

const settleVisible = ref(false)
const settleApt = ref<Appointment | null>(null)
const rescheduleVisible = ref(false)
const rescheduleId = ref('')
const closureVisible = ref(false)

function timeLabel(a: Appointment): string {
  return a.slot_type === 'HALF_DAY'
    ? `${fmtTime(a.scheduled_start).slice(5, 10)} ${a.day_part === 'AM' ? '上午' : '下午'}`
    : fmtTime(a.scheduled_start).slice(5, 16)
}

function openReschedule(apt: Appointment) {
  rescheduleId.value = apt.id
  rescheduleVisible.value = true
}

function memberName(id: string): string {
  return memberMap.value[id] ?? id.slice(0, 8)
}

function openSettle(apt: Appointment) {
  settleApt.value = apt
  settleVisible.value = true
}

function onPage(p: number) {
  page.value = p
  void load()
}

async function load() {
  loading.value = true
  try {
    const [res, m] = await Promise.all([
      listAppointments({ status: status.value || undefined, date: date.value || undefined, page: page.value, per_page: perPage }),
      Object.keys(memberMap.value).length === 0 ? listMembers('', 1, 100) : Promise.resolve(null),
    ])
    rows.value = res.data
    total.value = res.total
    if (m) memberMap.value = Object.fromEntries(m.data.map((x) => [x.id, `${x.name}（${x.phone}）`]))
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  align-items: center;
  flex-wrap: wrap;
}
.spacer {
  flex: 1;
}
.total {
  color: #909399;
  font-size: 13px;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
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
  margin: 8px 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.apt-member {
  font-size: 14px;
  color: #303133;
}
.apt-no {
  color: #b0b3b8;
  font-size: 12px;
}
.apt-note {
  color: #909399;
  font-size: 12px;
  margin-bottom: 8px;
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

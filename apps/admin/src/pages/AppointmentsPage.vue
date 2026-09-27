<template>
  <el-card shadow="never">
    <div class="toolbar">
      <el-date-picker v-model="date" type="date" value-format="YYYY-MM-DD" placeholder="全部日期"
        style="width: 150px" clearable @change="load" />
      <el-select v-model="status" placeholder="全部状态" clearable style="width: 140px" @change="load">
        <el-option v-for="(text, key) in APT_STATUS_TEXT" :key="key" :label="text" :value="key" />
      </el-select>
      <el-button type="primary" @click="load">查询</el-button>
      <span class="spacer" />
      <span class="total">共 {{ total }} 条</span>
    </div>

    <el-table :data="rows" v-loading="loading" size="default">
      <el-table-column prop="appointment_no" label="预约号" width="170" />
      <el-table-column label="顾客" width="150">
        <template #default="{ row }">{{ memberName(row.member_id) }}</template>
      </el-table-column>
      <el-table-column label="开始时间" width="150">
        <template #default="{ row }">{{ fmtTime(row.scheduled_start) }}</template>
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

    <el-pagination class="pager" background layout="prev, pager, next, total" :total="total"
      :page-size="perPage" :current-page="page" @current-change="onPage" />
  </el-card>

  <SettleDialog v-model="settleVisible" :appointment="settleApt" :service="null"
    :member-name="settleApt ? memberName(settleApt.member_id) : ''" @settled="load" />
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  listAppointments,
  listMembers,
  type Appointment,
} from '../core/api/admin'
import { APT_STATUS_TAG, APT_STATUS_TEXT, fmtTime } from '../core/format'
import { useAptActions } from '../components/aptActions'
import SettleDialog from '../components/SettleDialog.vue'

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
const act = useAptActions(load)

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
</style>

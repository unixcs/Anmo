<template>
  <div>
    <el-card shadow="never" class="block">
      <div class="head">
        <span class="hint">运营洞察：低余额 / 即将过期 / 睡眠会员（用于提醒顾客续卡复购）。每日执行一次"日常运维"清理过期卡并生成快照。</span>
        <el-button size="small" type="primary" :loading="sweeping" @click="runDaily">执行日常运维</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="block" header="低余额会员（剩余次数不足）">
      <el-table :data="insights.LOW_BALANCE ?? []" size="small" v-loading="loading">
        <el-table-column prop="member_name" label="会员" min-width="110" />
        <el-table-column prop="member_no" label="会员号" width="150" />
        <el-table-column prop="detail" label="情况" min-width="180" />
      </el-table>
    </el-card>

    <el-card shadow="never" class="block" header="即将过期的会员卡">
      <el-table :data="insights.EXPIRING ?? []" size="small" v-loading="loading">
        <el-table-column prop="member_name" label="会员" min-width="110" />
        <el-table-column prop="member_no" label="会员号" width="150" />
        <el-table-column prop="detail" label="情况" min-width="180" />
      </el-table>
    </el-card>

    <el-card shadow="never" class="block" header="睡眠会员（长期未到店）">
      <el-table :data="insights.DORMANT ?? []" size="small" v-loading="loading">
        <el-table-column prop="member_name" label="会员" min-width="110" />
        <el-table-column prop="member_no" label="会员号" width="150" />
        <el-table-column prop="detail" label="情况" min-width="180" />
      </el-table>
    </el-card>

    <el-card shadow="never" header="操作日志">
      <div class="tab-head">
        <el-input v-model="logAction" placeholder="按动作过滤，如 POST /admin/cards" clearable
          style="width: 260px" @keyup.enter="loadLogs" />
        <el-button size="small" @click="loadLogs">查询</el-button>
      </div>
      <el-table :data="logs" size="small" v-loading="logLoading" max-height="420">
        <el-table-column prop="action" label="动作" min-width="200" show-overflow-tooltip />
        <el-table-column prop="actor_type" label="操作者" width="90" />
        <el-table-column prop="target_type" label="对象类型" width="110" />
        <el-table-column prop="target_id" label="对象ID" width="130" show-overflow-tooltip />
        <el-table-column prop="detail" label="详情" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="120" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getInsights,
  listLogs,
  runDailyOps,
  type InsightItem,
  type LoggedOperation,
} from '../core/api/admin'

const insights = ref<Record<'LOW_BALANCE' | 'DORMANT' | 'EXPIRING', InsightItem[]>>({
  LOW_BALANCE: [],
  DORMANT: [],
  EXPIRING: [],
})
const logs = ref<LoggedOperation[]>([])
const loading = ref(false)
const logLoading = ref(false)
const sweeping = ref(false)
const logAction = ref('')

async function loadInsights() {
  loading.value = true
  try {
    insights.value = await getInsights()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载洞察失败')
  } finally {
    loading.value = false
  }
}

async function loadLogs() {
  logLoading.value = true
  try {
    logs.value = (await listLogs(logAction.value.trim() || undefined)) ?? []
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载日志失败')
  } finally {
    logLoading.value = false
  }
}

async function runDaily() {
  sweeping.value = true
  try {
    const res = await runDailyOps()
    ElMessage.success(`完成：清理过期卡 ${res.expired_swept} 张，生成快照 ${res.snapshots} 份`)
    await Promise.all([loadInsights(), loadLogs()])
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '执行失败')
  } finally {
    sweeping.value = false
  }
}

onMounted(() => {
  void loadInsights()
  void loadLogs()
})
</script>

<style scoped>
.block {
  margin-bottom: 14px;
}
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.hint {
  color: #909399;
  font-size: 12px;
}
.tab-head {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}
</style>

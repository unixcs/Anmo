<template>
  <!-- 预约状态操作按钮组（表格列与手机卡片共用，按状态显隐）。
       状态机 2026-09-28 收紧：创建即 WAITING，无确认环节。 -->
  <el-button v-if="row.status === 'WAITING'" size="small" type="primary"
    @click="act.startApt(row.id)">开始服务</el-button>
  <el-button v-if="row.status === 'WAITING'" size="small" @click="act.noShowApt(row.id)">未到店</el-button>
  <el-button v-if="row.status === 'IN_SERVICE'" size="small" type="success"
    @click="act.completeApt(row.id)">完成</el-button>
  <el-button v-if="['IN_SERVICE', 'COMPLETED'].includes(row.status)" size="small" type="warning"
    @click="$emit('settle')">结算</el-button>
  <el-button v-if="row.status === 'WAITING'" size="small"
    @click="$emit('reschedule', row.id)">改期</el-button>
  <el-button v-if="row.status === 'WAITING'" size="small" type="danger"
    @click="act.cancelApt(row.id)">取消</el-button>
</template>

<script setup lang="ts">
import { useAptActions } from './aptActions'
import type { Appointment } from '../core/api/admin'

const props = defineProps<{ row: Appointment; refresh: () => void }>()
defineEmits<{ (e: 'settle'): void; (e: 'reschedule', id: string): void }>()

const act = useAptActions(() => props.refresh())
</script>

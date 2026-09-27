<template>
  <!-- 预约状态操作按钮组（表格列与手机卡片共用，按状态显隐） -->
  <el-button v-if="row.status === 'PENDING_CONFIRM'" size="small" type="primary"
    @click="act.confirmApt(row.id)">确认</el-button>
  <el-button v-if="row.status === 'CONFIRMED'" size="small" type="primary"
    @click="act.startApt(row.id)">开始服务</el-button>
  <el-button v-if="row.status === 'CONFIRMED'" size="small" @click="act.noShowApt(row.id)">未到店</el-button>
  <el-button v-if="row.status === 'IN_SERVICE'" size="small" type="success"
    @click="act.completeApt(row.id)">完成</el-button>
  <el-button v-if="['IN_SERVICE', 'COMPLETED'].includes(row.status)" size="small" type="warning"
    @click="$emit('settle')">结算</el-button>
  <el-button v-if="['PENDING_CONFIRM', 'CONFIRMED'].includes(row.status)" size="small"
    @click="act.rescheduleApt(row.id)">改期</el-button>
  <el-button v-if="['PENDING_CONFIRM', 'CONFIRMED'].includes(row.status)" size="small" type="danger"
    @click="act.cancelApt(row.id)">取消</el-button>
</template>

<script setup lang="ts">
import { useAptActions } from './aptActions'
import type { Appointment } from '../core/api/admin'

const props = defineProps<{ row: Appointment; refresh: () => void }>()
defineEmits<{ (e: 'settle'): void }>()

const act = useAptActions(() => props.refresh())
</script>

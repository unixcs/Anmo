<script setup lang="ts">
// BRAND-GUIDELINES §4 状态徽标映射（状态词与小程序 format.js 同口径）：
// 待到店=warning、服务中=primary、已完成=success、已取消=muted、未到店=destructive。
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const MAP: Record<string, { variant: string; label: string }> = {
  WAITING: { variant: 'warning', label: '待到店' },
  IN_SERVICE: { variant: 'primary', label: '服务中' },
  COMPLETED: { variant: 'success', label: '已完成' },
  CANCELLED: { variant: 'muted', label: '已取消' },
  NO_SHOW: { variant: 'destructive', label: '未到店' },
  // 状态机收紧前的历史值（旧数据只读展示）
  PENDING_CONFIRM: { variant: 'muted', label: '待确认(历史)' },
  CONFIRMED: { variant: 'muted', label: '已确认(历史)' },
}

const view = computed(() => MAP[props.status] ?? { variant: 'muted', label: props.status })
</script>

<template>
  <span class="badge" :class="view.variant">{{ view.label }}</span>
</template>

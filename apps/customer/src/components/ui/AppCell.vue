<script setup lang="ts">
// BRAND-GUIDELINES §3 Cell：图标 + 文字 + 值 + chevron 的列表行。
// 未传 to 时渲染 button（供 @click 用），传 to 渲染 RouterLink。
import type { IconName } from './AppIcon.vue'
import AppIcon from './AppIcon.vue'

defineProps<{
  icon?: IconName
  title: string
  desc?: string
  value?: string
  to?: string
}>()

const emit = defineEmits<{ click: [] }>()
</script>

<template>
  <component
    :is="to ? 'RouterLink' : 'button'"
    :to="to"
    class="cell"
    @click="emit('click')"
  >
    <span v-if="icon" class="cell-icon"><AppIcon :name="icon" /></span>
    <span class="cell-body">
      <span class="cell-title">{{ title }}</span>
      <span v-if="desc" class="cell-desc">{{ desc }}</span>
      <slot />
    </span>
    <span v-if="value" class="cell-value">{{ value }}</span>
    <span v-if="to" class="chevron"><AppIcon name="chevron-right" :size="16" /></span>
  </component>
</template>

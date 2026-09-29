<script setup lang="ts">
// BRAND-GUIDELINES §3 Empty：图标 + 主句 + 次句 + 可选 CTA。
import type { IconName } from './AppIcon.vue'
import AppIcon from './AppIcon.vue'

defineProps<{
  icon?: IconName
  main: string
  sub?: string
}>()
</script>

<template>
  <div class="empty">
    <span v-if="icon" class="empty-icon"><AppIcon :name="icon" :size="40" :stroke-width="1.4" /></span>
    <span class="empty-main">{{ main }}</span>
    <p v-if="sub" class="empty-sub">{{ sub }}</p>
    <slot />
  </div>
</template>

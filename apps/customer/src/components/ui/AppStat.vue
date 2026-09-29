<script setup lang="ts">
// BRAND-GUIDELINES §3 StatTile：我的页统计块。
defineProps<{ label: string; value: string | number; hint?: string }>()
</script>

<template>
  <div class="stat">
    <div class="stat-label">{{ label }}</div>
    <div class="stat-value">{{ value }}</div>
    <div v-if="hint" style="font: var(--font-caption); color: var(--muted-foreground); margin-top: 1px">
      {{ hint }}
    </div>
    <slot />
  </div>
</template>

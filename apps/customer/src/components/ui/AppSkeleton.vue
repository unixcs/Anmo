<script setup lang="ts">
// BRAND-GUIDELINES §3 Skeleton：text / card 两型。
withDefaults(defineProps<{ variant?: 'text' | 'card'; width?: string; lines?: number }>(), {
  variant: 'text',
  width: undefined,
  lines: 3,
})
</script>

<template>
  <template v-if="variant === 'card'">
    <div class="skeleton card" style="margin-bottom: 10px" />
    <div class="skeleton card" style="margin-bottom: 10px" />
    <div class="skeleton card" />
  </template>
  <div v-else class="stack" style="display: flex; flex-direction: column; gap: 8px">
    <div
      v-for="i in lines"
      :key="i"
      class="skeleton text"
      :style="{ width: i === lines ? '60%' : width ?? '100%' }"
    />
  </div>
</template>

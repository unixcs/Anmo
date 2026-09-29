<script setup lang="ts">
// BRAND-GUIDELINES §3 Chip：筛选/单选 chip。
defineProps<{ picked?: boolean }>()
const emit = defineEmits<{ click: [] }>()
</script>

<template>
  <button type="button" class="chip pressable" :class="{ picked }" @click="emit('click')">
    <slot />
  </button>
</template>

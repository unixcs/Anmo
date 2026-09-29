<script setup lang="ts">
// BRAND-GUIDELINES §3 Card：白底圆角卡，plain 去阴影。
defineProps<{ plain?: boolean }>()
</script>

<template>
  <div class="card" :class="{ plain }">
    <slot />
  </div>
</template>

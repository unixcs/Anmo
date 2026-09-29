<script setup lang="ts">
// BRAND-GUIDELINES §3 Button：primary/secondary/outline/ghost/destructive。
withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'destructive'
    size?: 'sm' | 'md' | 'lg'
    block?: boolean
    loading?: boolean
    type?: 'button' | 'submit'
  }>(),
  { variant: 'primary', size: 'md', block: false, loading: false, type: 'button' },
)
</script>

<template>
  <button
    :type="type"
    class="btn pressable"
    :class="[variant, { block, lg: size === 'lg', sm: size === 'sm' }]"
    :disabled="loading || undefined"
  >
    <span v-if="loading" class="spinner" />
    <slot />
  </button>
</template>

<script setup lang="ts">
// BRAND-GUIDELINES §3 Dialog：居中确认框（替代 window.confirm）。
withDefaults(
  defineProps<{
    open: boolean
    title: string
    body?: string
    confirmText?: string
    cancelText?: string
    danger?: boolean
    loading?: boolean
  }>(),
  { body: '', confirmText: '确定', cancelText: '取消', danger: false, loading: false },
)

const emit = defineEmits<{ close: []; confirm: [] }>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mask center" @click.self="emit('close')">
      <div class="dialog">
        <h3 class="dialog-title">{{ title }}</h3>
        <p v-if="body" class="dialog-body">{{ body }}</p>
        <div class="dialog-ops">
          <button type="button" class="btn outline" @click="emit('close')">{{ cancelText }}</button>
          <button type="button" class="btn" :class="danger ? 'destructive' : 'primary'" :disabled="loading || undefined" @click="emit('confirm')">
            <span v-if="loading" class="spinner" />
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

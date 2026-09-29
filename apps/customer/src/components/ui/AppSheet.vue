<script setup lang="ts">
// BRAND-GUIDELINES §3 Sheet：底部动作面板容器（改期/选择器）。
defineProps<{ open: boolean; title: string }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mask" @click.self="emit('close')">
      <div class="sheet">
        <div class="sheet-head" style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px">
          <b style="font: var(--font-title)">{{ title }}</b>
          <button
            type="button"
            class="pressable"
            style="border: none; background: var(--muted); color: var(--muted-foreground); width: 30px; height: 30px; border-radius: 999px; display: flex; align-items: center; justify-content: center; cursor: pointer"
            @click="emit('close')"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="m6 6 12 12M18 6 6 18"/></svg>
          </button>
        </div>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

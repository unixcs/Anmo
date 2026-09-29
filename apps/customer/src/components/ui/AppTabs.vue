<script setup lang="ts">
// BRAND-GUIDELINES §3 SegmentedTabs：2–5 项分段选择。
export interface SegOption {
  value: string
  label: string
}

const props = defineProps<{ options: SegOption[]; modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

function pick(v: string): void {
  if (v !== props.modelValue) emit('update:modelValue', v)
}
</script>

<template>
  <div class="seg">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      class="seg-item"
      :class="{ picked: o.value === modelValue }"
      @click="pick(o.value)"
    >
      {{ o.label }}
    </button>
  </div>
</template>

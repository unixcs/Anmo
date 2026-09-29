<script setup lang="ts">
// 双端同源的线性图标（BRAND-GUIDELINES §2.5）：1.8px 描边、圆角端点、24 viewBox。
// weapp 端 components/app-icon 使用同一份 path（CSS mask 实现）。
import { computed } from 'vue'

export type IconName =
  | 'home'
  | 'calendar'
  | 'calendar-x'
  | 'wallet'
  | 'user'
  | 'qr'
  | 'clock'
  | 'history'
  | 'map-pin'
  | 'phone'
  | 'check'
  | 'chevron-right'
  | 'chevron-left'
  | 'x'
  | 'plus'
  | 'pencil'
  | 'scan'
  | 'ticket'
  | 'info'
  | 'alert'
  | 'lock'
  | 'inbox'
  | 'navigation'
  | 'arrow-right'
  | 'sparkles'
  | 'settings'

const PATHS: Record<IconName, string> = {
  home: '<path d="M4 10.5 12 4l8 6.5V20a1 1 0 0 1-1 1h-4.5v-6h-5v6H5a1 1 0 0 1-1-1z"/>',
  calendar:
    '<rect x="4" y="5" width="16" height="16" rx="2"/><path d="M8 3v4M16 3v4M4 10.5h16"/>',
  'calendar-x':
    '<rect x="4" y="5" width="16" height="16" rx="2"/><path d="M8 3v4M16 3v4M4 10.5h16"/><path d="m10 14.5 4 4M14 14.5l-4 4"/>',
  wallet:
    '<rect x="3" y="7" width="18" height="13" rx="2"/><path d="M3 8a2 2 0 0 1 2-2h11"/><path d="M15.5 13.5H21"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4.5 20.5c1.5-3.8 4.3-5.5 7.5-5.5s6 1.7 7.5 5.5"/>',
  qr: '<rect x="4" y="4" width="6" height="6" rx="1"/><rect x="14" y="4" width="6" height="6" rx="1"/><rect x="4" y="14" width="6" height="6" rx="1"/><path d="M17 14v3h3"/><path d="M14 20h.01"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  history:
    '<path d="M3.5 12a8.5 8.5 0 1 0 2.6-6.1L3.5 8.4"/><path d="M3.5 3.5v5h5"/><path d="M12 8v4.5l3 1.8"/>',
  'map-pin':
    '<path d="M19 10.5c0 4.9-5 9.6-7 11-2-1.4-7-6.1-7-11a7 7 0 0 1 14 0z"/><circle cx="12" cy="10.5" r="2.5"/>',
  phone:
    '<path d="M21 16.5v2.6a1.8 1.8 0 0 1-2 1.8 17.9 17.9 0 0 1-7.8-2.8 17.6 17.6 0 0 1-5.4-5.4A17.9 17.9 0 0 1 3 4.9 1.8 1.8 0 0 1 4.8 3h2.6a1.8 1.8 0 0 1 1.8 1.5c.11.86.32 1.7.62 2.5a1.8 1.8 0 0 1-.4 1.9L8.3 10a14.4 14.4 0 0 0 5.4 5.4l1.1-1.1a1.8 1.8 0 0 1 1.9-.4c.8.3 1.64.5 2.5.62a1.8 1.8 0 0 1 1.5 1.8z"/>',
  check: '<path d="m4.5 12.5 5.5 5.5L19.5 6.5"/>',
  'chevron-right': '<path d="m9.5 5 7 7-7 7"/>',
  'chevron-left': '<path d="m14.5 5-7 7 7 7"/>',
  x: '<path d="m6 6 12 12M18 6 6 18"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  pencil: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  scan: '<path d="M4 8V6a2 2 0 0 1 2-2h2M16 4h2a2 2 0 0 1 2 2v2M20 16v2a2 2 0 0 1-2 2h-2M8 20H6a2 2 0 0 1-2-2v-2"/><path d="M4 12h16"/>',
  ticket:
    '<path d="M3 8a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v1.5a2.5 2.5 0 0 0 0 5V16a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1.5a2.5 2.5 0 0 0 0-5z"/><path d="M13.5 6v2.2M13.5 11v2M13.5 15.8V18"/>',
  info: '<circle cx="12" cy="12" r="8.5"/><path d="M12 11v5M12 8v.01"/>',
  alert: '<path d="M12 4 3 19.5h18z"/><path d="M12 10v4M12 16.5v.01"/>',
  lock: '<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>',
  inbox: '<path d="M5 5h14l2 8v6a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-6z"/><path d="M3 13h5l1.5 3h5l1.5-3h5"/>',
  navigation: '<path d="m3.5 11 17-8-8 17-2.4-6.6z"/>',
  'arrow-right': '<path d="M4 12h16M13.5 5.5 20 12l-6.5 6.5"/>',
  sparkles:
    '<path d="M12 4.5 13.6 9l4.4 1.5-4.4 1.5L12 16.5 10.4 12 6 10.5 10.4 9z"/><path d="m18.5 15.5.9 2.1 2.1.9-2.1.9-.9 2.1-.9-2.1-2.1-.9 2.1-.9z"/>',
  settings:
    '<circle cx="12" cy="12" r="3"/><path d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M18.7 5.3l-2.1 2.1M7.4 16.6l-2.1 2.1"/>',
}

const props = withDefaults(
  defineProps<{ name: IconName; size?: number; strokeWidth?: number }>(),
  { size: 20, strokeWidth: 1.8 },
)

const html = computed(() => PATHS[props.name] ?? '')
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    v-html="html"
  />
</template>

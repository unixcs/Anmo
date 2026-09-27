// core — 响应式断点：≤768px 视为手机竖屏（商家端核心场景是手机）
import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

const MOBILE_MAX = 768

export function useIsMobile(): Ref<boolean> {
  const isMobile = ref(window.innerWidth <= MOBILE_MAX)
  const mq = window.matchMedia(`(max-width: ${MOBILE_MAX}px)`)
  const handler = (e: MediaQueryListEvent): void => {
    isMobile.value = e.matches
  }
  onMounted(() => mq.addEventListener('change', handler))
  onBeforeUnmount(() => mq.removeEventListener('change', handler))
  return isMobile
}

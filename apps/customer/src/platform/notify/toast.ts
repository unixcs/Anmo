/**
 * platform/notify — user-facing notifications (DOM belongs to platform).
 * 视觉规格来自 BRAND-GUIDELINES §3 Toast：顶部轻提示、暖墨底、圆角胶囊。
 */
export function notify(message: string): void {
  let host = document.getElementById('anmo-toast')
  if (!host) {
    host = document.createElement('div')
    host.id = 'anmo-toast'
    document.body.appendChild(host)
  }
  const tip = document.createElement('div')
  tip.className = 'anmo-toast'
  tip.textContent = message
  host.appendChild(tip)
  window.setTimeout(() => tip.remove(), 2600)
}

/** 在 index.html 挂载 toast 宿主样式（main.ts 启动时调用一次）。 */
export function registerGlobalStyles(): void {
  const style = document.createElement('style')
  style.textContent = `
    #anmo-toast { position: fixed; top: calc(12px + env(safe-area-inset-top)); left: 0; right: 0; display: flex; flex-direction: column; align-items: center; gap: 6px; z-index: 9999; pointer-events: none; }
    #anmo-toast .anmo-toast { background: rgba(34,30,27,.88); color: #fff; padding: 10px 16px; border-radius: 999px; font-size: 13px; font-weight: 500; max-width: 86vw; box-shadow: 0 12px 40px rgba(34,30,27,.18); animation: anmo-toast-in .22s cubic-bezier(.22,.61,.36,1) both; }
  `
  document.head.appendChild(style)
}

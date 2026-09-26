/**
 * platform/notify — user-facing notifications (DOM belongs to platform).
 */
export function notify(message: string): void {
  // 简单 toast：复用全局容器
  let host = document.getElementById('anmo-toast')
  if (!host) {
    host = document.createElement('div')
    host.id = 'anmo-toast'
    document.body.appendChild(host)
  }
  const tip = document.createElement('div')
  tip.className = 'toast'
  tip.textContent = message
  host.appendChild(tip)
  window.setTimeout(() => tip.remove(), 2600)
}

/** 在 index.html 挂载 toast 宿主样式（main.ts 启动时调用一次）。 */
export function registerGlobalStyles(): void {
  const style = document.createElement('style')
  style.textContent = `
    #anmo-toast { position: fixed; top: 12px; left: 0; right: 0; display: flex; flex-direction: column; align-items: center; gap: 6px; z-index: 9999; pointer-events: none; }
    #anmo-toast .toast { background: rgba(0,0,0,.78); color: #fff; padding: 8px 14px; border-radius: 8px; font-size: 14px; max-width: 80vw; }
  `
  document.head.appendChild(style)
}

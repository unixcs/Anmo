// core — 展示格式化与状态文案（金额一律整数分存储，仅展示层转元）

export function yuan(fen: number | null | undefined): string {
  if (fen === null || fen === undefined) return '-'
  return `¥${(fen / 100).toFixed(2).replace(/\.00$/, '')}`
}

export function toFen(yuanStr: string): number {
  return Math.round(Number(yuanStr || '0') * 100)
}

export function fmtTime(rfc3339: string | null | undefined): string {
  if (!rfc3339) return '-'
  const d = new Date(rfc3339)
  if (Number.isNaN(d.getTime())) return rfc3339
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

export function fmtDate(rfc3339: string | null | undefined): string {
  if (!rfc3339) return '-'
  return fmtTime(rfc3339).slice(0, 10)
}

export function todayStr(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

export const APT_STATUS_TEXT: Record<string, string> = {
  PENDING_CONFIRM: '待确认',
  CONFIRMED: '已确认',
  IN_SERVICE: '服务中',
  COMPLETED: '已完成',
  CANCELLED: '已取消',
  NO_SHOW: '未到店',
}

export const APT_STATUS_TAG: Record<string, 'info' | 'primary' | 'warning' | 'success' | 'danger'> = {
  PENDING_CONFIRM: 'warning',
  CONFIRMED: 'primary',
  IN_SERVICE: 'success',
  COMPLETED: 'info',
  CANCELLED: 'danger',
  NO_SHOW: 'danger',
}

export const PAY_METHOD_TEXT: Record<string, string> = {
  CARD: '次卡核销',
  WECHAT_TRANSFER: '微信转账',
  CASH: '现金',
  OTHER: '其他',
}

export const CARD_STATUS_TEXT: Record<string, string> = {
  ACTIVE: '有效',
  USED_UP: '已用完',
  EXPIRED: '已过期',
  CANCELLED: '已作废',
}

export const CARD_TX_TYPE_TEXT: Record<string, string> = {
  ISSUE: '开卡',
  REDEEM: '核销',
  ADJUSTMENT: '手工调整',
  REVERSAL: '撤销恢复',
}

export const CARD_TYPE_TEXT: Record<string, string> = {
  COUNT: '次卡',
  ACTIVITY: '活动卡',
}

export const VALIDITY_TEXT: Record<string, string> = {
  PERMANENT: '永久有效',
  FIXED: '固定期限',
}

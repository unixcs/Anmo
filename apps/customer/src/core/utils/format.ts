/**
 * core/utils — pure formatting helpers.
 */
export function yuan(cents: number): string {
  return (cents / 100).toFixed(cents % 100 === 0 ? 0 : 2)
}

export function statusText(status: string): string {
  const map: Record<string, string> = {
    PENDING_CONFIRM: '待确认',
    CONFIRMED: '已确认',
    IN_SERVICE: '服务中',
    COMPLETED: '已完成',
    CANCELLED: '已取消',
    NO_SHOW: '爽约',
    ACTIVE: '有效',
    USED_UP: '已用完',
    EXPIRED: '已过期',
  }
  return map[status] ?? status
}

export function hhmm(iso: string): string {
  return iso.slice(11, 16)
}

export function dateOnly(iso: string): string {
  return iso.slice(0, 10)
}

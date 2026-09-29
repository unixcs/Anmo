/**
 * core/utils — pure formatting helpers.
 */
export function yuan(cents: number): string {
  return (cents / 100).toFixed(cents % 100 === 0 ? 0 : 2)
}

export function statusText(status: string): string {
  const map: Record<string, string> = {
    WAITING: '待到店',
    // 状态机收紧前的历史值（旧状态日志展示）
    PENDING_CONFIRM: '待确认(历史)',
    CONFIRMED: '已确认(历史)',
    IN_SERVICE: '服务中',
    COMPLETED: '已完成',
    CANCELLED: '已取消',
    NO_SHOW: '未到店',
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

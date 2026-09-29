/**
 * core/logic — booking business rules. Pure functions, no browser APIs (§102).
 * 业务日期一律锚定北京时间（后端固定 Asia/Shanghai）：Date 先 +8h 偏移，
 * 再用 UTC 分量读取，设备时区不参与口径。
 */
export interface DayOption {
  label: string // "09-28 周一"
  value: string // "2026-09-28"
}

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

export interface BusinessHours {
  open: string // "09:00"
  close: string // "21:00"
  slotMinutes: number
  bookAheadDays: number
  minAheadHours: number
}

export const DEFAULT_HOURS: BusinessHours = {
  open: '09:00',
  close: '21:00',
  slotMinutes: 30,
  bookAheadDays: 30,
  minAheadHours: 2,
}

const BJ_MS = 8 * 3_600_000
const DAY_MS = 86_400_000

/** 当前北京时间映射的偏移 Date：分量一律用 getUTC* 读取。 */
function bjNow(): Date {
  return new Date(Date.now() + BJ_MS)
}

function bjDateStr(d: Date): string {
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0')
  const dd = String(d.getUTCDate()).padStart(2, '0')
  return `${d.getUTCFullYear()}-${mm}-${dd}`
}

/** 今天（北京）YYYY-MM-DD。 */
export function todayStr(): string {
  return bjDateStr(bjNow())
}

/** 'YYYY-MM-DD' 的星期（0=周日），由日期串直接计算、不经时区往返。 */
function weekdayOf(s: string): number {
  const [y, m, d] = s.split('-').map((x) => parseInt(x, 10))
  return (Math.floor(Date.UTC(y, m - 1, d) / DAY_MS) + 4) % 7
}

/** 候选日期（今天起 bookAheadDays 天内）。 */
export function candidateDays(hours: BusinessHours = DEFAULT_HOURS): DayOption[] {
  const bj = bjNow()
  const out: DayOption[] = []
  for (let i = 0; i < hours.bookAheadDays; i++) {
    const value = bjDateStr(new Date(bj.getTime() + i * DAY_MS))
    out.push({ label: `${value.slice(5, 10)} ${WEEKDAYS[weekdayOf(value)]}`, value })
  }
  return out
}

/** 半天当前是否可选（未闭店、未结束、池有余量）。 */
export function partOpen(
  half: { closed: boolean; remaining: number; ended?: boolean } | null | undefined,
): boolean {
  return !!half && !half.closed && !half.ended && half.remaining > 0
}

/** 半天选择卡片上的状态文案（与小程序 slot-picker 同口径）。 */
export function partMeta(half: { closed: boolean; remaining: number; ended?: boolean } | null | undefined): string {
  if (!half || half.closed) return '已闭店'
  if (half.ended) return '时段已过'
  if (half.remaining < 1) return '已约满'
  return `剩 ${half.remaining} 个名额`
}

/**
 * 后端 booking-options 下发当天全部时段槽（不按当前时刻裁剪）。
 * 客户端裁掉"已经开始"的槽；整段半天都过去则标记 ended（与小程序 format.trimPastSlots 同口径）。
 */
export function trimPastSlots(
  options: { am: { closed: boolean; remaining: number; ended?: boolean; slots: { time: string }[] | null } ; pm: { closed: boolean; remaining: number; ended?: boolean; slots: { time: string }[] | null } } | null | undefined,
  date: string,
): void {
  if (!options || !date) return
  const bj = bjNow()
  if (bjDateStr(bj) !== date) return
  const hm = `${String(bj.getUTCHours()).padStart(2, '0')}:${String(bj.getUTCMinutes()).padStart(2, '0')}`
  for (const half of [options.am, options.pm]) {
    if (!half || half.closed || !half.slots?.length) continue
    const keep = half.slots.filter((s) => s.time > hm)
    if (!keep.length) half.ended = true
    half.slots = keep
  }
}

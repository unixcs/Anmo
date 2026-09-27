/**
 * core/logic — booking business rules: slot grid generation and selection
 * validation. Pure functions, no browser APIs (§102).
 */
export interface SlotOption {
  label: string // "15:00"
  startTime: string // "YYYY-MM-DD HH:MM"
}

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

function minutesOf(hhmm: string): number {
  const [h, m] = hhmm.split(':').map((x) => parseInt(x, 10))
  return h * 60 + m
}

/** 候选日期（今天起 bookAheadDays 天内）。 */
export function candidateDays(hours: BusinessHours = DEFAULT_HOURS): DayOption[] {
  const out: DayOption[] = []
  const now = new Date()
  for (let i = 0; i < hours.bookAheadDays; i++) {
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() + i)
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    out.push({
      label: `${mm}-${dd} ${WEEKDAYS[d.getDay()]}`,
      value: `${d.getFullYear()}-${mm}-${dd}`,
    })
  }
  return out
}

/** 某一天的可选开始时间槽（30 分钟网格，营业时间内）。 */
export function slotsForDay(
  day: string,
  durationMinutes: number,
  hours: BusinessHours,
): SlotOption[] {
  const out: SlotOption[] = []
  const openM = minutesOf(hours.open)
  const closeM = minutesOf(hours.close)
  const step = hours.slotMinutes
  const now = new Date()
  const nowM = now.getHours() * 60 + now.getMinutes()
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`

  for (let m = openM; m + durationMinutes <= closeM; m += step) {
    // 提前量：今天的时间槽需满足最晚提前 N 小时
    if (day === today && m < nowM + hours.minAheadHours * 60) continue
    const hh = String(Math.floor(m / 60)).padStart(2, '0')
    const mm = String(m % 60).padStart(2, '0')
    out.push({ label: `${hh}:${mm}`, startTime: `${day} ${hh}:${mm}` })
  }
  return out
}

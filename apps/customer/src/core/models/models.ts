/**
 * core/models — API shapes shared across pages (no browser APIs here).
 */
export interface ServiceCategory {
  id: string
  name: string
  sort: number
  status: string
}

export interface ServiceItem {
  id: string
  category_id: string
  name: string
  description: string
  duration_minutes: number
  default_price: number // 整数分
  cover_image: string
  status: string
  sort: number
}

export interface Member {
  id: string
  member_no: string
  name: string
  phone: string
  gender: string
  birthday: string | null
  status: string
  remark: string
  last_visit_at: string | null
}

export interface MemberCard {
  id: string
  member_id: string
  card_template_id: string
  total_count: number
  remaining_count: number
  valid_from: string
  valid_until: string | null
  status: string
}

export interface Appointment {
  id: string
  appointment_no: string
  member_id: string
  scheduled_start: string
  scheduled_end: string
  status: string
  customer_note: string
  // 注意：internal_note 后台内部备注不下发（§107）
  confirmed_at: string | null
  started_at: string | null
  completed_at: string | null
  cancelled_at: string | null
  created_at: string
}

export interface Page<T> {
  data?: T[]
  total: number
  page: number
  per_page: number
}

export interface Envelope<T> {
  code?: string
  msg?: string
  data?: T
}

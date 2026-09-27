/**
 * core/api — endpoint catalog + typed wrappers.
 */
import { http } from './http'
import type { Appointment, Member, MemberCard, ServiceCategory, ServiceItem } from '../models/models'

export const endpoints = {
  sendSms: '/api/auth/sms/send',
  verifySms: '/api/auth/sms/verify',
  home: '/api/home',
  publicSettings: '/api/settings',
  catalog: '/api/services',
  myProfile: '/api/me/profile',
  myCards: '/api/me/cards',
  myCardTransactions: (id: string) => `/api/me/cards/${id}/transactions`,
  appointments: '/api/appointments',
  appointment: (id: string) => `/api/appointments/${id}`,
  cancel: (id: string) => `/api/appointments/${id}/cancel`,
  reschedule: (id: string) => `/api/appointments/${id}/reschedule`,
}

export interface CardTx {
  id: string
  member_card_id: string
  card_name: string
  type: string
  quantity: number
  before_count: number
  after_count: number
  reference_type: string
  remark: string
  created_at: string
}

export interface Catalog {
  categories: ServiceCategory[]
  services: ServiceItem[]
}

export interface HomeBanner {
  id: string
  title: string
  image: string
  link: string
}

export interface HomeAnnouncement {
  id: string
  title: string
  content: string
}

export interface HomeBlock {
  type: string
  data: Record<string, unknown>
}

export interface HomeContent {
  blocks: HomeBlock[]
  banners: HomeBanner[]
  announcements: HomeAnnouncement[]
}

/** 门店当前状态（§17：动态计算，不落库） */
export interface StoreStatus {
  status: 'FREE' | 'SERVING' | 'BUSY'
  free_at?: string
  open_time: string
  close_time: string
}

export const api = {
  sendSms: (phone: string) => http.post<{ sent: boolean }>(endpoints.sendSms, { phone }),
  verifySms: (phone: string, code: string) =>
    http.post<{ token: string; member_id: string }>(endpoints.verifySms, { phone, code }),
  home: () => http.get<HomeContent>(endpoints.home),
  publicSettings: () => http.get<Record<string, string>>(endpoints.publicSettings),
  storeStatus: () => http.get<StoreStatus>('/api/store/status'),
  catalog: () => http.get<Catalog>(endpoints.catalog),
  myProfile: () => http.get<{ member: Member; tags: { id: string; name: string }[] }>(endpoints.myProfile),
  updateProfile: (patch: Partial<Pick<Member, 'name' | 'gender' | 'birthday'>>) =>
    http.put<{ member: Member }>(endpoints.myProfile, patch),
  myCards: () => http.get<MemberCard[]>(endpoints.myCards),
  // §14 使用明细：开卡/核销/调整/撤销全量流水（含无预约的散客核销）
  myCardTransactions: (cardId: string) => http.get<CardTx[]>(endpoints.myCardTransactions(cardId)),
  // 注意：http 已解包包络，分页端点返回的就是 data 数组本身
  myAppointments: (status: string) =>
    http.get<Appointment[]>(`${endpoints.appointments}?status=${status}&page=1&per_page=50`),
  appointment: (id: string) =>
    http.get<{ appointment: Appointment; services: { service_name_snapshot: string; duration_minutes_snapshot: number; price_snapshot: number }[] }>(endpoints.appointment(id)),
  bookingOptions: (date: string) =>
    http.get<BookingOptions>(`/api/booking-options?date=${date}`),
  createAppointment: (serviceId: string, target: BookingTarget, note: string) =>
    http.post<Appointment>(endpoints.appointments, {
      service_id: serviceId,
      ...target,
      note,
    }),
  cancelAppointment: (id: string) => http.put<Appointment>(endpoints.cancel(id)),
  rescheduleAppointment: (id: string, target: BookingTarget) =>
    http.put<Appointment>(endpoints.reschedule(id), target),
}

/** 具体时间或模糊半天，二选一（D20）。 */
export interface BookingTarget {
  start_time?: string // "YYYY-MM-DD HH:MM"
  date?: string // "YYYY-MM-DD"（配 day_part）
  day_part?: 'AM' | 'PM'
}

export interface BookingSlot {
  time: string
  remaining: number
}

export interface BookingHalfDay {
  closed: boolean
  total: number
  remaining: number
  slots: BookingSlot[] | null
}

export interface BookingOptions {
  date: string
  open: boolean
  am: BookingHalfDay
  pm: BookingHalfDay
}

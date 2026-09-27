// core/api — Admin 端点封装。字段名与后端 struct json tag 严格一致
// （请求体启用 DisallowUnknownFields，多传字段会被 400 拒绝）。
import { http, idemKey } from './http'

// ---------- 模型 ----------

export interface AdminUser {
  id: string
  name: string
  role: string
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
  created_at: string
  updated_at: string
}

export interface Tag {
  id: string
  name: string
}

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
  default_price: number
  cover_image: string
  status: string
  sort: number
}

export interface CardTemplate {
  id: string
  name: string
  type: 'COUNT' | 'ACTIVITY'
  total_count: number
  validity_type: 'PERMANENT' | 'FIXED'
  valid_from: string | null
  valid_until: string | null
  price: number
  status: string
  service_ids?: string[]
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
  issued_at: string
}

export interface CardTransaction {
  id: string
  member_card_id: string
  type: 'ISSUE' | 'REDEEM' | 'ADJUSTMENT' | 'REVERSAL'
  quantity: number
  before_count: number
  after_count: number
  reference_type: string
  remark: string
  created_at: string
}

export type AppointmentStatus =
  | 'PENDING_CONFIRM'
  | 'CONFIRMED'
  | 'IN_SERVICE'
  | 'COMPLETED'
  | 'CANCELLED'
  | 'NO_SHOW'

export interface Appointment {
  id: string
  appointment_no: string
  member_id: string
  scheduled_start: string
  scheduled_end: string
  status: AppointmentStatus
  slot_type: 'SPECIFIC' | 'HALF_DAY'
  day_part?: 'AM' | 'PM'
  customer_note: string
  internal_note: string
  confirmed_at: string | null
  started_at: string | null
  completed_at: string | null
  cancelled_at: string | null
  created_at: string
}

export interface AppointmentService {
  id: string
  appointment_id: string
  service_id: string
  service_name_snapshot: string
  duration_minutes_snapshot: number
  price_snapshot: number
  quantity: number
}

export interface TodaySummary {
  date: string
  total: number
  pending_confirm: number
  confirmed: number
  in_service: number
  completed: number
  cancelled: number
  no_show: number
}

export interface TodayAppointment extends Appointment {
  service: AppointmentService | null
}

export interface Payment {
  id: string
  appointment_id: string | null // null = 散客核销（D19）
  member_id: string
  amount: number
  method: 'CARD' | 'WECHAT_TRANSFER' | 'CASH' | 'OTHER'
  status: 'VALID' | 'VOIDED'
  reference_no: string
  remark: string
}

export interface Redemption {
  id: string
  appointment_id: string | null // null = 散客核销（D19）
  member_id: string
  member_card_id: string
  service_id: string
  quantity: number
  before_count: number
  after_count: number
  status: 'SUCCESS' | 'REVERSED'
  idempotency_key: string
}

export interface WorkbenchCard {
  appointment: Appointment
  service: AppointmentService | null
  member_name: string
  payment: Payment | null
}

export interface Banner {
  id: string
  title: string
  image: string
  link: string
  sort: number
  status: string
}

export interface Announcement {
  id: string
  title: string
  content: string
  sort: number
  status: string
}

export interface PageBlock {
  type: string
  data: Record<string, unknown>
}

export interface InsightItem {
  kind: string
  member_id: string
  member_no: string
  member_name: string
  card_id?: string
  detail: string
}

export interface LoggedOperation {
  id: string
  actor_type: string
  actor_id: string
  action: string
  target_type: string
  target_id: string
  detail: string
  ip: string
}

// ---------- 认证 ----------

export function login(phone: string, password: string) {
  return http.post<{ token: string; user: AdminUser }>('/admin/auth/login', { phone, password })
}

// ---------- 会员 / 标签 ----------

export function listMembers(keyword: string, page: number, perPage: number) {
  const q = new URLSearchParams({ page: String(page), per_page: String(perPage) })
  if (keyword) q.set('keyword', keyword)
  return http.getPage<Member>(`/admin/members?${q}`)
}

export function createMember(body: {
  name: string
  phone: string
  gender?: string
  birthday?: string | null
  remark?: string
}) {
  return http.post<Member>('/admin/members', body)
}

export function getMember(id: string) {
  return http.get<{ member: Member; tags: Tag[] }>(`/admin/members/${id}`)
}

export function updateMember(
  id: string,
  body: { name?: string; gender?: string; birthday?: string | null; remark?: string },
) {
  return http.put<Member>(`/admin/members/${id}`, body)
}

export function setMemberTags(id: string, tagIds: string[]) {
  return http.put<Tag[]>(`/admin/members/${id}/tags`, { tag_ids: tagIds })
}

export function listTags() {
  return http.get<Tag[]>('/admin/tags')
}

export function createTag(name: string) {
  return http.post<Tag>('/admin/tags', { name })
}

// ---------- 服务分类 / 项目 ----------

export function listCategories() {
  return http.get<ServiceCategory[]>('/admin/service-categories')
}

export function createCategory(name: string, sort: number) {
  return http.post<ServiceCategory>('/admin/service-categories', { name, sort })
}

export function setCategoryStatus(id: string, status: 'ACTIVE' | 'INACTIVE') {
  return http.put<{ updated: boolean }>(`/admin/service-categories/${id}/status`, { status })
}

export function listServices() {
  return http.get<ServiceItem[]>('/admin/services')
}

export function createService(body: Omit<ServiceItem, 'id' | 'status'>) {
  return http.post<ServiceItem>('/admin/services', body)
}

export function updateService(id: string, body: Omit<ServiceItem, 'id' | 'status'>) {
  return http.put<ServiceItem>(`/admin/services/${id}`, body)
}

export function setServiceStatus(id: string, status: 'ACTIVE' | 'INACTIVE') {
  return http.put<{ updated: boolean }>(`/admin/services/${id}/status`, { status })
}

// ---------- 卡模板 / 会员卡 ----------

export function listCardTemplates() {
  return http.get<CardTemplate[]>('/admin/card-templates')
}

export function createCardTemplate(body: {
  name: string
  type: 'COUNT' | 'ACTIVITY'
  total_count: number
  validity_type: 'PERMANENT' | 'FIXED'
  valid_from?: string | null
  valid_until?: string | null
  price: number
}) {
  return http.post<CardTemplate>('/admin/card-templates', body)
}

export function setCardTemplateStatus(id: string, status: 'ACTIVE' | 'INACTIVE') {
  return http.put<{ updated: boolean }>(`/admin/card-templates/${id}/status`, { status })
}

export function setCardTemplateRules(id: string, serviceIds: string[]) {
  return http.put<{ updated: boolean }>(`/admin/card-templates/${id}/rules`, {
    service_ids: serviceIds,
  })
}

export function issueCard(memberId: string, cardTemplateId: string) {
  return http.post<MemberCard>('/admin/cards', {
    member_id: memberId,
    card_template_id: cardTemplateId,
  })
}

export function listMemberCards(memberId: string) {
  return http.get<MemberCard[]>(`/admin/members/${memberId}/cards`)
}

export function adjustCard(id: string, delta: number, remark: string) {
  return http.put<MemberCard>(`/admin/cards/${id}/adjust`, { delta, remark })
}

export function cancelCard(id: string) {
  return http.put<{ cancelled: boolean }>(`/admin/cards/${id}/cancel`)
}

export function listCardTransactions(id: string) {
  return http.get<CardTransaction[]>(`/admin/cards/${id}/transactions`)
}

// ---------- 预约 ----------

export function listAppointments(query: { status?: string; date?: string; page: number; per_page: number }) {
  const q = new URLSearchParams({ page: String(query.page), per_page: String(query.per_page) })
  if (query.status) q.set('status', query.status)
  if (query.date) q.set('date', query.date)
  return http.getPage<Appointment>(`/admin/appointments?${q}`)
}

export function getToday(date?: string) {
  const q = date ? `?date=${date}` : ''
  return http.get<{ summary: TodaySummary; appointments: TodayAppointment[] }>(`/admin/today${q}`)
}

export function confirmAppointment(id: string) {
  return http.put<Appointment>(`/admin/appointments/${id}/confirm`)
}

export function startAppointment(id: string) {
  return http.put<Appointment>(`/admin/appointments/${id}/start`)
}

export function completeAppointment(id: string) {
  return http.put<Appointment>(`/admin/appointments/${id}/complete`)
}

export function cancelAppointment(id: string, reason: string) {
  return http.put<Appointment>(`/admin/appointments/${id}/cancel`, { reason })
}

export function noShowAppointment(id: string) {
  return http.put<Appointment>(`/admin/appointments/${id}/no-show`)
}

/** 改期目标：具体时间或模糊半天，二选一（D20）。 */
export interface BookingTarget {
  start_time?: string
  date?: string
  day_part?: 'AM' | 'PM'
}

export function rescheduleAppointment(id: string, target: BookingTarget) {
  return http.put<Appointment>(`/admin/appointments/${id}/reschedule`, target)
}

// ---------- 可约时段 / 闭店（D20/D22） ----------

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

export function getBookingOptions(date: string) {
  return http.get<BookingOptions>(`/admin/booking-options?date=${date}`)
}

export interface Closure {
  id: string
  closure_date: string
  day_part: 'AM' | 'PM'
  remark: string
  created_at: string
}

export function listClosures(from = '') {
  return http.get<Closure[]>(`/admin/closures${from ? `?from=${from}` : ''}`)
}

export function createClosure(date: string, dayPart: 'AM' | 'PM' | 'FULL', remark: string) {
  return http.post<{ saved: boolean; conflict_count: number }>('/admin/closures', {
    date,
    day_part: dayPart,
    remark,
  })
}

export function deleteClosure(id: string) {
  return http.delete<{ deleted: boolean }>(`/admin/closures/${id}`)
}

// ---------- 结算 / 收款 / 撤销 ----------

export function redeemCard(appointmentId: string, cardId: string) {
  return http.post<{ redemption: Redemption; payment: Payment }>(
    `/admin/appointments/${appointmentId}/redeem`,
    { card_id: cardId, idempotency_key: idemKey() },
  )
}

/** 散客核销（D19）：无预约，按卡直接扣次，需指定服务（规则校验+金额）。 */
export function redeemWalkIn(cardId: string, serviceId: string) {
  return http.post<{ redemption: Redemption; payment: Payment }>(
    `/admin/cards/${cardId}/redeem`,
    { service_id: serviceId, idempotency_key: idemKey() },
  )
}

export function settlePayment(
  appointmentId: string,
  body: {
    method: 'WECHAT_TRANSFER' | 'CASH' | 'OTHER'
    amount: number
    reference_no?: string
    remark?: string
  },
) {
  return http.post<Payment>(`/admin/appointments/${appointmentId}/payments`, {
    ...body,
    idempotency_key: idemKey(),
  })
}

export function reverseRedemption(id: string, reason: string) {
  return http.put<{ reversed: boolean }>(`/admin/redemptions/${id}/reverse`, { reason })
}

export function getWorkbench(date?: string) {
  const q = date ? `?date=${date}` : ''
  return http.get<{ summary: TodaySummary; cards: WorkbenchCard[] }>(`/admin/workbench${q}`)
}

export function listPayments(status?: string) {
  const q = status ? `?status=${status}` : ''
  return http.get<Payment[]>(`/admin/payments${q}`)
}

export function listRedemptions(status?: string) {
  const q = status ? `?status=${status}` : ''
  return http.get<Redemption[]>(`/admin/redemptions${q}`)
}

// ---------- 内容 / 设置 ----------

export function getPageConfig(page: string) {
  return http.get<PageBlock[]>(`/admin/content/pages?page=${page}`)
}

export function savePageConfig(page: string, blocks: PageBlock[]) {
  return http.put<{ saved: boolean }>('/admin/content/pages', { page, blocks })
}

export function listBanners() {
  return http.get<Banner[]>('/admin/banners')
}

export function saveBanner(banner: Partial<Banner> & { title: string; image: string }) {
  return http.post<Banner>('/admin/banners', banner)
}

export function setBannerStatus(id: string, status: 'ACTIVE' | 'INACTIVE') {
  return http.put<{ updated: boolean }>(`/admin/banners/${id}/status`, { status })
}

export function listAnnouncements() {
  return http.get<Announcement[]>('/admin/announcements')
}

export function saveAnnouncement(a: Partial<Announcement> & { title: string; content: string }) {
  return http.post<Announcement>('/admin/announcements', a)
}

export function setAnnouncementStatus(id: string, status: 'ACTIVE' | 'INACTIVE') {
  return http.put<{ updated: boolean }>(`/admin/announcements/${id}/status`, { status })
}

export function getSettings() {
  return http.get<Record<string, string>>('/admin/settings')
}

export function saveSetting(key: string, value: string) {
  return http.put<{ saved: boolean }>('/admin/settings', { key, value })
}

// ---------- 运维 ----------

export function getInsights() {
  return http.get<Record<'LOW_BALANCE' | 'DORMANT' | 'EXPIRING', InsightItem[]>>('/admin/insights')
}

export function listLogs(action?: string) {
  const q = action ? `?action=${encodeURIComponent(action)}` : ''
  return http.get<LoggedOperation[]>(`/admin/logs${q}`)
}

export function runDailyOps() {
  return http.post<{ expired_swept: number; snapshots: number }>('/admin/ops/daily')
}
